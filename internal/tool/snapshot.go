package tool

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func GetSnapshotImageBuffer(
	videoPath string,
	frameNumber int,
) (*bytes.Buffer, error) {
	if videoPath == "" {
		return nil, errors.New("video path cannot be empty")
	}

	if frameNumber < 0 {
		return nil, errors.New("frame number cannot be negative")
	}

	imageBuffer := bytes.NewBuffer(nil)

	err := ffmpeg.
		Input(videoPath).
		Filter(
			"select",
			ffmpeg.Args{
				fmt.Sprintf("gte(n,%d)", frameNumber),
			},
		).
		Output(
			"pipe:",
			ffmpeg.KwArgs{
				"vframes": 1,
				"format":  "image2pipe",
				"vcodec":  "mjpeg",
			},
		).
		WithOutput(imageBuffer).
		WithErrorOutput(os.Stderr).
		Run()

	if err != nil {
		return nil, fmt.Errorf(
			"generate video snapshot failed: %w",
			err,
		)
	}

	if imageBuffer.Len() == 0 {
		return nil, errors.New("generated snapshot is empty")
	}

	return imageBuffer, nil
}
