package model

import (
	"gorm.io/gorm/clause"
)

const TableNameCommonMessage = "common_messages"

// CommonMessage mapped from table <common_messages>
type CommonMessage struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	MessageType    string `gorm:"column:message_type;not null" json:"message_type"`
	RoomID         uint64 `gorm:"column:room_id;not null" json:"room_id"`
	RoomDisplayId  string `gorm:"column:room_display_id;not null" json:"room_display_id"`
	RoomName       string `gorm:"column:room_name;not null" json:"room_name"`
	UserName       string `gorm:"column:user_name;not null" json:"user_name"`
	UserID         uint64 `gorm:"column:user_id;not null" json:"user_id"`
	UserDisplayId  string `gorm:"column:user_display_id;not null" json:"user_display_id"`
	Content        string `gorm:"column:content;not null" json:"content"`
	Timestamp      uint64 `gorm:"column:timestamp;not null" json:"timestamp"`
	FavoriteUserId uint64 `gorm:"column:favorite_user_id;default:0" json:"favorite"`
}

// TableName CommonMessage's table name
func (*CommonMessage) TableName() string {
	return TableNameCommonMessage
}

func (model *CommonMessage) Insert() error {
	if err := DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(model).Error; err != nil {
		return err
	}
	return nil
}

func (model *CommonMessage) BatchInsert(models []*CommonMessage) error {
	if err := DB.Create(models).Error; err != nil {
		return err
	}
	return nil
}
