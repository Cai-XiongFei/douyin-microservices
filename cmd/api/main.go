package main

import (
	"douyin/cmd/api/rpc"
	"douyin/internal/httpapi"
	appjwt "douyin/pkg/jwt"
	appViper "douyin/pkg/viper"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func main() {
	config := appViper.Init("api")

	host := config.Viper.GetString("server.host")
	port := config.Viper.GetInt("server.port")

	if environmentPort := os.Getenv("API_PORT"); environmentPort != "" {
		parsedPort, err := strconv.Atoi(environmentPort)
		if err != nil || parsedPort <= 0 || parsedPort > 65535 {
			log.Fatalf("invalid API_PORT: %s", environmentPort)
		}

		port = parsedPort
	}

	address := fmt.Sprintf("%s:%d", host, port)

	if err := rpc.InitUser(); err != nil {
		log.Fatalf("initialize user RPC client failed: %v", err)
	}

	if err := rpc.InitFavorite(); err != nil {
		log.Fatalf("initialize favorite RPC client failed: %v", err)
	}
	if err := rpc.InitComment(); err != nil {
		log.Fatalf(
			"initialize comment RPC client failed: %v", err)
	}
	if err := rpc.InitRelation(); err != nil {
		log.Fatalf(
			"initialize relation RPC client failed: %v", err)
	}
	if err := rpc.InitMessage(); err != nil {
		log.Fatalf(
			"initialize message RPC client failed: %v", err)
	}
	if err := rpc.InitVideo(); err != nil {
		log.Fatalf("initialize video RPC client failed: %v", err)
	}
	maxRequestBodySizeMB := config.Viper.GetInt(
		"server.maxRequestBodySizeMB",
	)

	if maxRequestBodySizeMB <= 0 {
		maxRequestBodySizeMB = 60
	}

	h := server.Default(
		server.WithHostPorts(address),
		server.WithMaxRequestBodySize(maxRequestBodySizeMB*1024*1024),
	)

	jwtManager := appjwt.NewJWT(
		[]byte(config.Viper.GetString("JWT.signingKey")),
	)

	httpapi.RegisterRouters(h)

	// 婵炲鍔岄崬浠嬪箮閺嶎厾鍙惧☉鎾磋壘婵喖骞掗妷銉ョ稉
	registerRoutes(h, jwtManager)

	log.Printf("HTTP API service is starting: %s", address)

	h.Spin()
}
