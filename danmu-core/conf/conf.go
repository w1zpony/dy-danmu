package conf

import (
	"flag"
	"log"

	"github.com/spf13/viper"
)

type Database struct {
	Type         string
	User         string
	Password     string
	Host         string
	Port         string
	DBName       string
	SearchPath   string
	MaxIdleConns int
	MaxOpenConns int
}

type Log struct {
	LogSavePath   string
	LogFileName   string
	MaxSize       int
	MaxBackups    int
	MaxAge        int
	Compress      bool
	LogLevel      string
	TimeFormat    string
	ConsoleOutput bool
}

type Rpc struct {
	Host           string
	Port           string
	MaxIdleConns   int
	MaxOpenConns   uint32
	ConnectTimeout int
}

type Config struct {
	Database *Database
	Log      *Log
	Rpc      *Rpc
}

func GetGlobalConfig() (*Config, error) {
	var configPath string
	flag.StringVar(&configPath, "config", "conf/app.yaml", "path to config file")
	flag.Parse()

	log.Printf("GetGlobalConfig load config from: %s", configPath)

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("conf.Setup failure, path: %s, error: %v", configPath, err)
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		log.Printf("conf.Unmarshal failure, error: %v", err)
		return nil, err
	}

	return config, nil
}
