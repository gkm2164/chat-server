package ws

import (
	"chat-server/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type AllUsers map[*websocket.Conn]string

var users = AllUsers{}

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

var msgCh chan ChatMessageDetail

func Init(log *logrus.Logger) {
	msgCh = make(chan ChatMessageDetail)
	go Handler(log, msgCh)
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
	msgCh <- MessageSend{
		Action: MembersAction,
		Message: UpdateMembers{
			Members: len(users),
		},
	}

	defer func(conn *websocket.Conn, name string, users int) {
		msgCh <- LeaveMessage{
			Name: name,
			Conn: conn,
		}
		msgCh <- MessageSend{
			Action: MembersAction,
			Message: UpdateMembers{
				Members: users,
			},
		}
	}(conn, name, len(users))

	for {
		msg := ""
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		msgCh <- MessageSend{
			Action: MessageAction,
			Message: BroadcastMessage{
				Sender:  name,
				Message: msg,
			},
		}
	}
}

func Now() string {
	return time.Now().Format("15:04:05")
}

func broadcastMessage(m map[string]*websocket.Conn, message interface{}) {
	for _, conn := range m {
		if err := conn.WriteJSON(message); err != nil {
			continue
		}
	}
}

func Handler(log *logrus.Logger, msgCh chan ChatMessageDetail) {
	var m = map[string]*websocket.Conn{}

	for msg := range msgCh {
		switch msg.(type) {
		case JoinMessage:
			joinMsg := msg.(JoinMessage)
			m[joinMsg.Name] = joinMsg.Conn
			log.Infof("joined: %s", joinMsg.Name)
			broadcastMessage(m, fmt.Sprintf("[%s] %s joined room", Now(), joinMsg.Name))
			break
		case MessageSend:
			msgSend := msg.(MessageSend)
			switch msgSend.Action {
			case MessageAction:
				sendMsg := msgSend.Message.(BroadcastMessage)
				if strings.HasPrefix(sendMsg.Message, "/") {
					conn := m[sendMsg.Sender]
					var message string
					if ret, err := parseMessage(sendMsg.Message, m); err != nil {
						message = fmt.Sprintf("error while parse your command: %v", err)
					} else {
						message = ret
					}

					if err := conn.WriteJSON(gin.H{
						"action": "message",
						"message": fmt.Sprintf("[%s] SYSTEM: %s", Now(), message),
					}); err != nil {
						break
					}
				} else {
					log.Infof("%s typed [%s]", sendMsg.Sender, sendMsg.Message)
					broadcastMessage(m, gin.H{
						"action": "message",
						"message": fmt.Sprintf("[%s] %s: %s", Now(), sendMsg.Sender, sendMsg.Message),
					})
				}
				break
			case MembersAction:
				_ = msgSend.Message.(UpdateMembers)
				broadcastMessage(m, gin.H{
					"action": "members",
					"members": len(m),
				})
			}
		case LeaveMessage:
			leaveMsg := msg.(LeaveMessage)
			log.Infof("%s left", leaveMsg.Name)
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
			message = message[i+1:]
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
		return fmt.Sprintf("List of members %d: [%s]", len(members), strings.Join(members, ", ")), nil
	case "whisper":
		fallthrough
	default:
		return "", fmt.Errorf("unknown command: %v", command)
	}
}