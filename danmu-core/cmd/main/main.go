package main

import (
	"danmu-core/conf"
	"danmu-core/core"
	"danmu-core/internal/model"
	"danmu-core/internal/server"
	"danmu-core/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func init() {

}

func main() {
	// 1. 加载配置
	config, err := conf.GetGlobalConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志
	logger.Init(config.Log)

	// 3. 初始化数据库
	if err := model.Init(config.Database); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	core.InitTaskManager()
	rpcserver := server.NewRPCServer(config.Rpc)
	err = rpcserver.Start()
	if err != nil {
		logger.Fatal().Err(err).Msg("rpc-server start fail")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info().Msg("Shutting down server...")

	rpcserver.Stop()
	model.Close()

	logger.Info().Msg("Server exited")
}
