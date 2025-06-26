package logger

import (
	"danmu-core/setting"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger zerolog.Logger

func init() {
	// 创建日志目录
	if err := os.MkdirAll(setting.LogSetting.LogSavePath, 0755); err != nil {
		panic(fmt.Sprintf("create log directory failed: %v", err))
	}
	// 配置日志分片
	fileLogger := &lumberjack.Logger{
		Filename:   filepath.Join(setting.LogSetting.LogSavePath, setting.LogSetting.LogFileName),
		MaxSize:    setting.LogSetting.MaxSize,
		MaxBackups: setting.LogSetting.MaxBackups,
		MaxAge:     setting.LogSetting.MaxAge,
		Compress:   setting.LogSetting.Compress,
	}
	// 设置日志级别
	level := getLogLevel(setting.LogSetting.LogLevel)
	// 设置时间格式
	zerolog.TimeFieldFormat = setting.LogSetting.TimeFormat
	var writer zerolog.LevelWriter
	if level == zerolog.DebugLevel {
		// 配置带颜色的控制台输出
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: setting.LogSetting.TimeFormat,
			NoColor:    false,
		}
		// 同时输出到控制台和文件
		writer = zerolog.MultiLevelWriter(consoleWriter, fileLogger)
	} else {
		writer = zerolog.MultiLevelWriter(fileLogger)
	}

	// 初始化全局日志记录器
	Logger = zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	// 替换 zerolog 默认的全局 logger
	log.Logger = Logger
}

// getLogLevel 将字符串转换为 zerolog 日志级别
func getLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug 输出调试日志
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info 输出信息日志
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn 输出警告日志
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error 输出错误日志
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal 输出致命错误日志并退出程序
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// logger.Info().
//     Str("key", "value").
//     Int("count", 42).
//     Msg("info message")
//
// logger.Error().
//     Err(err).
//     Str("manager", "api").
//     Msg("manager error")
