package service

import (
	"fmt"
	"image/jpeg"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	storage "douyin/pkg/minio"
)

func TestVideoPublish(t *testing.T) {
	tempDirectory := t.TempDir()
	localVideoPath := filepath.Join(
		tempDirectory,
		"test.mp4",
	)

	// 生成一个一秒钟的蓝色视频。
	command := exec.Command(
		"ffmpeg",
		"-y",
		"-f",
		"lavfi",
		"-i",
		"color=c=blue:s=320x240:d=1",
		"-pix_fmt",
		"yuv420p",
		localVideoPath,
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"create test video failed: %v\n%s",
			err,
			string(output),
		)
	}

	videoData, err := os.ReadFile(localVideoPath)
	if err != nil {
		t.Fatalf("read test video failed: %v", err)
	}

	timestamp := time.Now().UnixNano()

	videoName := fmt.Sprintf(
		"tests/video-%d.mp4",
		timestamp,
	)

	coverName := fmt.Sprintf(
		"tests/cover-%d.jpg",
		timestamp,
	)

	defer func() {
		_ = storage.DeleteFile(
			storage.VideoBucketName,
			videoName,
		)

		_ = storage.DeleteFile(
			storage.CoverBucketName,
			coverName,
		)
	}()

	err = VideoPublish(
		videoData,
		videoName,
		coverName,
	)
	if err != nil {
		t.Fatalf("publish video files failed: %v", err)
	}

	videoURL, err := storage.GetFileTemporaryURL(
		storage.VideoBucketName,
		videoName,
	)
	if err != nil {
		t.Fatalf("get video URL failed: %v", err)
	}

	videoResponse, err := http.Get(videoURL)
	if err != nil {
		t.Fatalf("download video failed: %v", err)
	}
	defer videoResponse.Body.Close()

	if videoResponse.StatusCode != http.StatusOK {
		t.Fatalf(
			"unexpected video status: %d",
			videoResponse.StatusCode,
		)
	}

	coverURL, err := storage.GetFileTemporaryURL(
		storage.CoverBucketName,
		coverName,
	)
	if err != nil {
		t.Fatalf("get cover URL failed: %v", err)
	}

	coverResponse, err := http.Get(coverURL)
	if err != nil {
		t.Fatalf("download cover failed: %v", err)
	}
	defer coverResponse.Body.Close()

	if coverResponse.StatusCode != http.StatusOK {
		t.Fatalf(
			"unexpected cover status: %d",
			coverResponse.StatusCode,
		)
	}

	_, err = jpeg.Decode(coverResponse.Body)
	if err != nil {
		t.Fatalf(
			"uploaded cover is not a valid JPEG: %v",
			err,
		)
	}
}
