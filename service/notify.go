package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"go.uber.org/zap"
)

type NotifyServer interface {
	SendNotify(name string, message map[string]interface{}) error
}

type notifyServer struct {
}

func (i *notifyServer) SendNotify(name string, message map[string]interface{}) error {
	msg := make(map[string]string)
	for k, v := range message {
		bt, _ := json.Marshal(v)
		msg[k] = string(bt)
	}
	response, err := MyService.MessageBus().PublishEventWithResponse(context.Background(), common.ServiceName, name, msg)
	if err != nil {
		logger.Error("failed to publish event to message bus", zap.Error(err), zap.Any("event", msg))
		return err
	}
	if response.StatusCode() != http.StatusOK {
		logger.Error("failed to publish event to message bus", zap.String("status", response.Status()), zap.Any("response", response))
	}
	// SocketServer.BroadcastToRoom("/", "public", path, message)
	return nil
}

func NewNotifyService() NotifyServer {
	return &notifyServer{}
}
