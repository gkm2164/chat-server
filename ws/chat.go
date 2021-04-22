package ws

import (
	"chat-server/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"
)

type AllUsers map[*websocket.Conn]string

var users = AllUsers{}

var msgCh chan ChatMessageDetail

func init() {
	msgCh = make(chan ChatMessageDetail)
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

	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	if _, ok := users[conn]; !ok {
		users[conn] = util.RandStringBytes(8)
	}

	name := users[conn]
	msgCh <- JoinMessage{
		Name: name,
		Conn: conn,
	}

	defer func(conn *websocket.Conn, name string) {
		msgCh <- LeaveMessage{
			Name: name,
			Conn: conn,
		}
	}(conn, name)

	for {
		msg := ""
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		msgCh <- MessageSend{
			Sender: name,
			Message: msg,
		}
	}
}

type ChatMessageDetail interface{}

type JoinMessage struct {
	Name string
	Conn *websocket.Conn
}

type MessageSend struct {
	Sender  string
	Message string
}

type LeaveMessage struct {
	Name string
	Conn *websocket.Conn
}

func Now() string {
	return time.Now().Format("15:04:05")
}

func broadcastMessage(m map[string]*websocket.Conn, message string) {
	for _, conn := range m {
		if err := conn.WriteJSON(message); err != nil {
			continue
		}
	}
}

func Handler(msgCh chan ChatMessageDetail) {
	var m = map[string]*websocket.Conn{}

	for msg := range msgCh {
		switch msg.(type) {
		case JoinMessage:
			joinMsg := msg.(JoinMessage)
			m[joinMsg.Name] = joinMsg.Conn
			broadcastMessage(m, fmt.Sprintf("[%s] %s joined room", Now(), joinMsg.Name))
			break
		case MessageSend:
			sendMsg := msg.(MessageSend)
			if strings.HasPrefix(sendMsg.Message, "/") {
				conn := m[sendMsg.Sender]
				var message string
				if ret, err := parseMessage(sendMsg.Message, m); err != nil {
					message = fmt.Sprintf("error while parse your command: %v", err)
				} else {
					message = ret
				}

				if err := conn.WriteJSON(fmt.Sprintf("SYSTEM: %s", message)); err != nil {
					break
				}
			} else {
				broadcastMessage(m, fmt.Sprintf("[%s] %s: %s", Now(), sendMsg.Sender, sendMsg.Message))
			}
			break
		case LeaveMessage:
			leaveMsg := msg.(LeaveMessage)
			delete(m, leaveMsg.Name)
			broadcastMessage(m, fmt.Sprintf("[%s] %s left room", Now(), leaveMsg.Name))
			break
		default:
			fmt.Printf("unknown message: %v", msg)
		}
	}
}

func parseMessage(message string, m map[string]*websocket.Conn) (string, error) {
	var command = ""
	for i := 1; i < len(message); i++ {
		if message[i] == ' ' {
			command = message[1:i]
			message = message[i + 1:]
		}
	}

	if command == "" {
		command = message[1:]
	}

	switch command {
	case "members":
		var members []string
		for member, _ := range m {
			members = append(members, member)
		}
		return fmt.Sprintf("[%s] List of members %d: [%s]",
			Now(), len(members), strings.Join(members, ", ")), nil
	case "whisper":
		fallthrough
	default:
		return "", fmt.Errorf("unknown command: %v", command)
	}
}
