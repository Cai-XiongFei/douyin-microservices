package service

import (
	"fmt"

	"douyin/internal/tool"
	storage "douyin/pkg/minio"
)

func uploadVideo(
	data []byte,
	videoName string,
) (string, error) {
	err := storage.UploadFile(
		storage.VideoBucketName,
		videoName,
		data,
		"video/mp4",
	)
	if err != nil {
		return "", fmt.Errorf(
			"upload video to MinIO failed: %w",
			err,
		)
	}

	videoURL, err := storage.GetFileTemporaryURL(
		storage.VideoBucketName,
		videoName,
	)
	if err != nil {
		// 临时地址生成失败时，删除已经上传的视频。
		_ = storage.DeleteFile(
			storage.VideoBucketName,
			videoName,
		)

		return "", fmt.Errorf(
			"get temporary video URL failed: %w",
			err,
		)
	}

	return videoURL, nil
}

func uploadCover(
	videoURL string,
	coverName string,
) error {
	coverBuffer, err := tool.GetSnapshotImageBuffer(
		videoURL,
		1,
	)
	if err != nil {
		return fmt.Errorf(
			"generate video cover failed: %w",
			err,
		)
	}

	err = storage.UploadFile(
		storage.CoverBucketName,
		coverName,
		coverBuffer.Bytes(),
		"image/jpeg",
	)
	if err != nil {
		return fmt.Errorf(
			"upload cover to MinIO failed: %w",
			err,
		)
	}

	return nil
}

func VideoPublish(
	data []byte,
	videoName string,
	coverName string,
) error {
	if len(data) == 0 {
		return fmt.Errorf("video data cannot be empty")
	}

	videoURL, err := uploadVideo(
		data,
		videoName,
	)
	if err != nil {
		return err
	}

	err = uploadCover(
		videoURL,
		coverName,
	)
	if err != nil {
		// 封面生成或上传失败，清理已经上传的视频。
		_ = storage.DeleteFile(
			storage.VideoBucketName,
			videoName,
		)

		return err
	}

	return nil
}
