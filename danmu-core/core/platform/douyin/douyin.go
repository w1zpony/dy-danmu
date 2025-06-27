package platform

import (
	"bytes"
	"danmu-core/core/platform/douyin/jsScript"
	"danmu-core/generated/douyin"
	"danmu-core/generated/dystruct"
	"danmu-core/logger"
	"danmu-core/utils"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/proto"
)

const (
	gzipBufferSize = 1024 * 4
)

type Douyin struct {
	ua         string
	ttwid      string
	roomId     string
	webRid     string
	secUid     string
	liveurl    string
	bufferPool *sync.Pool
	client     *req.Client
	header     map[string]string
	gd         *jsScript.GojaDouyin
}

func NewDouyinPlatform(liveurl string) (*Douyin, error) {
	ua := utils.RandomUserAgent()
	ttwid, err := getTTWID()
	if err != nil {
		return nil, fmt.Errorf("fetch ttwid error: %w", err)
	}
	dy := &Douyin{
		ua:         ua,
		ttwid:      ttwid,
		client:     req.C(),
		bufferPool: &sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, gzipBufferSize)) }},
		liveurl:    liveurl,
	}
	dy.gd, err = jsScript.LoadGoja(ua)
	if err != nil {
		return nil, fmt.Errorf("init goja js error: %w", err)
	}
	dy.header = map[string]string{
		"User-Agent": ua,
		"Referer":    "https://live.douyin.com/",
		"Cookie":     fmt.Sprintf("ttwid=%s", dy.ttwid),
	}
	return dy, nil
}

func (dy *Douyin) GetHeartbeatValue() (interval time.Duration, hb []byte) {
	heartbeatInterval := 10
	hb, _ = proto.Marshal(&douyin.PushFrame{
		PayloadType: "hb",
	})
	return time.Duration(heartbeatInterval) * time.Second, hb
}

func (dy *Douyin) GetWsInfo() (url string, headers http.Header, err error) {
	url, err = dy.getDouyinWsUrl()
	headers = http.Header{}
	headers.Set("User-Agent", dy.ua)
	headers.Set("cookie", fmt.Sprintf("ttwid=%s", dy.ttwid))
	return url, headers, err
}

func (dy *Douyin) DecodeMsg(data []byte, recvMsg chan interface{}) (ack []byte, err error) {
	var pushFrame dystruct.Webcast_Im_PushFrame
	if err := proto.Unmarshal(data, &pushFrame); err != nil {
		return nil, fmt.Errorf("unmarshal push frame error: %w", err)
	}

	decompressed, err := dy.decompressGzip(pushFrame.Payload)
	if err != nil {
		return nil, fmt.Errorf("gzip解压数据失败: %w", err)
	}

	var response dystruct.Webcast_Im_Response
	if err := proto.Unmarshal(decompressed, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response error: %w", err)
	}

	for _, msg := range response.Messages {
		recvMsg <- msg
	}

	var ackData []byte
	if response.NeedAck {
		ackFrame := &dystruct.Webcast_Im_PushFrame{
			LogID:       pushFrame.LogID,
			PayloadType: "ack",
			Payload:     []byte(response.InternalExt),
		}
		ack, err = proto.Marshal(ackFrame)
		if err != nil {
			logger.Warn().Err(err).Msg("marshal ack frame error")
		}
	}

	return ackData, nil
}

func (dy *Douyin) CheckStream() (bool, error) {
	webRidMatches := webRidRg.FindStringSubmatch(dy.liveurl)
	if len(webRidMatches) <= 1 {
		return false, fmt.Errorf("未找到 web_rid")
	}
	dy.webRid = webRidMatches[1]
	info, err := dy.getWebRoomInfo(dy.webRid)
	if err != nil {
		return false, err
	}
	user := info.Get("data.user")
	if !user.Exists() {
		return false, fmt.Errorf("未找到用户信息: %s", info.Raw)
	}
	dy.secUid = user.Get("sec_uid").String()

	var finalRoomInfo gjson.Result

	if dataArray := info.Get("data.data"); dataArray.Exists() && dataArray.IsArray() {
		if len(dataArray.Array()) > 0 {
			finalRoomInfo = dataArray.Array()[0]
		}
	} else {
		return false, fmt.Errorf("room info is not exist: %s", info.Raw)
	}
	dy.roomId = finalRoomInfo.Get("id_str").String()
	status := finalRoomInfo.Get("status").Int()
	if status != 2 {
		return false, fmt.Errorf("未开播")
	}

	return true, nil
}

func (dy *Douyin) getWebRoomInfo(webRid string) (gjson.Result, error) {
	targetURL, err := dy.BuildRequestURL(fmt.Sprintf("https://live.douyin.com/webcast/room/web/enter/?web_rid=%s", webRid))
	if err != nil {
		return gjson.Result{}, fmt.Errorf("build request url error: %w", err)
	}
	resp, err := dy.client.R().SetHeaders(dy.header).Get(targetURL)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode != 200 {
		return gjson.Result{}, fmt.Errorf("请求返回状态码: %d", resp.StatusCode)
	}

	return gjson.Parse(resp.String()), nil
}

func (dy *Douyin) getDouyinWsUrl() (string, error) {
	uniqueId := utils.GetUserUniqueID()
	smap := NewSigMap(dy.roomId, uniqueId)
	signaturemd5 := GetxMSStub(smap)
	signature := dy.gd.GetSign(signaturemd5)
	baseURl := "wss://webcast5-ws-web-lf.douyin.com/webcast/im/push/v2/"
	initialWss := baseURl + "?" + NewWebCast5Param(dy.roomId, uniqueId, signature).Encode()
	return dy.BuildRequestURL(initialWss)
}
