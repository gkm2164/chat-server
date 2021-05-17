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

var users = AllUsers{}

var msgCh chan ChatMessageDetail

func Init(log *logrus.Logger) {
	msgCh = make(chan ChatMessageDetail)
	go Handler(log, msgCh)
}

func Upgrade(c *gin.Context) {
	Chat(c.Writer, c.Request)
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
	sendMessage(msgCh, JoinMessage{
		Name: name,
		Conn: conn,
	}, MessageSend{
		Action: MembersAction,
		Message: UpdateMembers{
			Members: len(users),
		},
	})

	defer func(conn *websocket.Conn, name string, users int) {
		sendMessage(msgCh, LeaveMessage{
			Name: name,
			Conn: conn,
		}, MessageSend{
			Action: MembersAction,
			Message: UpdateMembers{
				Members: users,
			},
		})
	}(conn, name, len(users))

	for {
		msg := ""
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		sendMessage(msgCh, MessageSend{
			Action: MessageAction,
			Message: BroadcastMessage{
				Sender:  name,
				Message: msg,
			},
		})
	}
}

func sendMessage(ch chan ChatMessageDetail, message ...ChatMessageDetail) {
	for _, msg := range message {
		ch <- msg
	}
}

func Now() string {
	return time.Now().Format("15:04:05")
}

func broadcastMessage(m map[string]*websocket.Conn, message interface{}) {
	for key, conn := range m {
		if err := conn.WriteJSON(message); err != nil {
			logrus.Errorf("failed to write error: %v", err)
		} else {
			logrus.Infof("send message to %s successfully", key)
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
			if err := joinMsg.Conn.WriteJSON(gin.H{
				"action": MembersCmd,
				"message": fmt.Sprintf("You are joined as [%s]", joinMsg.Name),
			}); err != nil {
				return
			}
			log.Infof("joined: %s", joinMsg.Name)
			broadcastMessage(m, gin.H{
				"action": MembersCmd,
				"message": fmt.Sprintf("[%s] %s joined room", Now(), joinMsg.Name),
			})
			break
		case MessageSend:
			msgSend := msg.(MessageSend)
			switch msgSend.Action {
			case MessageAction:
				sendMsg := msgSend.Message.(BroadcastMessage)
				if strings.HasPrefix(sendMsg.Message, "/") {
					conn := m[sendMsg.Sender]
					var message string
					if ret, err := parseMessage(sendMsg.Sender, sendMsg.Message, m); err != nil {
						message = fmt.Sprintf("error while parse your command: %v", err)
					} else {
						message = ret
					}

					if err := conn.WriteJSON(gin.H{
						"action":  MembersCmd,
						"message": fmt.Sprintf("[%s] SYSTEM: %s", Now(), message),
					}); err != nil {
						break
					}
				} else {
					log.Infof("%s typed [%s]", sendMsg.Sender, sendMsg.Message)
					broadcastMessage(m, gin.H{
						"action":  MembersCmd,
						"message": fmt.Sprintf("[%s] %s: %s", Now(), sendMsg.Sender, sendMsg.Message),
					})
				}
				break
			case MembersAction:
				_ = msgSend.Message.(UpdateMembers)
				broadcastMessage(m, gin.H{
					"action":  MembersCmd,
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
			log.Errorf("unknown message: %v", msg)
		}
	}
}

const (
	MembersCmd = "members"
	WhisperCmd = "whisper"
)

func parseMessage(sender, message string, m map[string]*websocket.Conn) (string, error) {
	message = strings.TrimPrefix(message, "/")
	command, message := splitAtChar(message, ' ')

	switch command {
	case MembersCmd:
		var members []string
		for member, _ := range m {
			members = append(members, member)
		}
		return fmt.Sprintf("List of members %d: [%s]", len(members), strings.Join(members, ", ")), nil
	case WhisperCmd:
		to, message := splitAtChar(message, ' ')
		if conn, ok := m[to]; !ok {
			return "", fmt.Errorf("user %s is not exist", to)
		} else if err := conn.WriteJSON(gin.H{
			"action": MembersCmd,
			"message": fmt.Sprintf("[%s] %s whispered: %s", Now(), sender, message),
		}); err != nil {
			return "", fmt.Errorf("failed to send message to %s", to)
		} else {
			return fmt.Sprintf("sent message to %s, '%s'", to, message), nil
		}
	default:
		return "", fmt.Errorf("unknown command: %v", command)
	}
}

func splitAtChar(str string, ch uint8) (string, string) {
	for i := 0; i < len(str); i++ {
		if str[i] == ch {
			return str[:i], str[i+1:]
		}
	}

	return str, ""
}
