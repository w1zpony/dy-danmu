package conf

import (
	"log"

	"github.com/spf13/viper"
)

type Task struct {
	RoomDisplayID string `mapstructure:"room_display_id"`
	URL           string `mapstructure:"url"`
	Name          string `mapstructure:"name"`
	Cron          string `mapstructure:"cron"`
	Enable        bool   `mapstructure:"enable"`
}

func GetAllTasks() ([]Task, error) {
	v := viper.New()
	v.SetConfigFile("conf/tasks.yaml")
	v.SetConfigType("yaml")

	log.Printf("GetAllTasks load config from: conf/tasks.yaml")
	if err := v.ReadInConfig(); err != nil {
		log.Printf("GetAllTasks failure, error: %v", err)
		return nil, err
	}

	var taskMap map[string]Task
	if err := v.Unmarshal(&taskMap); err != nil {
		log.Printf("GetAllTasks unmarshal failure, error: %v", err)
		return nil, err
	}

	tasks := make([]Task, 0, len(taskMap))
	for name, task := range taskMap {
		task.Name = name
		tasks = append(tasks, task)
	}

	return tasks, nil
}
