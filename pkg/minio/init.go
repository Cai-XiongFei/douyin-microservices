package minio

import (
	appViper "douyin/pkg/viper"

	minioSDK "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	minioClient *minioSDK.Client
	config      = appViper.Init("minio")

	Endpoint                  = config.Viper.GetString("Minio.Endpoint")
	AccessKeyID               = config.Viper.GetString("Minio.AccessKeyId")
	SecretAccessKey           = config.Viper.GetString("Minio.SecretAccessKey")
	UseSSL                    = config.Viper.GetBool("Minio.UseSSL")
	VideoBucketName           = config.Viper.GetString("Minio.VideoBucketName")
	CoverBucketName           = config.Viper.GetString("Minio.CoverBucketName")
	AvatarBucketName          = config.Viper.GetString("Minio.AvatarBucketName")
	BackgroundImageBucketName = config.Viper.GetString("Minio.BackgroundImageBucketName")
	ExpireTime                = config.Viper.GetUint32("Minio.ExpireTime")
)

func init() {
	client, err := minioSDK.New(Endpoint, &minioSDK.Options{
		Creds: credentials.NewStaticV4(
			AccessKeyID,
			SecretAccessKey,
			"",
		),
		Secure: UseSSL,
	})
	if err != nil {
		panic(err)
	}

	minioClient = client

	buckets := []string{
		VideoBucketName,
		CoverBucketName,
		AvatarBucketName,
		BackgroundImageBucketName,
	}

	for _, bucket := range buckets {
		if err := CreateBucket(bucket); err != nil {
			panic(err)
		}
	}
}
