package model

import (
	"danmu-core/generated/douyin"
	"strings"

	"gorm.io/gorm/clause"
)

const TableNameGiftMessage = "gift_messages"

// GiftMessage mapped from table <gift_messages>
type GiftMessage struct {
	ID              int64  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UserID          uint64 `gorm:"column:user_id;not null" json:"user_id"`                       // User ID who sent the gift
	UserName        string `gorm:"column:user_name;not null" json:"user_name"`                   // User Name
	UserDisplayId   string `gorm:"column:user_display_id;not null" json:"user_display_id"`       // User Display ID
	ToUserID        uint64 `gorm:"column:to_user_id;not null" json:"to_user_id"`                 // To User ID
	ToUserName      string `gorm:"column:to_user_name;not null" json:"to_user_name"`             // To User Name
	ToUserDisplayId string `gorm:"column:to_user_display_id;not null" json:"to_user_display_id"` // To User Display ID
	GiftName        string `gorm:"column:gift_name;not null" json:"gift_name"`                   // Gift ID (could be a foreign key)
	GiftID          int64  `gorm:"column:gift_id;not null" json:"gift_id"`
	RoomID          int64  `gorm:"column:room_id;not null" json:"room_id"`                 // Room ID
	RoomDisplayId   string `gorm:"column:room_display_id;not null" json:"room_display_id"` // Room Display ID
	RoomName        string `gorm:"column:room_name;not null" json:"room_name"`             // Room Name
	Message         string `gorm:"column:message;not null" json:"message"`                 // The gift message
	Timestamp       int64  `gorm:"column:timestamp;not null" json:"timestamp"`
	DiamondCount    uint32 `gorm:"column:diamond_count;not null" json:"diamond_count"`
	Image           string `gorm:"column:image_url" json:"image_url"`
	RepeatEnd       int32  `gorm:"column:repeat_end" json:"repeat_end"`
	ComboCount      string `gorm:"column:combo_count" json:"combo_count"`
}

// TableName GiftMessage's table name
func (*GiftMessage) TableName() string {
	return TableNameGiftMessage
}

func NewGiftMessage(message *douyin.GiftMessage) *GiftMessage {
	model := &GiftMessage{
		UserID:        message.User.Id,
		UserName:      message.User.NickName,
		UserDisplayId: message.User.DisplayId,
		GiftName:      message.Gift.Name,
		RoomID:        message.Common.RoomId,
		Message:       message.Common.Describe,
		Timestamp:     message.Common.CreateTime,
		DiamondCount:  message.Gift.DiamondCount,
		Image:         message.Gift.Image.Url,
		RepeatEnd:     message.RepeatEnd,
		GiftID:        message.GiftId,
	}
	if message.ToUser != nil {
		model.ToUserID = message.ToUser.Id
		model.ToUserName = message.ToUser.NickName
		model.ToUserDisplayId = message.ToUser.DisplayId
	}
	if message.Common != nil && message.Common.DisplayText != nil && len(message.Common.DisplayText.Pieces) > 2 {
		pattern := message.Common.DisplayText.DefaultPattern
		pattern = strings.ReplaceAll(pattern, " ", "")
		if pattern == "{0:user}送给{1}{2}个{3:string}{4:image}" {
			model.ComboCount = message.Common.DisplayText.Pieces[2].StringValue
		} else if pattern == "{0:user}送出{1:string}{2:image}{3:string}" {
			model.ComboCount = message.Common.DisplayText.Pieces[3].StringValue[1:]
		} else if pattern == "{0:user}{1:gift}{2:string}" {
			model.ComboCount = message.Common.DisplayText.Pieces[2].StringValue[1:]
		}
	}
	return model
}

// Insert
func (model *GiftMessage) Insert() error {
	if err := DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(model).Error; err != nil {
		return err
	}
	return nil
}

func (model *GiftMessage) BatchInsert(models []*GiftMessage) error {
	if err := DB.Create(models).Error; err != nil {
		return err
	}
	return nil
}
