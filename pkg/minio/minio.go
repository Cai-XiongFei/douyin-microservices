package minio

import (
	"bytes"
	"context"
	"errors"
	"time"

	minioSDK "github.com/minio/minio-go/v7"
)

func CreateBucket(bucketName string) error {
	if bucketName == "" {
		return errors.New("bucket name cannot be empty")
	}

	ctx := context.Background()

	err := minioClient.MakeBucket(
		ctx,
		bucketName,
		minioSDK.MakeBucketOptions{},
	)
	if err == nil {
		return nil
	}

	exists, existsErr := minioClient.BucketExists(ctx, bucketName)
	if existsErr != nil {
		return existsErr
	}

	if exists {
		return nil
	}

	return err
}

func GetFileTemporaryURL(
	bucketName string,
	objectName string,
) (string, error) {
	if bucketName == "" || objectName == "" {
		return "", errors.New("bucket name and object name cannot be empty")
	}

	expiry := time.Duration(ExpireTime) * time.Second

	url, err := minioClient.PresignedGetObject(
		context.Background(),
		bucketName,
		objectName,
		expiry,
		nil,
	)
	if err != nil {
		return "", err
	}

	return url.String(), nil
}

func UploadFile(
	bucketName string,
	objectName string,
	data []byte,
	contentType string,
) error {
	if bucketName == "" {
		return errors.New("bucket name cannot be empty")
	}

	if objectName == "" {
		return errors.New("object name cannot be empty")
	}

	if len(data) == 0 {
		return errors.New("file data cannot be empty")
	}

	_, err := minioClient.PutObject(
		context.Background(),
		bucketName,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minioSDK.PutObjectOptions{
			ContentType: contentType,
		},
	)

	return err
}

func DeleteFile(
	bucketName string,
	objectName string,
) error {
	if bucketName == "" || objectName == "" {
		return errors.New("bucket name and object name cannot be empty")
	}

	return minioClient.RemoveObject(
		context.Background(),
		bucketName,
		objectName,
		minioSDK.RemoveObjectOptions{},
	)
}
