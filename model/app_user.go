package model

import (
	"time"
)

type AppUser struct {
	AppID    uint       `gorm:"primary_key;foreignKey:AppID;references:Application.ID" json:"appId"` // 显式关联 Application.ID
	UserID   uint       `gorm:"primary_key;foreignKey:UserID;references:User.ID" json:"userId"`      // 显式关联 User.ID
	CreateAt *time.Time `gorm:"autoCreateTime" json:"createAt"`
	DeleteAt *time.Time `gorm:"index" json:"deleteAt,omitempty"`
}

// 可选：自定义表名（如果表名与模型名复数形式不同）
func (AppUser) TableName() string {
	return "app_users"
}
