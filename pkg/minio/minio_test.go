package minio

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestUploadFile(t *testing.T) {
	objectName := fmt.Sprintf(
		"tests/upload-%d.txt",
		time.Now().UnixNano(),
	)

	fileData := []byte("hello minio")

	err := UploadFile(
		VideoBucketName,
		objectName,
		fileData,
		"text/plain",
	)
	if err != nil {
		t.Fatalf("upload file failed: %v", err)
	}

	defer func() {
		if err := DeleteFile(VideoBucketName, objectName); err != nil {
			t.Errorf("delete test file failed: %v", err)
		}
	}()

	fileURL, err := GetFileTemporaryURL(
		VideoBucketName,
		objectName,
	)
	if err != nil {
		t.Fatalf("get temporary URL failed: %v", err)
	}

	response, err := http.Get(fileURL)
	if err != nil {
		t.Fatalf("download uploaded file failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"unexpected HTTP status: %d",
			response.StatusCode,
		)
	}

	downloadedData, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read downloaded file failed: %v", err)
	}

	if !bytes.Equal(downloadedData, fileData) {
		t.Fatalf(
			"file contents do not match: got %q, want %q",
			downloadedData,
			fileData,
		)
	}
}
