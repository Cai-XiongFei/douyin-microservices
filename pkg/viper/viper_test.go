package viper

import "testing"

func TestReadDBConfig(t *testing.T) {
	config := Init("db")

	port := config.Viper.GetInt("mysql.source.port")
	if port != 3309 {
		t.Fatalf("期望 MySQL 端口为 3309，实际为 %d", port)
	}
}

func TestReadUserConfig(t *testing.T) {
	config := Init("user")

	serviceName := config.Viper.GetString("server.name")
	if serviceName != "TiktokUserServer" {
		t.Fatalf("用户服务名称错误：%s", serviceName)
	}
}

func TestReadAPIConfig(t *testing.T) {
	config := Init("api")

	port := config.Viper.GetInt("server.port")
	if port != 18089 {
		t.Fatalf("期望 API 端口为 8089，实际为 %d", port)
	}
}

func TestReadMinIOConfig(t *testing.T) {
	config := Init("minio")

	endpoint := config.Viper.GetString("Minio.Endpoint")
	if endpoint != "127.0.0.1:19000" {
		t.Fatalf("MinIO 地址错误：%s", endpoint)
	}
}
