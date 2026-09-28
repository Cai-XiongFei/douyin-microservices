package response

import (
	"douyin/kitex/kitex_gen/video"
	videopb "douyin/kitex/kitex_gen/video"
)

type PublishList struct {
	Base
	VideoList []*videopb.Video `json:"video_list"`
}

type PublishAction struct {
	Base
}

type Feed struct {
	Base
	VideoList []*video.Video `json:"video_list"`
	NextTime  int64          `json:"next_time"`
}
