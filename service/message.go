package service

import (
	"github.com/gin-gonic/gin"
	"go-notify/auth"
	"go-notify/model"
	"log"
	"time"
)

type MessageDatabaseService interface {
	GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetMessagesByUserSince(userID uint, limit int, since uint) ([]*model.Message, error)
	GetBroadcastMessage(limit int) ([]*model.Message, error)
	DeleteMessageByID(id uint) error
	GetMessageByID(id uint) (*model.Message, error)
	DeleteMessagesByUser(userID uint) error
	DeleteMessagesByApplication(applicationID uint) error
	CreateMessage(message *model.Message) error
	GetApplicationByToken(token string) (*model.Application, error)
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

// 获取指定用户ID的所有消息
// 用户可能关注了不同的板块(应用程序)，需要返回所有板块的消息，包含系统信息
func (mess *MessageService) GetMessages(ctx *gin.Context) {
	userID := auth.TryGetUserID(ctx)
	withPaging(ctx, func(params *pagingParams) {
		messages, err := mess.DB.GetMessagesByUserSince(userID, params.Limit+1, params.Since)
		boradcastMessage, err := mess.DB.GetBroadcastMessage(params.Limit + 1)
		ctx.JSON(http.StatusOk)
	})
}
