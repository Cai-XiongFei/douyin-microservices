package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	messagepb "douyin/kitex/kitex_gen/message"
	"douyin/kitex/kitex_gen/message/messageservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var messageClient messageservice.Client

func InitMessage() error {
	config := appViper.Init("message")

	etcdAddress := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("etcd.host"),
		config.Viper.GetInt("etcd.port"),
	)
	serviceName := config.Viper.GetString("server.name")

	resolver, err := appEtcd.NewEtcdResolver([]string{etcdAddress})
	if err != nil {
		return err
	}

	rpcClient, err := messageservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	messageClient = rpcClient
	return nil
}

func MessageAction(
	ctx context.Context,
	request *messagepb.MessageActionRequest,
) (*messagepb.MessageActionResponse, error) {
	if messageClient == nil {
		return nil, errors.New("message RPC client is not initialized")
	}

	return messageClient.MessageAction(ctx, request)
}

func MessageChat(
	ctx context.Context,
	request *messagepb.MessageChatRequest,
) (*messagepb.MessageChatResponse, error) {
	if messageClient == nil {
		return nil, errors.New("message RPC client is not initialized")
	}

	return messageClient.MessageChat(ctx, request)
}
