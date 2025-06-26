package main

import (
	"danmu-core/core"
	"danmu-core/internal/model"
	"danmu-core/internal/server"
	"danmu-core/logger"
	"os"
	"os/signal"
	"syscall"
)

func init() {

}

func main() {
	core.InitTaskManager()
	rpcserver := server.NewRPCServer()
	err := rpcserver.Start()
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
