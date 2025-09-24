package service

import (
	"go-notify/database"
	"testing"
)

func TestBroadcastMessage(t *testing.T) {
	db, err := database.NewGormDatabase("D:\\GolandProjects\\go-notify\\data\\go-notify.db")
	if err != nil {
		t.Error(err)
		return
	}
	messages, err := db.GetBroadcastMessage(10)
	if err != nil {
		t.Error(err)
	}
	t.Log(messages[0].Message)
}
