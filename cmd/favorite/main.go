package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	favoriteconsumer "douyin/cmd/favorite/consumer"
	"douyin/cmd/favorite/service"
	"douyin/internal/outbox"
	"douyin/kitex/kitex_gen/favorite/favoriteservice"
	appEtcd "douyin/pkg/etcd"
	appViper "douyin/pkg/viper"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	config := appViper.Init("favorite")

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

	handler := service.NewFavoriteServiceImpl(
		[]byte(
			config.Viper.GetString("JWT.signingKey"),
		),
	)
	outboxConfig := outbox.DefaultConfig()
	if value := config.Viper.GetInt("outbox.batch_size"); value > 0 {
		outboxConfig.BatchSize = value
	}
	if value := config.Viper.GetInt("outbox.poll_interval_ms"); value > 0 {
		outboxConfig.PollInterval = time.Duration(value) * time.Millisecond
	}
	if value := config.Viper.GetInt("outbox.lease_seconds"); value > 0 {
		outboxConfig.LeaseDuration = time.Duration(value) * time.Second
	}
	if value := config.Viper.GetInt("outbox.publish_timeout_seconds"); value > 0 {
		outboxConfig.PublishTimeout = time.Duration(value) * time.Second
	}
	if value := config.Viper.GetInt("outbox.max_retries"); value > 0 {
		outboxConfig.MaxRetries = value
	}
	if value := config.Viper.GetInt("outbox.base_retry_delay_ms"); value > 0 {
		outboxConfig.BaseRetryDelay = time.Duration(value) * time.Millisecond
	}
	if value := config.Viper.GetInt("outbox.max_retry_delay_ms"); value > 0 {
		outboxConfig.MaxRetryDelay = time.Duration(value) * time.Millisecond
	}

	outboxRelay, err := outbox.NewRelay(outboxConfig)
	if err != nil {
		log.Fatalf(
			"create outbox relay failed: %v",
			err,
		)
	}

	svr := favoriteservice.NewServer(
		handler,
		server.WithServiceAddr(address),
		server.WithRegistry(registry),
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			},
		),
	)

	consumerCtx, cancelConsumer := context.WithCancel(
		context.Background(),
	)
	defer cancelConsumer()

	go func() {
		for {
			err := favoriteconsumer.ConsumeFavoriteEvents(
				consumerCtx,
			)
			if consumerCtx.Err() != nil {
				return
			}

			log.Printf(
				"favorite event consumer stopped and will restart: %v",
				err,
			)

			select {
			case <-consumerCtx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()

	go outboxRelay.Run(consumerCtx)

	log.Printf(
		"favorite RPC service is starting: name=%s address=%s",
		serviceName,
		serviceAddress,
	)

	if err := svr.Run(); err != nil {
		log.Fatalf(
			"favorite RPC service stopped: %v",
			err,
		)
	}
}
