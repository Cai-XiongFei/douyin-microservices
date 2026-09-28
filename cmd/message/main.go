package main

import (
	"context"
	"fmt"
	"log"
	"net"

	messageconsumer "douyin/cmd/message/consumer"
	"douyin/cmd/message/service"
	"douyin/kitex/kitex_gen/message/messageservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	config := appViper.Init("message")

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

	registry, err := appEtcd.NewEtcdRegistry([]string{etcdAddress})
	if err != nil {
		log.Fatalf("create etcd registry failed: %v", err)
	}

	address, err := net.ResolveTCPAddr("tcp", serviceAddress)
	if err != nil {
		log.Fatalf("resolve service address failed: %v", err)
	}

	handler := service.NewMessageServiceImpl(
		[]byte(config.Viper.GetString("JWT.signingKey")),
	)

	svr := messageservice.NewServer(
		handler,
		server.WithServiceAddr(address),
		server.WithRegistry(registry),
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{ServiceName: serviceName},
		),
	)

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		if err := messageconsumer.ConsumeMessageEvents(consumerCtx); err != nil {
			log.Printf("message event consumer stopped: %v", err)
		}
	}()

	log.Printf(
		"message RPC service is starting: name=%s address=%s",
		serviceName,
		serviceAddress,
	)

	if err := svr.Run(); err != nil {
		log.Fatalf("message RPC service stopped: %v", err)
	}
}
