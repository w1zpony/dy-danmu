package core

import (
	"context"
	platform "danmu-core/core/platform/douyin"
	"danmu-core/internal/model"
	"danmu-core/logger"
	"danmu-core/utils"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
)

type Platform interface {
	GetHeartbeatValue() (interval time.Duration, hb []byte)
	GetWsInfo() (url string, headers http.Header, err error)
	DecodeMsg(data []byte, recvMsg chan interface{}, ctx context.Context, cf context.CancelFunc) (ack []byte, err error)
	CheckStream() (bool, error)
}

type MsgHandler interface {
	Handle(msg interface{}) error
}

const DefaultCron = "0 0/15 * * * ?"

type Client struct {
	liveurl    string
	p          Platform
	conn       *websocket.Conn
	connMu     sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
	enable     atomic.Bool
	isLive     atomic.Bool
	mu         sync.Mutex
	cronTask   *cron.Cron
	RecvMsg    chan interface{}
	handlers   []MsgHandler
}

type zerologCronLogger struct{}

func (z zerologCronLogger) Info(msg string, keysAndValues ...interface{}) {
	logger.Info().Msgf(msg, keysAndValues...)
}

func (z zerologCronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	logger.Error().Err(err).Msgf(msg, keysAndValues...)
}

func MakeClient(conf *model.LiveConf) *Client {
	if conf.Cron == "" {
		conf.Cron = DefaultCron
	}
	client := &Client{
		liveurl: conf.URL,
		connMu:  sync.RWMutex{},
		mu:      sync.Mutex{},
	}
	client.enable.Store(conf.Enable)
	client.isLive.Store(false)
	var err error
	switch {
	case strings.Contains(conf.URL, "douyin.com"):
		client.p, err = platform.NewDouyinPlatform(conf.URL)
	default:
		logger.Warn().Str("liveurl", conf.URL).Msg("Unsupported platform")
		return nil
	}
	if err != nil {
		logger.Warn().Str("liveurl", conf.URL).Err(err).Msg("Init platform error")
		return nil
	}

	// 初始化定时任务，用于定期检查直播状态
	// 使用 cron 库创建定时器，支持秒级精度
	client.cronTask = cron.New(
		cron.WithSeconds(), // 启用秒级精度，支持 "0 */15 * * * *" 格式
		cron.WithChain(
			cron.Recover(zerologCronLogger{}),
		),
	)

	// 添加定时任务：定期检查直播状态
	// conf.Cron 格式说明：
	// "0 */15 * * * *" - 每15分钟检查一次
	// "0 */5 * * * *"  - 每5分钟检查一次
	// "0 0 * * * *"    - 每小时检查一次
	_, err = client.cronTask.AddFunc(conf.Cron, func() {
		utils.SafeRun(client.checkStreamTask)
	})

	if err != nil {
		logger.Warn().
			Err(err).
			Str("liveurl", conf.URL).
			Str("cron", conf.Cron).
			Msg("添加定时任务失败")
		return nil
	}

	logger.Info().
		Str("liveurl", conf.URL).
		Str("cron", conf.Cron).
		Bool("enable", conf.Enable).
		Msg("客户端创建成功")

	return client
}

func (c *Client) Start() {
	c.enable.Store(true)
	utils.SafeRun(c.checkStreamTask)
	c.cronTask.Start()
	logger.Info().Str("liveurl", c.liveurl).Msg("Start Task")
}

func (c *Client) SetEnable(enable bool) {
	if !c.enable.CompareAndSwap(c.enable.Load(), enable) {
		logger.Info().Str("liveurl", c.liveurl).Msg("enable 状态已更新")
		return
	}
	if enable {
		c.Start()
	} else {
		c.Stop()
	}
}

func (c *Client) Stop() {
	c.enable.Store(false)
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	if c.cronTask != nil {
		ctx := c.cronTask.Stop()
		<-ctx.Done()
		logger.Info().Str("liveurl", c.liveurl).Msg("定时任务已停止")
	}
	logger.Info().Str("liveurl", c.liveurl).Msg("Stop Task")
}

func (c *Client) close() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	if c.RecvMsg != nil {
		close(c.RecvMsg)
		c.RecvMsg = nil
	}

	logger.Info().Str("liveurl", c.liveurl).Msg("客户端已关闭")
}

func (c *Client) run() {
	if !c.mu.TryLock() {
		return
	}
	if !c.enable.Load() {
		return
	}
	if !c.isLive.Load() {
		logger.Info().Str("liveurl", c.liveurl).Msg("live is not Living")
		return
	}
	c.RecvMsg = make(chan interface{}, 100)
	defer c.mu.Unlock()
	defer c.close()

	logger.Info().Str("liveurl", c.liveurl).Msg("Start DouyinLive")

	for c.enable.Load() && c.isLive.Load() {

		if !c.connectWebsocket(1) {
			time.Sleep(time.Second * 5)
			continue
		}

		c.ctx, c.cancelFunc = context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			logger.Info().Str("liveurl", c.liveurl).Msg("启动fetchMessage")
			utils.SafeRun(c.fetchMessage)
			logger.Info().Str("liveurl", c.liveurl).Msg("停止fetchMessage()")
			c.cancelFunc()
		}()
		go func() {
			defer wg.Done()
			logger.Info().Str("liveurl", c.liveurl).Msg("启动heartbeat")
			utils.SafeRun(c.heartbeat)
			logger.Info().Str("liveurl", c.liveurl).Msg("停止heartbeat()")
			c.cancelFunc()
		}()
		go func() {
			defer wg.Done()
			logger.Info().Str("liveurl", c.liveurl).Msg("启动processMsg")
			utils.SafeRun(c.processMsg)
			logger.Info().Str("liveurl", c.liveurl).Msg("停止processMsg()")
			c.cancelFunc()
		}()
		wg.Wait()
		if c.enable.Load() && c.isLive.Load() {
			c.checkStreamTask()
		}
	}
}

