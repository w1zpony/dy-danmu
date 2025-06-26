package main

import (
	"danmu-core/core"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	/*	conf := &model.LiveConf{
			Name:   "test",
			URL:    "https://live.douyin.com/84062371649",
			Enable: true,
		}
		handler, _ := handler.NewDymsg2dbHandler(conf)
		c := core.MakeClient(conf)
		c.Subscribe(handler)
		c.Start()*/
	core.InitTaskManager()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-quit
}
