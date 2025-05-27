package live

import (
	"context"
	"danmu-core/core/jsScript"
	"danmu-core/generated/douyin"
	"danmu-core/internal/model"
	"danmu-core/logger"
	"danmu-core/utils"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"google.golang.org/protobuf/proto"
)

// 正则表达式用于提取 roomID 和 pushID
var (
	heartbeatInterval = 10
	hb, _             = proto.Marshal(&douyin.PushFrame{
		PayloadType: "hb",
	})
)

func NewDouyinLive(conf *model.LiveConf) (*DouyinLive, error) {
	if !strings.Contains(conf.URL, "https://live.douyin.com/") && !strings.Contains(conf.URL, "v.douyin") {
		return nil, fmt.Errorf("“This Url is not Douyin url: %v", conf.URL)
	}
	ua := utils.RandomUserAgent()
	d := &DouyinLive{
		liveurl:       conf.URL,
		userAgent:     ua,
		c:             req.C(),
		eventHandlers: make([]EventHandler, 0),
		isLive:        atomic.Bool{},
		enable:        atomic.Bool{},
	}
	d.c.SetUserAgent(ua)
	d.c.SetTimeout(time.Second * 10)

	//设置requset.Client header用于获取roomInfo
	d.c.SetCommonHeaders(map[string]string{
		"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"accept-language": "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3",
	})

	d.c.SetRedirectPolicy(req.NoRedirectPolicy())

	//初始化去重lru cache
	var err error
	d.distinctCache, err = lru.New(100)
	if err != nil {
		return nil, err
	}

	d.enable.Store(conf.Enable)

	// 加载 JavaScript 脚本
	err = jsScript.LoadGoja(d.userAgent)
	if err != nil {
		return nil, fmt.Errorf("加载 Goja 脚本失败: %w", err)
	}
	logger.Info().Str("liveurl", d.liveurl).Msg("DouyinLive创建成功")
	return d, nil
}

func (d *DouyinLive) SetEnable(enable bool) error {
	// If the current value equals the new value, log and return
	if d.enable.Load() == enable {
		logger.Info().
			Str("liveurl", d.liveurl).
			Bool("enable", enable).
			Msg("DouyinLive enable state unchanged")
		return nil
	}

	// Set the new value and take appropriate action
	if enable {
		_, err := d.CheckStream()
		if err != nil {
			return err
		}
		go utils.SafeRun(d.Start)
		d.enable.Store(true)
	} else {
		d.Stop()
		d.enable.Store(false)
	}

	logger.Info().
		Str("liveurl", d.liveurl).
		Bool("enable", enable).
		Msg("DouyinLive enable state changed")
	return nil
}

// 外部设置enable=false
func (d *DouyinLive) Stop() {
	if !d.enable.CompareAndSwap(true, false) {
		logger.Info().Str("liveurl", d.liveurl).Msg("DouyinLive is not running")
		return
	}
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	logger.Info().Str("liveurl", d.liveurl).Msg("Stop DouyinLive")
}

func (d *DouyinLive) Start() {
	if !d.enable.Load() {
		return
	}
	if !d.isLive.Load() {
		logger.Info().Str("liveurl", d.liveurl).Msg("DouyinLive is not Living")
		return
	}
	if !d.mu.TryLock() {
		return
	}
	defer d.mu.Unlock()
	defer d.Close()

	err := d.fetchTTWID()
	if err != nil {
		logger.Warn().Msg("fetch ttwid error")
		return
	}
	d.c.SetCommonCookies(&http.Cookie{
		Name:  "ttwid",
		Value: d.ttwid,
	})
	//设置websocket header
	d.headers = http.Header{}
	d.headers.Add("user-agent", d.userAgent)
	d.headers.Add("cookie", fmt.Sprintf("ttwid=%s", d.ttwid))
	d.pushid = utils.GetUserUniqueID()

	logger.Info().Str("liveurl", d.liveurl).Msg("Start DouyinLive")
	d.wssurl = d.StitchUrl()
	d.recvMsgChan = make(chan *douyin.Response, 100)
	defer d.isLive.Store(false)
	defer close(d.recvMsgChan)

	for d.enable.Load() && d.isLive.Load() {
		if !d.reconnect(1) {
			continue
		}
		d.ctx, d.cancelFunc = context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			utils.SafeRun(d.fetchMessage)
			logger.Info().Str("liveurl", d.liveurl).Msg("停止fetchMessage()")
			d.cancelFunc()
		}()
		go func() {
			defer wg.Done()
			utils.SafeRun(d.heartbeat)
			logger.Info().Str("liveurl", d.liveurl).Msg("停止heartbeat()")
			d.cancelFunc()
		}()
		go func() {
			defer wg.Done()
			utils.SafeRun(d.processingRecvMessage)
			logger.Info().Str("liveurl", d.liveurl).Msg("停止processRecvMessage()")
			d.cancelFunc()
		}()
		wg.Wait()
		time.Sleep(time.Second * 10)
	}
}

