package auth

import (
	"github.com/gin-gonic/gin"
	"go-notify/model"
)

func TryGetUserID(ctx *gin.Context) uint {
	user := ctx.MustGet("user").(*model.User)
	if user == nil {
		userID := ctx.MustGet("userid").(uint)
		return userID
	}

	return user.ID
}

func GetUserID(ctx *gin.Context) uint {
	id := TryGetUserID(ctx)
	if id == 0 {
		panic("token and user may not be null")
	}
	return id
}

func GetTokenID(ctx *gin.Context) string {
	return ctx.MustGet("tokenid").(string)
}
