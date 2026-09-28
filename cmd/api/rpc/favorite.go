package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	favoritepb "douyin/kitex/kitex_gen/favorite"
	"douyin/kitex/kitex_gen/favorite/favoriteservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var favoriteClient favoriteservice.Client

func InitFavorite() error {
	config := appViper.Init("favorite")

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

	rpcClient, err := favoriteservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	favoriteClient = rpcClient

	return nil
}

func FavoriteAction(
	ctx context.Context,
	request *favoritepb.FavoriteActionRequest,
) (*favoritepb.FavoriteActionResponse, error) {
	if favoriteClient == nil {
		return nil, errors.New(
			"favorite RPC client is not initialized",
		)
	}

	return favoriteClient.FavoriteAction(
		ctx,
		request,
	)
}

func FavoriteList(
	ctx context.Context,
	request *favoritepb.FavoriteListRequest,
) (*favoritepb.FavoriteListResponse, error) {
	if favoriteClient == nil {
		return nil, errors.New(
			"favorite RPC client is not initialized",
		)
	}

	return favoriteClient.FavoriteList(
		ctx,
		request,
	)
}