func (d *DouyinLive) fetchMessage() {
	d.recvMsgChan = make(chan *douyin.Response, 100)
	defer close(d.recvMsgChan)
	logger.Info().Str("liveurl", d.liveurl).Msg("启动fetchMessage")
	var pbPac = &douyin.PushFrame{}
	var pbResp = &douyin.Response{}
	var pbAck = &douyin.PushFrame{}
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			d.Conn.SetReadDeadline(time.Now().Add(11 * time.Second))
			msgType, message, err := d.Conn.ReadMessage()
			if err != nil {
				logger.Info().Str("liveurl", d.liveurl).Int("messageType", msgType).Err(err).Msg("websocket is closed or error")
				if d.reconnect(3) {
					continue
				}
				return
			}
			if message == nil {
				continue
			}
			err = proto.Unmarshal(message, pbPac)
			if err != nil {
				logger.Info().Str("liveurl", d.liveurl).Err(err).Msg("解析消息失败")
				continue
			}
			//心跳
			if pbPac.LogId == 0 {
				continue
			}
			if !d.distinct(pbPac.LogId) {
				continue
			}
			n := utils.HasGzipEncoding(pbPac.HeadersList)
			if n && pbPac.PayloadType == "msg" {
				uncompressedData, err := d.GzipUnzipReset(pbPac.Payload)
				if err != nil {
					logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("Gzip解压失败")
					continue
				}

				err = proto.Unmarshal(uncompressedData, pbResp)
				if err != nil {
					logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("解析消息失败")
					continue
				}
				if pbResp.NeedAck {
					pbAck.Reset()
					pbAck.LogId = pbPac.LogId
					pbAck.PayloadType = "ack"
					pbAck.Payload = []byte(pbResp.InternalExt)
					serializedAck, err := proto.Marshal(pbAck)
					if err != nil {
						logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("Ack包序列化失败")
						continue
					}
					err = d.Conn.WriteMessage(websocket.BinaryMessage, serializedAck)
					if err != nil {
						logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("Ack包发送失败")
						continue
					}
				}
				d.recvMsgChan <- pbResp
			}
		}
	}
}

// reconnect 尝试重新连接
func (d *DouyinLive) reconnect(i int) bool {
	d.connMu.Lock()
	defer d.connMu.Unlock()
	var err error
	logger.Info().Str("liveurl", d.liveurl).Msg("尝试连接...")
	for attempt := 0; attempt < i && d.isLive.Load(); attempt++ {
		if d.Conn != nil {
			err := d.Conn.Close()
			if err != nil {
				logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("关闭连接失败")
			}
		}
		d.Conn, _, err = websocket.DefaultDialer.Dial(d.wssurl, d.headers)
		if err != nil {
			logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("重连失败")
			time.Sleep(15 * time.Second)
		} else {
			logger.Info().Str("liveurl", d.liveurl).Msg("重连成功")
			return true
		}
	}
	logger.Warn().Str("liveurl", d.liveurl).Msg("重连失败")
	return false
}

// emit 触发事件处理器
func (d *DouyinLive) emit(eventData *douyin.Message) {
	for _, handler := range d.eventHandlers {
		handler(eventData)
	}
}

// ProcessingMessage 处理接收到的消息
func (d *DouyinLive) processingRecvMessage() {
	logger.Info().Str("liveurl", d.liveurl).Msg("启动processingRecvMessage")
	for {
		select {
		case <-d.ctx.Done():
			return

		case response, ok := <-d.recvMsgChan:
			if !ok {
				logger.Info().Str("liveurl", d.liveurl).Msg("Channel closed, Stop ProcessingRecvMessage()")
				return
			}
			for _, data := range response.MessagesList {
				d.emit(data)
				if data.Method == "WebcastControlMessage" {
					msg := &douyin.ControlMessage{}
					err := proto.Unmarshal(data.Payload, msg)
					if err != nil {
						logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("解析protobuf失败")
						return
					}
					if msg.Status == 3 || msg.Status == 4 {
						logger.Info().Str("liveurl", d.liveurl).Msg("msg.Status为3|4，关闭ws链接")
						return
					}
				}
			}
		}
	}
}

