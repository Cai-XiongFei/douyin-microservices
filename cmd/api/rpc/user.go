package rpc

import (
	"context"
	"fmt"
	"time"

	"douyin/kitex/kitex_gen/user"
	"douyin/kitex/kitex_gen/user/userservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/client"
)

var userClient userservice.Client

func InitUser() error {
	config := appViper.Init("user")

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

	rpcClient, err := userservice.NewClient(
		serviceName,
		client.WithResolver(resolver),
		client.WithConnectTimeout(3*time.Second),
		client.WithRPCTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	userClient = rpcClient
	return nil
}

func Register(
	ctx context.Context,
	request *user.UserRegisterRequest,
) (*user.UserRegisterResponse, error) {
	return userClient.Register(ctx, request)
}

func Login(
	ctx context.Context,
	request *user.UserLoginRequest,
) (*user.UserLoginResponse, error) {
	response, err := userClient.Login(ctx, request)
	if err == nil {
		return response, nil
	}

	// 第一次发生临时网络错误时，再尝试一次。
	return userClient.Login(ctx, request)
}

func UserInfo(
	ctx context.Context,
	request *user.UserInfoRequest,
) (*user.UserInfoResponse, error) {
	return userClient.UserInfo(ctx, request)
}
