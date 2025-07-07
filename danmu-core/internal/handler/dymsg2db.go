package handler

import (
	platform "danmu-core/core/platform/douyin"
	"danmu-core/generated/dystruct"
	"danmu-core/internal/model"
	"danmu-core/logger"
	"danmu-core/utils"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Dymsg2dbHandler struct {
	cache         *lru.Cache
	roomDisplayId string
	roomName      string
	liveUrl       string
}

func NewDymsg2dbHandler(conf *model.LiveConf) (*Dymsg2dbHandler, error) {
	cache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("Dymsg2dbHandler Init Cache failure, err:%v", err)
	}
	return &Dymsg2dbHandler{
		cache:         cache,
		roomDisplayId: conf.RoomDisplayID,
		roomName:      conf.Name,
		liveUrl:       conf.URL,
	}, nil
}

func (h *Dymsg2dbHandler) Handle(msg interface{}) error {
	message := msg.(*dystruct.Webcast_Im_Message)
	unMarshallMsg, err := platform.MatchMethod(message.Method)
	if err != nil || unMarshallMsg == nil {
		return fmt.Errorf("proto type undefied")
	}
	if _, exists := h.cache.Get(message.MsgId); exists {
		return nil
	}
	if err := proto.Unmarshal(message.Payload, unMarshallMsg); err != nil {
		return fmt.Errorf("unmarshal failed")
	}
	if err := h.saveToDB(unMarshallMsg, message.Method, message.MsgId); err != nil {
		return err
	}
	h.cache.Add(message.MsgId, true)
	return nil
}

func (h *Dymsg2dbHandler) saveToDB(msg protoreflect.ProtoMessage, method string, id uint64) error {
	var common *model.CommonMessage
	switch method {
	case platform.WebcastGiftMessage:
		m := msg.(*dystruct.Webcast_Im_GiftMessage)
		// 先处理用户信息
		user := model.NewUser(m.User)
		if err := user.CheckAndInsert(); err != nil {
			logger.Warn().Str("liveid", h.roomDisplayId).Err(err).Msg("Failed to process user")
			// 不返回错误，继续处理礼物消息
		}

		if m.RepeatEnd == 1 {
			return nil
		}
		common = &model.CommonMessage{
			MessageType:   method,
			UserName:      m.User.Nickname,
			UserID:        m.User.Id,
			UserDisplayId: m.User.DisplayId,
			RoomID:        m.Common.RoomId,
			Content:       m.Common.Describe,
			Timestamp:     m.Common.CreateTime,
			RoomName:      h.roomName,
			RoomDisplayId: h.roomDisplayId,
		}
		giftMessage := model.NewGiftMessage(m)
		giftMessage.ID = int64(id)
		giftMessage.RoomDisplayId = h.roomDisplayId
		giftMessage.RoomName = h.roomName
		if err := giftMessage.Insert(); err != nil {
			logger.Warn().Str("liveid", h.roomDisplayId).Err(err).
				Msgf("Failed to insert gift message: %v", m)
			return err
		} else {
			logger.Debug().Str("liveid", h.roomDisplayId).Msgf("insert new giftmessage [%d]%v", giftMessage.ID, giftMessage.Message)
		}
	case platform.WebcastChatMessage:
		m := msg.(*dystruct.Webcast_Im_ChatMessage)
		common = &model.CommonMessage{
			MessageType:   method,
			UserName:      m.User.Nickname,
			UserID:        m.User.Id,
			UserDisplayId: m.User.DisplayId,
			RoomID:        m.Common.RoomId,
			RoomDisplayId: h.roomDisplayId,
			RoomName:      h.roomName,
			Content:       fmt.Sprintf("[%v]: %v", m.User.Nickname, m.Content),
			Timestamp:     m.EventTime,
		}

	default:
		return nil
	}
	if common != nil {
		common.ID = id
		common.Timestamp = uint64(utils.NormalizeTimestamp(int64(common.Timestamp)))
		if err := common.Insert(); err != nil {
			logger.Warn().Str("liveid", h.roomDisplayId).Err(err).
				Msgf("Failed to insert common message: %v", common)
			return err
		} else {
			logger.Debug().Str("liveid", h.roomDisplayId).Msgf("insert new commonmessage [%d]%v", common.ID, common.Content)
		}
		//	logger.Debug().Uint64("commonid", common.ID).Msg("save to db")
	}
	return nil
}
