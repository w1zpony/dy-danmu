package setting

import (
	"flag"
	"github.com/go-ini/ini"
	"log" // 使用标准库的 log
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

var DatabaseSetting = &Database{}

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

var LogSetting = &Log{}

type Rpc struct {
	Host           string
	Port           string
	MaxIdleConns   int
	MaxOpenConns   uint32
	ConnectTimeout int
}

var RpcSetting = &Rpc{}

var cfg *ini.File
var configPath string

func init() {
	flag.StringVar(&configPath, "config", "conf/app.ini", "path to config file")
	flag.Parse()

	var err error
	log.Printf("settingh.Setup load config from: %s", configPath)
	cfg, err = ini.Load(configPath)
	if err != nil {
		log.Fatalf("setting.Setup failure, path: %s, error: %v", configPath, err)
		return
	}

	mapTo("database", DatabaseSetting)
	mapTo("log", LogSetting)
	mapTo("rpc", RpcSetting)
}

func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("settings.MapTo error, section: %s, error: %v", section, err)
	}
}