func (c *Client) connectWebsocket(i int) bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	wssUrl, headers, wsErr := c.p.GetWsInfo()
	if wsErr != nil {
		logger.Warn().Str("liveurl", c.liveurl).Err(wsErr).Msg("获取ws url失败")
		return false
	}
	var err error
	for attempt := 0; attempt < i && c.isLive.Load(); attempt++ {
		if c.conn != nil {
			err := c.conn.Close()
			if err != nil {
				logger.Warn().Str("liveurl", c.liveurl).Err(err).Msg("关闭连接失败")
			}
		}
		var resp *http.Response
		c.conn, resp, err = websocket.DefaultDialer.Dial(wssUrl, headers)
		if err != nil {
			logger.Warn().Str("liveurl", c.liveurl).Interface("resp", resp).Err(err).Msg("重连失败")
			time.Sleep(5 * time.Second)
		} else {
			logger.Info().Str("liveurl", c.liveurl).Msg("连接成功")
			return true
		}
	}
	logger.Warn().Str("liveurl", c.liveurl).Msg("重连失败")
	return false
}

func (c *Client) heartbeat() {
	interval, hb := c.p.GetHeartbeatValue()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			err := c.write(hb)
			if err != nil {
				logger.Warn().Str("liveurl", c.liveurl).Err(err).Msg("发送心跳包失败,尝试重连")
				if c.connectWebsocket(3) {
					logger.Info().Str("liveurl", c.liveurl).Msg("重连成功,继续发送心跳包")
					continue
				} else {
					logger.Warn().Str("liveurl", c.liveurl).Msg("重连失败,停止心跳")
					return
				}
			} else {
				logger.Debug().Str("liveurl", c.liveurl).Msg("Send Heart Beat")
			}
		}
	}
}

func (c *Client) fetchMessage() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msgType, data, err := c.read()
			if err != nil {
				logger.Info().Str("liveurl", c.liveurl).Int("messageType", msgType).Err(err).Msg("websocket is closed or error")
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					logger.Info().Str("liveurl", c.liveurl).Msg("WS 远程关闭")
					return
				} else {
					logger.Warn().Str("liveurl", c.liveurl).Str("resp", string(data)).Err(err).Msg("WebSocket 错误")
					if c.connectWebsocket(3) {
						continue
					} else {
						return
					}
				}
			}
			if msgType != websocket.BinaryMessage || len(data) == 0 {
				continue
			}
			ack, err := c.p.DecodeMsg(data, c.RecvMsg, c.ctx, c.cancelFunc)
			if err != nil {
				logger.Info().Str("liveurl", c.liveurl).Err(err).Msg("Parse data error")
				continue
			}
			if ack != nil {
				err := c.write(ack)
				if err != nil {
					logger.Warn().Str("liveurl", c.liveurl).Err(err).Msg("Send ack error")
				}
			}
		}
	}
}

func (c *Client) processMsg() {
	for {
		select {
		case <-c.ctx.Done():
			for response, ok := <-c.RecvMsg; ; {
				if !ok {
					logger.Info().Str("liveurl", c.liveurl).Msg("Channel closed, Stop ProcessingRecvMessage()")
					c.RecvMsg = nil
					return
				}
				c.emit(response)
			}

		case response, ok := <-c.RecvMsg:
			if !ok {
				logger.Info().Str("liveurl", c.liveurl).Msg("Channel closed, Stop ProcessingRecvMessage()")
				c.RecvMsg = nil
				return
			}
			c.emit(response)
		}
	}
}

func (c *Client) checkStreamTask() {
	isLive, err := c.p.CheckStream()
	c.isLive.Store(isLive)
	logger.Info().
		Err(err).
		Str("liveurl", c.liveurl).
		Msgf("CheckStream: %v", isLive)
	if isLive {
		go c.run()
	} else {
		c.close()
	}

}

func (c *Client) write(data []byte) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection not available")
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second * 8))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *Client) read() (int, []byte, error) {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	if conn == nil {
		return 0, nil, fmt.Errorf("connection not available")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 120))
	return conn.ReadMessage()
}

func (c *Client) Subscribe(handler MsgHandler) {
	c.handlers = append(c.handlers, handler)
}

func (c *Client) emit(msg interface{}) {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()

			// 获取文件名和行号
			_, file, line, ok := runtime.Caller(2)
			callerInfo := "unknown"
			if ok {
				callerInfo = fmt.Sprintf("%s:%d", file, line)
			}

			logger.Error().
				Str("caller", callerInfo).
				Interface("panic", err).
				Str("stack", string(stack)).
				Msg("Panic recovered in SafeRun")
		}
	}()
	for _, handler := range c.handlers {
		err := handler.Handle(msg)
		if err != nil {
			logger.Warn().Str("liveurl", c.liveurl).Err(err).Msg("handle msg error")
			continue
		}
	}
}
