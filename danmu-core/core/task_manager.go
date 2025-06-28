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
		if err := Add(conf); err != nil {
			logger.Warn().Err(err).Str("liveurl", conf.URL).Msg("Add task failed")
		}
	}
}

func Add(conf *model.LiveConf) error {
	if _, ok := muMap[conf.ID]; ok {
		logger.Info().Str("liveurl", conf.URL).Msg("task already exists")
		return nil
	}
	muMap[conf.ID] = &sync.Mutex{}
	mu, ok := muMap[conf.ID]
	if !ok {
		logger.Info().Int64("id", conf.ID).Msg("task not found")
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	client := MakeClient(conf)
	if client == nil {
		logger.Warn().Str("liveurl", conf.URL).Msg("MakeClient failed")
		delete(muMap, conf.ID)
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
	TaskMap[conf.ID] = task
	return nil
}

func Update(conf *model.LiveConf) error {
	mu, ok := muMap[conf.ID]
	if !ok {
		logger.Info().Int64("id", conf.ID).Msg("task not found")
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	var task *Task
	task, ok = TaskMap[conf.ID]
	if !ok {
		logger.Warn().Int64("id", conf.ID).Msg("task not found")
		return fmt.Errorf("task not found")
	}
	if conf.URL != task.url {
		task.client.Stop()
		delete(TaskMap, conf.ID)

		client := MakeClient(conf)
		if client == nil {
			logger.Warn().Str("liveurl", conf.URL).Msg("MakeClient failed")
			delete(muMap, conf.ID)
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
		TaskMap[conf.ID] = task
		return nil
	}
	if conf.Enable != task.client.enable.Load() {
		task.client.SetEnable(conf.Enable)
	}
	return nil
}
func Delete(id int64) error {
	mu, ok := muMap[id]
	if !ok {
		logger.Info().Int64("id", id).Msg("task not found")
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if task, ok := TaskMap[id]; ok {
		task.client.Stop()
		delete(TaskMap, id)
	}
	delete(muMap, id)
	return nil
}

func Stop(id int64) error {
	mu, ok := muMap[id]
	if !ok {
		logger.Info().Int64("id", id).Msg("task not found")
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if task, ok := TaskMap[id]; ok {
		task.client.Stop()
	}
	return nil
}
