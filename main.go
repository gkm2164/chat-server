package main

import (
	"chat-server/ws"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	server := gin.New()

	server.GET("/ws", func(c *gin.Context) {
		ws.Chat(c.Writer, c.Request)
	})

	server.NoRoute(static.Serve("/",
		static.LocalFile("./assets", false)))

	if err := server.Run(":8081"); err != nil {
		log.Fatalf("error while starting server: %+v", err)
	}
}
