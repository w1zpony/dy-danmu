package manager

import (
	"danmu-core/core/live"
	"danmu-core/internal/handler"
	"danmu-core/internal/model"
	"danmu-core/logger"
	"danmu-core/utils"
	"fmt"
	"sync"
	"time"
)

var (
	taskList map[int64]*DouyinTask
	muMap    map[int64]*sync.Mutex
)

type DouyinTask struct {
	live *live.DouyinLive
	conf *model.LiveConf
}

func InitDouyinManager() {
	logger.Info().Msg("INIT DOYIN MANAGER")
	taskList = make(map[int64]*DouyinTask)
	confs, err := model.GetAllLiveConf()
	muMap = make(map[int64]*sync.Mutex, len(taskList))
	if err != nil {
		logger.Fatal().Err(err).Msg("get live conf failed")
	}
	for _, conf := range confs {
		dylive, err := live.NewDouyinLive(conf)
		if err != nil {
			logger.Warn().Err(err).Str("room_display_id", conf.RoomDisplayID).Msg("init live failed")
			continue
		}
		dyhandler, err := handler.NewDouyinHandler(conf)
		if err != nil {
			logger.Warn().Err(err).Str("room_display_id", conf.RoomDisplayID)
			continue
		}
		dylive.Subscribe(dyhandler.HandleDouyinMessage)
		taskList[conf.ID] = &DouyinTask{
			live: dylive,
			conf: conf,
		}
		muMap[conf.ID] = &sync.Mutex{}
	}
	go checkAllLiveTimer()
}

func AddDouyinTask(conf *model.LiveConf) error {
	muMap[conf.ID].Lock()
	defer muMap[conf.ID].Unlock()
	if _, ok := taskList[conf.ID]; ok {
		logger.Warn().Interface("conf", conf).Msg("[Add]task already exist")
		return fmt.Errorf("live task already exists: %s", conf.RoomDisplayID)
	}
	dylive, err := live.NewDouyinLive(conf)
	if err != nil {
		logger.Warn().Err(err).Interface("conf", conf).Msg("[Add]Create douyinLive fail")
		return fmt.Errorf("init live failed: %w", err)
	}
	dyhandler, err := handler.NewDouyinHandler(conf)
	if err != nil {
		logger.Warn().Err(err).Interface("conf", conf).Msg("[Add]handler create fail")
		return fmt.Errorf("live handler create failed: %w", err)
	}
	dylive.Subscribe(dyhandler.HandleDouyinMessage)
	taskList[conf.ID] = &DouyinTask{
		live: dylive,
		conf: conf,
	}
	isLive, err := dylive.CheckStream()
	if err != nil {
		logger.Warn().Err(err).Interface("conf", conf).Msg("[Add]Check Stream Error")
	}
	if isLive {
		go utils.SafeRun(dylive.Start)
	}
	logger.Info().Str("room_display_id", conf.RoomDisplayID).Str("room_name", conf.Name).Msg("[Add]add live task success")
	return nil
}

func DeleteDouyinTask(id int64) error {
	muMap[id].Lock()
	defer muMap[id].Unlock()
	task, ok := taskList[id]
	if !ok {
		logger.Info().Msg("[Delete]delete live task fail, task doesn't exist")
		return fmt.Errorf("task not found: %d", id)
	}
	task.live.Stop()
	delete(taskList, id)
	logger.Info().Str("room_display_id", task.conf.RoomDisplayID).Str("room_name", task.conf.Name).Msg("[Delete]delete live task success")
	return nil
}

func UpdateDouyinTask(conf *model.LiveConf) error {
	muMap[conf.ID].Lock()
	defer muMap[conf.ID].Unlock()
	task, ok := taskList[conf.ID]
	if !ok {
		logger.Info().Interface("conf", conf).Msg("[Update] task not found, create new task")
		if err := AddDouyinTask(conf); err != nil {
			logger.Warn().Interface("conf", conf).Msg("[Update] task create failure")
			return fmt.Errorf("task not found ,create task failed")
		}
		return nil
	}
	logger.Info().Str("room_display_id", conf.RoomDisplayID).Str("room_name", conf.Name).Msg("update live task")

	if conf.URL != task.conf.URL {
		dylive, err := live.NewDouyinLive(conf)
		if err != nil {
			return fmt.Errorf("init live failed: %w", err)
		}
		dyhandler, err := handler.NewDouyinHandler(conf)
		if err != nil {
			logger.Error().Err(err).Str("room_display_id", conf.RoomDisplayID).Str("room_name", conf.Name)
		}
		dylive.Subscribe(dyhandler.HandleDouyinMessage)
		taskList[conf.ID].live.Stop()
		taskList[conf.ID] = &DouyinTask{
			live: dylive,
			conf: conf,
		}
		return nil
	}

	if conf.Enable != task.conf.Enable {
		err := task.live.SetEnable(conf.Enable)
		if err != nil {
			return err
		}
	}
	task.conf = conf
	return nil
}

func CloseDouyinManager() {
	for _, task := range taskList {
		task.live.Stop()
		delete(taskList, task.conf.ID)
	}
}

func checkAllLiveTimer() {
	for {
		logger.Info().Msg("BEGIN TO CHECK ALL LIVE")
		for _, task := range taskList {
			muMap[task.conf.ID].Lock()
			if !task.conf.Enable {
				continue
			}
			utils.SafeRun(func() {
				if isLive, err := task.live.CheckStream(); err != nil {
					if isLive {
						logger.Info().Str("url", task.conf.URL).Msg("CheckStream: live is living")
						go utils.SafeRun(task.live.Start)
					} else {
						logger.Info().Str("url", task.conf.URL).Msg("CheckStream: live is closed")
					}
				} else {
					logger.Warn().Err(err)
				}
			})
			muMap[task.conf.ID].Unlock()
		}
		time.Sleep(time.Minute * 10)
	}
}
