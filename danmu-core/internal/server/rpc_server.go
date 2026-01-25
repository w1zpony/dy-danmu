package server

import (
	"danmu-core/conf"
	"danmu-core/generated/api"
	"danmu-core/internal/service"
	"danmu-core/logger"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type RPCServer struct {
	server *grpc.Server
	cfg    *conf.Rpc
}

func NewRPCServer(cfg *conf.Rpc) *RPCServer {
	// 创建 gRPC 服务器选项
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: time.Duration(cfg.ConnectTimeout) * time.Second,
			MaxConnectionAge:  time.Duration(cfg.ConnectTimeout) * time.Second,
			Time:              time.Duration(cfg.ConnectTimeout) * time.Second,
			Timeout:           time.Duration(cfg.ConnectTimeout) * time.Second,
		}),
		grpc.MaxConcurrentStreams(cfg.MaxOpenConns),
	}

	// 创建 gRPC 服务器
	server := grpc.NewServer(opts...)

	return &RPCServer{
		server: server,
		cfg:    cfg,
	}
}

func (s *RPCServer) Start() error {
	// 注册服务
	api.RegisterLiveServiceServer(s.server, service.NewLiveServer())

	// 创建监听器
	addr := fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	logger.Info().Str("addr", addr).Msg("starting gRPC server")

	// 启动服务器
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

func (s *RPCServer) Stop() {
	if s.server != nil {
		logger.Info().Msg("stopping gRPC server")
		s.server.GracefulStop()
	}
}
