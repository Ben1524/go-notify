package database

import (
	"go-notify/model"
	"testing"
)

func TestMessageGetMessagesByUserSince(t *testing.T) {
	db, _ := NewGormDatabase("data/go-notify.db")
	defer db.Close()
	db.CreateUser(&model.User{
		Name:  "test",
		Pass:  []byte("test"),
		Admin: false,
	})

	db.CreateApplication(&model.Application{
		Name:        "system",
		Token:       "testtoken",
		Description: "系统应用",
		Internal:    true,
	})
}
