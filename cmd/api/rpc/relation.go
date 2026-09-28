package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	relationpb "douyin/kitex/kitex_gen/relation"
	"douyin/kitex/kitex_gen/relation/relationservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var relationClient relationservice.Client

func InitRelation() error {
	config := appViper.Init("relation")

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

	rpcClient, err := relationservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	relationClient = rpcClient
	return nil
}

func RelationAction(
	ctx context.Context,
	request *relationpb.RelationActionRequest,
) (*relationpb.RelationActionResponse, error) {
	if relationClient == nil {
		return nil, errors.New("relation RPC client is not initialized")
	}

	return relationClient.RelationAction(ctx, request)
}

func RelationFollowList(
	ctx context.Context,
	request *relationpb.RelationFollowListRequest,
) (*relationpb.RelationFollowListResponse, error) {
	if relationClient == nil {
		return nil, errors.New("relation RPC client is not initialized")
	}

	return relationClient.RelationFollowList(ctx, request)
}

func RelationFollowerList(
	ctx context.Context,
	request *relationpb.RelationFollowerListRequest,
) (*relationpb.RelationFollowerListResponse, error) {
	if relationClient == nil {
		return nil, errors.New("relation RPC client is not initialized")
	}

	return relationClient.RelationFollowerList(ctx, request)
}

func RelationFriendList(
	ctx context.Context,
	request *relationpb.RelationFriendListRequest,
) (*relationpb.RelationFriendListResponse, error) {
	if relationClient == nil {
		return nil, errors.New("relation RPC client is not initialized")
	}

	return relationClient.RelationFriendList(ctx, request)
}
