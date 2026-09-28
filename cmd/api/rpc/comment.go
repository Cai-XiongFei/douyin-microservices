package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	commentpb "douyin/kitex/kitex_gen/comment"
	"douyin/kitex/kitex_gen/comment/commentservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var commentClient commentservice.Client

func InitComment() error {
	config := appViper.Init("comment")

	etcdAddress := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("etcd.host"),
		config.Viper.GetInt("etcd.port"),
	)

	serviceName := config.Viper.GetString(
		"server.name",
	)

	resolver, err := appEtcd.NewEtcdResolver(
		[]string{etcdAddress},
	)
	if err != nil {
		return err
	}

	rpcClient, err := commentservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	commentClient = rpcClient

	return nil
}

func CommentAction(
	ctx context.Context,
	request *commentpb.CommentActionRequest,
) (*commentpb.CommentActionResponse, error) {
	if commentClient == nil {
		return nil, errors.New(
			"comment RPC client is not initialized",
		)
	}

	return commentClient.CommentAction(
		ctx,
		request,
	)
}

func CommentList(
	ctx context.Context,
	request *commentpb.CommentListRequest,
) (*commentpb.CommentListResponse, error) {
	if commentClient == nil {
		return nil, errors.New(
			"comment RPC client is not initialized",
		)
	}

	return commentClient.CommentList(
		ctx,
		request,
	)
}
