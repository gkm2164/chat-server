package main

import (
	"chat-server/ws"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

func main() {
	log := logrus.New()
	switch gin.Mode() {
	case gin.ReleaseMode:
		log.SetFormatter(&logrus.JSONFormatter{})
	default:
	}

	ws.Init(log)

	server := gin.New()
	server.Use(ginlogrus.Logger(log), gin.Recovery())
	server.GET("/ws", func(c *gin.Context) {
		ws.Chat(c.Writer, c.Request)
	})

	server.NoRoute(static.Serve("/",
		static.LocalFile("./assets", false)))

	if err := server.Run(":8081"); err != nil {
		log.Fatalf("error while starting server: %+v", err)
	}
}
