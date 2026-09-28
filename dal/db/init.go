package db

import (
	"fmt"
	"time"

	appViper "douyin/pkg/viper"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

var (
	database *gorm.DB
	config   = appViper.Init("db")
)

func getDSN(role string) string {
	username := config.Viper.GetString(role + ".username")
	password := config.Viper.GetString(role + ".password")
	host := config.Viper.GetString(role + ".host")
	port := config.Viper.GetInt(role + ".port")
	databaseName := config.Viper.GetString(role + ".database")

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		databaseName,
	)
}

func init() {
	sourceDSN := getDSN("mysql.source")

	var err error

	database, err = gorm.Open(mysql.Open(sourceDSN), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Info),
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(fmt.Sprintf("connect MySQL failed: %v", err))
	}

	replica1DSN := getDSN("mysql.replica1")
	replica2DSN := getDSN("mysql.replica2")

	err = database.Use(dbresolver.Register(dbresolver.Config{
		Sources: []gorm.Dialector{
			mysql.Open(sourceDSN),
		},
		Replicas: []gorm.Dialector{
			mysql.Open(replica1DSN),
			mysql.Open(replica2DSN),
		},
		Policy: dbresolver.RandomPolicy{},
	}))
	if err != nil {
		panic(fmt.Sprintf("闂備焦婢樼粔鍫曟偪閸℃稑鏋侀柣妤€鐗嗙粊锕傚箹鐎涙ɑ宕勬い鏇氬嵆瀹曟ê鈻庤箛鎾垛偓鑽ょ磼閸屾繍鍤欓柕鍥ㄥ灩閹峰綊濡烽婊呯崶%v", err))
	}

	if err := database.AutoMigrate(
		&User{},
		&Video{},
		&Favorite{},
		&Comment{},
		&Relation{},
		&Message{},
		&OutboxEvent{},
		&ProcessedEvent{},
	); err != nil {
		panic(fmt.Sprintf(
			"migrate database tables failed: %v",
			err,
		))
	}

	sqlDB, err := database.DB()
	if err != nil {
		panic(fmt.Sprintf("get underlying database connection failed: %v", err))
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(60 * time.Minute)
}

func GetDB() *gorm.DB {
	return database
}
