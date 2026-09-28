package response

import messagepb "douyin/kitex/kitex_gen/message"

type MessageAction struct {
	Base
}

type MessageChat struct {
	Base
	MessageList []*messagepb.Message `json:"message_list"`
}
