package redis

import (
	"context"
	"fmt"
	"log"

	appViper "douyin/pkg/viper"

	redisSDK "github.com/redis/go-redis/v9"
)

var Client *redisSDK.Client

func init() {
	config := appViper.Init("redis")

	address := fmt.Sprintf(
		"%s:%d",
		config.Viper.GetString("redis.host"),
		config.Viper.GetInt("redis.port"),
	)

	Client = redisSDK.NewClient(&redisSDK.Options{
		Addr:     address,
		Password: config.Viper.GetString("redis.password"),
		DB:       config.Viper.GetInt("redis.database"),
	})

	if err := Client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("连接 Redis 失败：%v", err)
	}

	log.Printf("Redis connected successfully: %s", address)
}

func GetClient() *redisSDK.Client {
	return Client
}
