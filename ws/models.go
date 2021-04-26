package ws

import "github.com/gorilla/websocket"

type AllUsers map[*websocket.Conn]string

type UpdateMembers struct {
	Members int `json:"members"`
}

type BroadcastMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

type ChatMessageDetail interface{}

type JoinMessage struct {
	Name string
	Conn *websocket.Conn
}

type MessageSend struct {
	Action  string      `json:"action"`
	Message interface{} `json:"message"`
}

type LeaveMessage struct {
	Name string
	Conn *websocket.Conn
}

const (
	MembersAction = "members"
	MessageAction = "message"
)
