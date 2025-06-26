package model

import (
	"danmu-core/generated/dystruct"
)

const TableNameUser = "users"

// User mapped from table <users>
type User struct {
	UserID    uint64 `gorm:"column:user_id;not null;index" json:"user_id"`
	DisplayID string `gorm:"column:display_id;not null" json:"display_id"`
	UserName  string `gorm:"column:user_name;not null" json:"user_name"`
}

// TableName User's table name
func (*User) TableName() string {
	return TableNameUser
}

func NewUser(user *dystruct.Webcast_Data_User) *User {
	return &User{
		UserID:    user.Id,
		DisplayID: user.DisplayId,
		UserName:  user.Nickname,
	}
}

// Insert
func (model *User) Insert() error {
	if err := DB.Create(model).Error; err != nil {
		return err
	}
	return nil
}

// CheckAndInsert 检查用户是否存在且信息是否变化，如果变化则插入新记录
func (model *User) CheckAndInsert() error {
	var existingUser User
	// 查找该用户最新的一条记录
	if err := DB.Where("user_id = ? AND user_name = ? AND display_id = ?", model.UserID, model.UserName, model.DisplayID).
		Order("id DESC").
		First(&existingUser).Error; err == nil {
		return nil
	}
	// 用户不存在或信息有变化，插入新记录
	return model.Insert()
}
