package main

import (
	"context"
	"fmt"
	"log"
	"net"

	videoconsumer "douyin/cmd/video/consumer"
	"douyin/cmd/video/service"
	"douyin/kitex/kitex_gen/video/videoservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	config := appViper.Init("video")

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

	r, err := appEtcd.NewEtcdRegistry([]string{etcdAddress})
	if err != nil {
		log.Fatalf("create etcd registry failed: %v", err)
	}

	address, err := net.ResolveTCPAddr("tcp", serviceAddress)
	if err != nil {
		log.Fatalf("resolve service address failed: %v", err)
	}

	handler := service.NewVideoServiceImpl(
		[]byte(
			config.Viper.GetString("JWT.signingKey"),
		),
		config.Viper.GetInt64("video.maxSizeLimit"),
	)

	svr := videoservice.NewServer(
		handler,
		server.WithServiceAddr(address),
		server.WithRegistry(r),
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			},
		),
	)

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		if err := videoconsumer.ConsumeVideoEvents(consumerCtx); err != nil {
			log.Printf("video event consumer stopped: %v", err)
		}
	}()

	log.Printf(
		"video RPC service is starting: name=%s address=%s",
		serviceName,
		serviceAddress,
	)

	if err := svr.Run(); err != nil {
		log.Fatalf("video RPC service stopped: %v", err)
	}
}
