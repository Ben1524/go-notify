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
