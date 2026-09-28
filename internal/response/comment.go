package response

import commentpb "douyin/kitex/kitex_gen/comment"

type CommentAction struct {
	Base
	Comment *commentpb.Comment `json:"comment,omitempty"`
}

type CommentList struct {
	Base
	CommentList []*commentpb.Comment `json:"comment_list"`
}
