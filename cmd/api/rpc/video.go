package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"douyin/kitex/kitex_gen/video"
	videopb "douyin/kitex/kitex_gen/video"
	"douyin/kitex/kitex_gen/video/videoservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var videoClient videoservice.Client

func InitVideo() error {
	config := appViper.Init("video")

	etcdAddress := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("etcd.host"),
		config.Viper.GetInt("etcd.port"),
	)

	serviceName := config.Viper.GetString("server.name")

	resolver, err := appEtcd.NewEtcdResolver(
		[]string{etcdAddress},
	)
	if err != nil {
		return err
	}

	rpcClient, err := videoservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Minute),
	)
	if err != nil {
		return err
	}

	videoClient = rpcClient
	return nil
}

func PublishList(
	ctx context.Context,
	request *videopb.PublishListRequest,
) (*videopb.PublishListResponse, error) {
	return videoClient.PublishList(ctx, request)
}

func PublishAction(
	ctx context.Context,
	request *videopb.PublishActionRequest,
) (*videopb.PublishActionResponse, error) {
	return videoClient.PublishAction(ctx, request)
}

func Feed(ctx context.Context, req *video.FeedRequest) (*video.FeedResponse, error) {
	if videoClient == nil {
		return nil, errors.New("video RPC client is not initialized")
	}

	return videoClient.Feed(ctx, req)
}
