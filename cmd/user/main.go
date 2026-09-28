package main

import (
	"fmt"
	"log"
	"net"

	"douyin/cmd/user/service"
	"douyin/kitex/kitex_gen/user/userservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	config := appViper.Init("user")

	serviceName := config.Viper.GetString("server.name")

	serviceAddress := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("server.host"),
		config.Viper.GetInt("server.port"),
	)

	etcdAddress := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("etcd.host"),
		config.Viper.GetInt("etcd.port"),
	)

	signingKey := config.Viper.GetString("JWT.signingKey")

	r, err := appEtcd.NewEtcdRegistry([]string{etcdAddress})
	if err != nil {
		log.Fatalf("create etcd registry failed: %v", err)
	}

	address, err := net.ResolveTCPAddr("tcp", serviceAddress)
	if err != nil {
		log.Fatalf("resolve service address failed: %v", err)
	}

	handler := service.NewUserServiceImpl(signingKey)

	server := userservice.NewServer(
		handler,
		server.WithServiceAddr(address),
		server.WithRegistry(r),
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			},
		),
	)

	log.Printf(
		"user RPC service is starting: name=%s address=%s",
		serviceName,
		serviceAddress,
	)

	if err := server.Run(); err != nil {
		log.Fatalf("user RPC service stopped: %v", err)
	}
}
