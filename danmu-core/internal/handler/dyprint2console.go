package handler

import (
	platform "danmu-core/core/platform/douyin"
	"danmu-core/generated/dystruct"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type DyPrint2Console struct {
	roomDisplayId string
}

func NewDyPrint2ConsoleHandler(roomId string) *DyPrint2Console {
	return &DyPrint2Console{
		roomDisplayId: roomId,
	}
}

func (h *DyPrint2Console) Handle(msg interface{}) error {
	message := msg.(*dystruct.Webcast_Im_Message)
	unMarshallMsg, err := platform.MatchMethod(message.Method)
	if err != nil || unMarshallMsg == nil {
		return fmt.Errorf("proto type undefied")
	}

	if err := proto.Unmarshal(message.Payload, unMarshallMsg); err != nil {
		return fmt.Errorf("unmarshal failed")
	}
	if err := h.print(unMarshallMsg, message.Method, message.MsgId); err != nil {
		return err
	}
	return nil
}

func (h *DyPrint2Console) print(msg protoreflect.ProtoMessage, method string, id uint64) error {
	var content string
	switch method {
	case platform.WebcastGiftMessage:
		m := msg.(*dystruct.Webcast_Im_GiftMessage)
		content = fmt.Sprintf("[%v]: %v", m.User.Nickname, m.Common.Describe)
	case platform.WebcastChatMessage:
		m := msg.(*dystruct.Webcast_Im_ChatMessage)
		content = fmt.Sprintf("[%v]: %v", m.User.Nickname, m.Content)
	case platform.WebcastMemberMessage:
		m := msg.(*dystruct.Webcast_Im_MemberMessage)
		content = fmt.Sprintf("%v 来了, 人数 %v", m.User.Nickname, m.MemberCount)

	case platform.WebcastSocialMessage:
		m := msg.(*dystruct.Webcast_Im_SocialMessage)
		content = fmt.Sprintf("%v 关注了，Follow Count: %v", m.User.Nickname, m.FollowCount)

	case platform.WebcastLikeMessage:
		m := msg.(*dystruct.Webcast_Im_LikeMessage)
		content = fmt.Sprintf("%v 为主播点赞， Total: %v", m.User.Nickname, m.Total)
	default:
		return nil
	}
	fmt.Printf("room[%v]: %v \n", h.roomDisplayId, content)
	return nil
}
