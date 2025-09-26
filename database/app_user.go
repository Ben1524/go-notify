package database

import (
	"github.com/jinzhu/gorm"
	"go-notify/model"
)

func (d *GormDatabase) JudgeUserOwnsApplication(userID, appID uint) (bool, error) {
	var tmpVal model.AppUser
	tmpVal.UserID = 0
	d.DB.Where("app_id = ? AND user_id = ?", appID, userID).First(&tmpVal)
	if tmpVal.UserID != 0 {
		return true, nil
	}
	if d.DB.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return false, d.DB.Error
}

// 判断消息是否能被该用户操作
func (d *GormDatabase) IsUserAlloweOpMessage(userID uint, msgID []uint) (bool, error) {
	// 获取消息所属的应用ID
	var appIDs []uint
	err := d.DB.Table("messages").Where("id IN (?)", msgID).Pluck("application_id", &appIDs).Error
	if err != nil || len(appIDs) == 0 {
		return false, err
	}
	// 判断用户是否拥有这些应用
	for _, appID := range appIDs {
		own, err := d.JudgeUserOwnsApplication(userID, appID)
		if err != nil || !own {
			return false, err
		}
	}
	return true, nil
}
