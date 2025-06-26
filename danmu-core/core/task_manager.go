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
