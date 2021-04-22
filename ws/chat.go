package ws

import (
	"chat-server/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

type AllUsers map[*websocket.Conn]string

var users = AllUsers{}

var msgCh chan *ChatMessage

func init() {
	msgCh = make(chan *ChatMessage)
	go Handler(msgCh)
}

func Chat(w gin.ResponseWriter, r *http.Request) {
	ws := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	if conn, err := ws.Upgrade(w, r, nil); err != nil {
		fmt.Println(err)
		return
	} else {
		if _, ok := users[conn]; !ok {
			users[conn] = util.RandStringBytes(8)
		}

		name := users[conn]
		msgCh <- &ChatMessage{
			command: "join",
			msg: JoinMessage{
				Name: name,
				Conn: conn,
			},
		}

		defer func(conn *websocket.Conn, name string) {
			msgCh <- &ChatMessage{
				command: "leave",
				msg: LeaveMessage{
					Name: name,
					Conn: conn,
				},
			}
		}(conn, name)

		for {
			msg := ""
			if err := conn.ReadJSON(&msg); err != nil {
				break
			}

			msgCh <- &ChatMessage{
				command: "message",
				msg: MessageSend{
					Message: fmt.Sprintf("%s: %s", name, msg),
				},
			}
		}
	}
}

type ChatMessageDetail interface{}

type JoinMessage struct {
	Name string
	Conn *websocket.Conn
}

type MessageSend struct {
	Message string
}

type LeaveMessage struct {
	Name string
	Conn *websocket.Conn
}

type ChatMessage struct {
	command string
	msg     ChatMessageDetail
}

func Now() string {
	return time.Now().Format("15:04:05")
}

func Handler(msgCh chan *ChatMessage) {
	var m = map[string]*websocket.Conn{}

	for msg := range msgCh {
		switch msg.command {
		case "join":
			joinMsg := msg.msg.(JoinMessage)
			m[joinMsg.Name] = joinMsg.Conn
			for _, conn := range m {
				if err := conn.WriteJSON(fmt.Sprintf("[%s] %s joined room", Now(), joinMsg.Name)); err != nil {
					continue
				}
			}
			break
		case "message":
			sendMsg := msg.msg.(MessageSend)
			for _, conn := range m {
				if err := conn.WriteJSON(sendMsg.Message); err != nil {
					continue
				}
			}
			break
		case "leave":
			leaveMsg := msg.msg.(LeaveMessage)
			delete(m, leaveMsg.Name)
			for _, conn := range m {
				if err := conn.WriteJSON(fmt.Sprintf("[%s] %s left room", Now(), leaveMsg.Name)); err != nil {
					continue
				}
			}
			break
		default:
			fmt.Printf("unknown message: %s", msg.command)
		}
	}
}