package main

import (
	"context"
	"fmt"
	"log"
	"net"

	commentconsumer "douyin/cmd/comment/consumer"
	"douyin/cmd/comment/service"
	"douyin/kitex/kitex_gen/comment/commentservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	config := appViper.Init("comment")

	serviceName := config.Viper.GetString(
		"server.name",
	)

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

	registry, err := appEtcd.NewEtcdRegistry(
		[]string{etcdAddress},
	)
	if err != nil {
		log.Fatalf(
			"create etcd registry failed: %v",
			err,
		)
	}

	address, err := net.ResolveTCPAddr(
		"tcp",
		serviceAddress,
	)
	if err != nil {
		log.Fatalf(
			"resolve service address failed: %v",
			err,
		)
	}

	handler := service.NewCommentServiceImpl(
		[]byte(
			config.Viper.GetString("JWT.signingKey"),
		),
	)

	svr := commentservice.NewServer(
		handler,
		server.WithServiceAddr(address),
		server.WithRegistry(registry),
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			},
		),
	)

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		if err := commentconsumer.ConsumeCommentEvents(consumerCtx); err != nil {
			log.Printf("comment event consumer stopped: %v", err)
		}
	}()

	log.Printf(
		"comment RPC service is starting: name=%s address=%s",
		serviceName,
		serviceAddress,
	)

	if err := svr.Run(); err != nil {
		log.Fatalf(
			"comment RPC service stopped: %v",
			err,
		)
	}
}
