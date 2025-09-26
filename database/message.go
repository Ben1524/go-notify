package database

import (
	"github.com/jinzhu/gorm"
	"go-notify/model"
)

// GetMessageByID returns the messages for the given id or nil.
func (d *GormDatabase) GetMessageByID(id uint) (*model.Message, error) {
	msg := new(model.Message)
	err := d.DB.Find(msg, id).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if msg.ID == id {
		return msg, err
	}
	return nil, err
}

// CreateMessage creates a message.
func (d *GormDatabase) CreateMessage(message *model.Message) error {
	return d.DB.Create(message).Error
}

// GetMessagesByUser returns all messages from a user.
func (d *GormDatabase) GetMessagesByUser(userID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Joins("JOIN applications ON applications.user_id = ?", userID).
		Where("messages.application_id = applications.id").Order("id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByUserSince returns limited messages from a user.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByUserSince(userID uint, limit int, since uint) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.
		// 左连接applications表，确保application_id=0的消息也能保留
		Joins("LEFT JOIN applications ON messages.application_id = applications.id").
		// 左连接中间表app_users，关联用户与应用（条件：应用属于当前用户）
		Joins("LEFT JOIN app_users ON applications.id = app_users.app_id AND app_users.user_id = ?", userID).
		// 核心条件：要么是用户关联的应用消息，要么是application_id=0的系统消息
		Where("(app_users.user_id = ? AND app_users.deleted_at IS NULL)", userID)

	// 处理since参数：如果since>0，只查询ID大于since的消息（获取更新的消息）
	if since > 0 {
		db = db.Where("messages.id > ?", since)
	}
	// 执行查询：按ID降序（最新的在前），限制数量
	result := db.Order("messages.id DESC").Limit(limit).Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	return messages, nil
}

// GetMessagesByApplication returns all messages from an application.
func (d *GormDatabase) GetMessagesByApplication(tokenID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Where("application_id = ?", tokenID).Order("id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplicationSince returns limited messages from an application.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.Where("application_id = ?", appID).Order("id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// DeleteMessageByID deletes a message by its id.
func (d *GormDatabase) DeleteMessageByID(id []uint) error {
	return d.DB.Where("id IN (?)", id).Delete(&model.Message{}).Error
}

// DeleteMessagesByApplication deletes all messages from an application.
func (d *GormDatabase) DeleteMessagesByApplication(applicationID uint) error {
	return d.DB.Where("application_id = ?", applicationID).Delete(&model.Message{}).Error
}

// DeleteMessagesByUser deletes all messages from a user.
func (d *GormDatabase) DeleteMessagesByUser(userID uint) error {
	app, _ := d.GetApplicationsByUser(userID)
	for _, app := range app {
		d.DeleteMessagesByApplication(app.ID)
	}
	return nil
}

func (d *GormDatabase) GetBroadcastMessage(limit int) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.Table("messages").Where("application_id = 1").Order("id desc").Limit(limit)
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}
