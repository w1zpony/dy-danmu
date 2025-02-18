package model

import (
	"danmu-core/generated/douyin"
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

func NewUser(user *douyin.User) *User {
	return &User{
		UserID:    user.Id,
		DisplayID: user.DisplayId,
		UserName:  user.NickName,
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
	if err := DB.Where("user_id = ?", model.UserID).
		Order("id DESC").
		First(&existingUser).Error; err == nil {
		// 用户存在，检查信息是否相同
		if existingUser.UserName == model.UserName &&
			existingUser.DisplayID == model.DisplayID {
			// 信息完全相同，直接返回
			return nil
		}
	}
	// 用户不存在或信息有变化，插入新记录
	return model.Insert()
}
