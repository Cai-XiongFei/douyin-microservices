package response

import videopb "douyin/kitex/kitex_gen/video"

type FavoriteAction struct {
	Base
}

type FavoriteList struct {
	Base
	VideoList []*videopb.Video `json:"video_list"`
}
