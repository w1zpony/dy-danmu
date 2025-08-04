package core

import (
	"danmu-core/internal/handler"
	"danmu-core/internal/model"
	"danmu-core/logger"
	"fmt"
	"sync"
)

var TaskMap map[int64]*Task
var muMap map[int64]*sync.Mutex
var mapMutex sync.RWMutex

type Task struct {
	url      string
	taskId   int64
	client   *Client
	handlers []MsgHandler
	RecvChan chan interface{}
}

// todo 提供修改cron表达式功能
// todo 提供unsubscribe handler功能，用于变动直播间名字时重新订阅handler
// todo handler改为可配置，添加rpc接口
// todo 添加cookie自定义
// todo 修改初始化，支持distributed多节点部署，动态负载均衡
// todo 修改platform和handler获取方式，使用plugin特性动态加载，可不重新启动项目就能动态加载新的Platform和handler
func InitTaskManager() {
	confs, err := model.GetAllLiveConf()
	if err != nil {
		logger.Error().Err(err).Msg("获取所有直播配置失败")
		return
	}
	TaskMap = make(map[int64]*Task, len(confs))
	muMap = make(map[int64]*sync.Mutex, len(confs))
	for _, conf := range confs {
		go func(c *model.LiveConf) {
			if err := Add(c); err != nil {
				logger.Warn().Err(err).Str("liveurl", c.URL).Msg("Add task failed")
			}
		}(conf)
	}
}

func Add(conf *model.LiveConf) error {
	mapMutex.RLock()
	_, ok := muMap[conf.ID]
	mapMutex.RUnlock()

	if ok {
		logger.Info().Str("liveurl", conf.URL).Msg("task already exists")
		return nil
	}

	mapMutex.Lock()
	// Double-check after acquiring write lock
	if _, exists := muMap[conf.ID]; exists {
		mapMutex.Unlock()
		logger.Info().Str("liveurl", conf.URL).Msg("task already exists")
		return nil
	}
	muMap[conf.ID] = &sync.Mutex{}
	mu := muMap[conf.ID]
	mapMutex.Unlock()

	mu.Lock()
	defer mu.Unlock()
	client := MakeClient(conf)
	if client == nil {
		logger.Warn().Str("liveurl", conf.URL).Msg("MakeClient failed")
		mapMutex.Lock()
		delete(muMap, conf.ID)
		mapMutex.Unlock()
		return fmt.Errorf("MakeClient failed,conf: %v", conf)
	}
	task := &Task{
		url:      conf.URL,
		taskId:   conf.ID,
		client:   client,
		handlers: []MsgHandler{},
		RecvChan: client.RecvMsg,
	}
	h, err := handler.NewDymsg2dbHandler(conf)
	if err != nil {
		logger.Warn().Err(err).Str("liveurl", conf.URL).Msg("NewDymsg2dbHandler failed")
		return fmt.Errorf("NewDymsg2dbHandler failed,conf: %v", conf)
	}
	task.client.Subscribe(h)
	if conf.Enable {
		task.client.Start()
	}
	mapMutex.Lock()
	TaskMap[conf.ID] = task
	mapMutex.Unlock()
	return nil
}

func Update(conf *model.LiveConf) error {
	mapMutex.RLock()
	mu, ok := muMap[conf.ID]
	if !ok {
		mapMutex.RUnlock()
		logger.Info().Int64("id", conf.ID).Msg("task not found")
		return nil
	}
	task, taskExists := TaskMap[conf.ID]
	mapMutex.RUnlock()

	if !taskExists {
		logger.Warn().Int64("id", conf.ID).Msg("task not found")
		return fmt.Errorf("task not found")
	}

	mu.Lock()
	defer mu.Unlock()
	if conf.URL != task.url {
		task.client.Stop()
		mapMutex.Lock()
		delete(TaskMap, conf.ID)
		mapMutex.Unlock()

		client := MakeClient(conf)
		if client == nil {
			logger.Warn().Str("liveurl", conf.URL).Msg("MakeClient failed")
			mapMutex.Lock()
			delete(muMap, conf.ID)
			mapMutex.Unlock()
			return fmt.Errorf("MakeClient failed,conf: %v", conf)
		}
		task := &Task{
			url:      conf.URL,
			taskId:   conf.ID,
			client:   client,
			handlers: []MsgHandler{},
			RecvChan: client.RecvMsg,
		}
		h, err := handler.NewDymsg2dbHandler(conf)
		if err != nil {
			logger.Warn().Err(err).Str("liveurl", conf.URL).Msg("NewDymsg2dbHandler failed")
			return fmt.Errorf("NewDymsg2dbHandler failed,conf: %v", conf)
		}
		task.client.Subscribe(h)
		if conf.Enable {
			task.client.Start()
		}
		mapMutex.Lock()
		TaskMap[conf.ID] = task
		mapMutex.Unlock()
		return nil
	}
	if conf.Enable != task.client.enable.Load() {
		task.client.SetEnable(conf.Enable)
	}
	return nil
}
func Delete(id int64) error {
	mapMutex.RLock()
	mu, ok := muMap[id]
	if !ok {
		mapMutex.RUnlock()
		logger.Info().Int64("id", id).Msg("task not found")
		return nil
	}
	task, taskExists := TaskMap[id]
	mapMutex.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	if taskExists {
		task.client.Stop()
	}

	mapMutex.Lock()
	delete(TaskMap, id)
	delete(muMap, id)
	mapMutex.Unlock()
	return nil
}
