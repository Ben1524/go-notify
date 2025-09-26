package service

import (
	"errors"
	json "github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"go-notify/auth"
	"go-notify/model"
	"log"
	"math/bits"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type MessageDatabaseService interface {
	GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetMessagesByUserSince(userID uint, limit int, since uint) ([]*model.Message, error)
	GetBroadcastMessage(limit int) ([]*model.Message, error)
	DeleteMessageByID(id []uint) error
	GetMessageByID(id uint) (*model.Message, error)
	DeleteMessagesByUser(userID uint) error
	DeleteMessagesByApplication(applicationID uint) error
	CreateMessage(message *model.Message) error
	GetApplicationByToken(token string) (*model.Application, error)
	JudgeUserOwnsApplication(userID, appID uint) (bool, error)
	IsUserAlloweOpMessage(userID uint, msgID []uint) (bool, error)
}

var timeNow = time.Now

type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
	BroadcastNotify(message *model.MessageExternal) // 广播通知
}

type MessageService struct {
	DB       MessageDatabaseService
	Notifier Notifier
}

type pagingParams struct {
	Limit int  `form:"limit" binding:"min=1,max=200"`
	Since uint `form:"since" binding:"min=0"`
}

// 获取分页参数的信息并执行回调函数
func withPaging(ctx *gin.Context, f func(params *pagingParams)) {
	params := &pagingParams{
		Since: 1, // 增加默认页码
		Limit: 100,
	}

	// 使用ShouldBindQuery更合适，不会自动返回错误响应
	if err := ctx.ShouldBindQuery(params); err != nil {
		log.Printf("Failed to bind paging parameters: %v", err)
		return
	}

	// 可以添加参数校验和修正逻辑
	if params.Limit > 1000 { // 限制最大条数
		params.Limit = 1000
	}

	f(params)
}

func toExternalMessage(msg *model.Message) *model.MessageExternal {
	res := &model.MessageExternal{
		ID:            msg.ID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Priority:      msg.Priority,
		Date:          msg.Date,
	}
	if len(msg.Extras) != 0 {
		res.Extras = make(map[string]interface{})
		json.Unmarshal(msg.Extras, &res.Extras)
	}
	return res
}

func toExternalMessages(msg []*model.Message) []*model.MessageExternal {
	res := make([]*model.MessageExternal, len(msg))
	for i := range msg {
		res[i] = toExternalMessage(msg[i])
	}
	return res
}
func buildWithPaging(ctx *gin.Context, paging *pagingParams, messages []*model.Message) *model.PagedMessages {
	next := ""
	since := uint(0)
	useMessages := messages
	if len(messages) > paging.Limit {
		useMessages = messages[:len(messages)-1]
		since = useMessages[len(useMessages)-1].ID
		url := Get(ctx)
		url.Path = ctx.Request.URL.Path
		query := url.Query()
		query.Add("limit", strconv.Itoa(paging.Limit))
		query.Add("since", strconv.FormatUint(uint64(since), 10))
		url.RawQuery = query.Encode()
		next = url.String()
	}
	return &model.PagedMessages{
		Paging:   model.Paging{Size: len(useMessages), Limit: paging.Limit, Next: next, Since: since},
		Messages: toExternalMessages(useMessages),
	}
}

// 获取指定用户ID的所有消息
// 用户可能关注了不同的板块(应用程序)，需要返回所有板块的消息，包含系统信息
func (mess *MessageService) GetMessages(ctx *gin.Context) {
	userID := auth.TryGetUserID(ctx)
	withPaging(ctx, func(params *pagingParams) {
		messages, err := mess.DB.GetMessagesByUserSince(userID, params.Limit+1, params.Since)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
			return
		}
		ctx.JSON(http.StatusOK, buildWithPaging(ctx, params, messages))
	})
}

func withIntegerParam(ctx *gin.Context, param string, f func(id uint)) {
	if id, err := strconv.ParseUint(ctx.Param(param), 10, bits.UintSize); err == nil {
		f(uint(id))
	} else {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid param"))
	}
}

// 需要有用户信息和待求的应用程序ID
func (mess *MessageService) GetMessageWithApplication(ctx *gin.Context) {
	withIntegerParam(ctx, "id", func(id uint) {
		withPaging(ctx, func(params *pagingParams) {
			userID := auth.GetUserID(ctx)
			if res, err := mess.DB.JudgeUserOwnsApplication(userID, id); res == true && err == nil {
				// the +1 is used to check if there are more messages and will be removed on buildWithPaging
				messages, err := mess.DB.GetMessagesByApplicationSince(id, params.Limit+1, params.Since)
				if success := successOrAbort(ctx, 500, err); !success {
					return
				}
				ctx.JSON(200, buildWithPaging(ctx, params, messages))
			} else {
				ctx.AbortWithError(404, errors.New("application does not exist"))
			}
		})
	})
}

func parseUintSlice(strs []string) []uint {
	uints := make([]uint, 0, len(strs))
	for _, s := range strs {
		if id, err := strconv.ParseUint(s, 10, bits.UintSize); err == nil {
			uints = append(uints, uint(id))
		}
	}
	return uints
}

// 允许管理员删除自己的消息，该接口需要提前通过中间件验证管理员身份
func (mess *MessageService) DeleteMessages(ctx *gin.Context) {
	// 获取待删除的所有消息ID
	messIDs := parseUintSlice(ctx.QueryArray("message_ids"))
	if len(messIDs) == 0 {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("no message ids provided"))
		return
	}
	userID := auth.GetUserID(ctx)

	// 查看这些消息是否都属于该用户
	res, err := mess.DB.IsUserAlloweOpMessage(userID, messIDs)
	if err != nil || res == false {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("failed to verify message ownership"))
		return
	}
	// 删除消息
	err = mess.DB.DeleteMessageByID(messIDs)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errors.New("failed to delete messages"))
		return
	}
	ctx.JSON(200, gin.H{"status": "messages deleted"})
}

func toInternalMessage(msg *model.MessageExternal) *model.Message {
	res := &model.Message{
		ID:            msg.ID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Date:          msg.Date,
	}
	if msg.Priority != 0 {
		res.Priority = msg.Priority
	}

	if msg.Extras != nil {
		res.Extras, _ = json.Marshal(msg.Extras)
	}
	return res
}

// 创建消息，创建成功后会通知用户
// 只有管理员能创建，需要通过中间件验证管理员身份
func (mess *MessageService) CreateMessage(ctx *gin.Context) {
	message := model.MessageExternal{}
	if err := ctx.Bind(&message); err == nil {
		application, err := mess.DB.GetApplicationByToken(auth.GetTokenID(ctx))
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		message.ApplicationID = application.ID
		if strings.TrimSpace(message.Title) == "" {
			message.Title = application.Name
		}

		if message.Priority == 0 { // 如果没有指定优先级，则使用应用程序的默认优先级
			message.Priority = application.DefaultPriority
		}

		message.Date = timeNow()
		message.ID = 0 // 随意设置ID，防止客户端指定ID，数据库ID自增
		msgInternal := toInternalMessage(&message)
		if success := successOrAbort(ctx, http.StatusInternalServerError, mess.DB.CreateMessage(msgInternal)); !success {
			return
		}
		mess.Notifier.Notify(auth.GetUserID(ctx), toExternalMessage(msgInternal))
		ctx.JSON(200, toExternalMessage(msgInternal))
	}
}
