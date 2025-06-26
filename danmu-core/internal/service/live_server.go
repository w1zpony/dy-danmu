package service

import (
	"context"
	"danmu-core/core"
	"danmu-core/generated/api"
	"danmu-core/internal/model"
)

type LiveServer struct {
	api.UnimplementedLiveServiceServer
}

func NewLiveServer() *LiveServer {
	return &LiveServer{}
}

func (s *LiveServer) AddTask(ctx context.Context, req *api.LiveConf) (*api.Response, error) {
	conf := &model.LiveConf{
		ID:            req.Id,
		URL:           req.Url,
		RoomDisplayID: req.RoomDisplayId,
		Name:          req.Name,
		Enable:        req.Enable,
	}

	if err := core.Add(conf); err != nil {
		return &api.Response{
			Code:    400,
			Message: err.Error(),
		}, nil
	}

	return &api.Response{
		Code:    200,
		Message: "success",
	}, nil
}

func (s *LiveServer) DeleteTask(ctx context.Context, req *api.TaskID) (*api.Response, error) {
	if err := core.Delete(req.Id); err != nil {
		return &api.Response{
			Code:    400,
			Message: err.Error(),
		}, nil
	}

	return &api.Response{
		Code:    200,
		Message: "success",
	}, nil
}

func (s *LiveServer) UpdateTask(ctx context.Context, req *api.LiveConf) (*api.Response, error) {
	conf := &model.LiveConf{
		ID:            req.Id,
		URL:           req.Url,
		RoomDisplayID: req.RoomDisplayId,
		Name:          req.Name,
		Enable:        req.Enable,
	}

	if err := core.Update(conf); err != nil {
		return &api.Response{
			Code:    400,
			Message: err.Error(),
		}, nil
	}

	return &api.Response{
		Code:    200,
		Message: "success",
	}, nil
}