// Subscribe 订阅事件处理器
func (d *DouyinLive) Subscribe(handler EventHandler) {
	d.eventHandlers = append(d.eventHandlers, handler)
}

func (d *DouyinLive) Close() {
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	if d.gzip != nil {
		err := d.gzip.Close()
		d.gzip = nil
		if err != nil {
			logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("gzip关闭失败")
		} else {
			logger.Info().Str("liveurl", d.liveurl).Msg("gzip关闭")
		}
	}
	d.connMu.Lock()
	defer d.connMu.Unlock()
	if d.Conn != nil {
		err := d.Conn.Close()
		d.Conn = nil
		if err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("关闭ws链接失败")
		} else {
			logger.Info().Str("liveurl", d.liveurl).Msg("抖音ws链接关闭")
		}
	}
}

// SendHeartbeat 发送心跳包
func (d *DouyinLive) heartbeat() {
	ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
	logger.Info().Str("liveurl", d.liveurl).Msg("启动heartbeat")
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if !d.connMu.TryLock() {
				continue
			}
			if err := d.Conn.WriteMessage(websocket.BinaryMessage, hb); err != nil {
				logger.Warn().Str("liveurl", d.liveurl).Err(err).Msg("发送心跳包失败")
			} else {
				logger.Debug().Str("liveurl", d.liveurl).Msg("Send Heart Beat")
			}
			d.connMu.Unlock()
		}
	}
}

func (d *DouyinLive) CheckStream() (bool, error) {
	if d.ttwid == "" {
		err := d.fetchTTWID()
		if err != nil {
			return false, err
		}
		d.c.SetCommonCookies(&http.Cookie{
			Name:  "ttwid",
			Value: d.ttwid,
		})
	}
	if strings.Contains(d.liveurl, "v.douyin") {
		resp, err := d.c.SetRedirectPolicy(req.NoRedirectPolicy()).R().Get(d.liveurl)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
			d.isLive.Store(false)
			return false, err
		}

		nextUrl := resp.Header.Get("Location")
		if !strings.Contains(nextUrl, "webcast.amemv") {
			d.isLive.Store(false)
			return false, err
		}
		var secUidRegexp = regexp.MustCompile(`sec_user_id=(.*?)&`)
		var roomIdRegexp = regexp.MustCompile(`(\d+)`)
		d.secUid = extractMatch(secUidRegexp, nextUrl)
		d.roomId = extractMatch(roomIdRegexp, strings.Split(nextUrl, "?")[0])
	} else {
		webRid := strings.Split(strings.Split(d.liveurl, "douyin.com/")[1], "/")[0]
		webRid = strings.TrimPrefix(webRid, "+")
		d.webRid = webRid //抖音displayID
	}

	// 获取房间信息
	var roomInfo map[string]interface{}
	if d.webRid != "" {
		var err error
		roomInfo, err = d.getWebRoomInfo(d.webRid)
		if err != nil {
			logger.Warn().Err(err)
		}
		if roomInfo != nil && roomInfo["data"] != nil && roomInfo["data"].(map[string]interface{})["user"] != nil {
			d.secUid = roomInfo["data"].(map[string]interface{})["user"].(map[string]interface{})["sec_uid"].(string)
		}
	}

	// 如果没有数据，尝试通过 sec_uid 和 room_id 获取房间信息
	if roomInfo == nil || roomInfo["data"] == nil {
		roomInfo, _ = d.getRoomInfo(d.secUid, d.roomId)
		if roomInfo != nil && roomInfo["data"].(map[string]interface{})["room"].(map[string]interface{})["owner"] != nil {
			d.webRid = roomInfo["data"].(map[string]interface{})["room"].(map[string]interface{})["owner"].(map[string]interface{})["web_rid"].(string)
		}
	}

	// 检查房间状态
	if roomInfo != nil {
		roomData := roomInfo["data"].(map[string]interface{})["data"].([]interface{})
		if roomData != nil {
			// 如果房间信息的状态不等于 2，则表示未开播
			if roomData[0].(map[string]interface{})["status"].(float64) != 2 {
				d.isLive.Store(false)
				return false, nil
			}
			d.roomId = roomData[0].(map[string]interface{})["id_str"].(string)
		}
	}

	d.isLive.Store(true)
	return true, nil
}

/*https://live.douyin.com/webcast/room/web/enter/?aid=6383&device_platform=web&language=zh-CN&enter_from=web_live&browser_language=zh-CN&browser_platform=Win32&browser_name=Firefox&browser_version=134.0&web_rid=370234906886*/
