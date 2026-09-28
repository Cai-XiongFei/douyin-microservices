package viper

import (
	"log"

	V "github.com/spf13/viper"
)

type Config struct {
	Viper *V.Viper
}

func Init(configName string) Config {
	config := Config{
		Viper: V.New(),
	}

	v := config.Viper

	v.SetConfigType("yml")
	v.SetConfigName(configName)

	v.AddConfigPath("./config")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")
	v.AddConfigPath("../../../config")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件 %s 失败：%v", configName, err)
	}

	return config
}
