package tool

import (
	"bytes"
	"image/jpeg"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetSnapshotImageBuffer(t *testing.T) {
	tempDirectory := t.TempDir()
	videoPath := filepath.Join(tempDirectory, "test.mp4")

	command := exec.Command(
		"ffmpeg",
		"-y",
		"-f",
		"lavfi",
		"-i",
		"color=c=blue:s=320x240:d=1",
		"-pix_fmt",
		"yuv420p",
		videoPath,
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"create test video failed: %v\n%s",
			err,
			string(output),
		)
	}

	imageBuffer, err := GetSnapshotImageBuffer(
		videoPath,
		1,
	)
	if err != nil {
		t.Fatalf("generate snapshot failed: %v", err)
	}

	if imageBuffer.Len() == 0 {
		t.Fatal("snapshot is empty")
	}

	_, err = jpeg.Decode(
		bytes.NewReader(imageBuffer.Bytes()),
	)
	if err != nil {
		t.Fatalf(
			"snapshot is not a valid JPEG image: %v",
			err,
		)
	}
}
