package response

import (
	relationpb "douyin/kitex/kitex_gen/relation"
	userpb "douyin/kitex/kitex_gen/user"
)

type RelationAction struct {
	Base
}

type RelationFollowList struct {
	Base
	UserList []*userpb.User `json:"user_list"`
}

type RelationFollowerList struct {
	Base
	UserList []*userpb.User `json:"user_list"`
}

type RelationFriendList struct {
	Base
	UserList []*relationpb.FriendUser `json:"user_list"`
}
