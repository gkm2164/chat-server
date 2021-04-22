package main

import (
	"chat-server/ws"
	"github.com/gin-gonic/gin"
	"log"
	"net/http/httputil"
	"net/url"
)

func main() {
	server := gin.New()

	server.GET("/ws", func(c *gin.Context) {
		ws.Chat(c.Writer, c.Request)
	})

	server.NoRoute(func(c *gin.Context) {
		u, _ := url.Parse("http://localhost:3000/")

		proxy := httputil.NewSingleHostReverseProxy(u)

		c.Request.URL.Host = u.Host
		c.Request.URL.Scheme = u.Scheme
		c.Request.Header.Set("X-Forwarded-Host", c.Request.Header.Get("Host"))
		c.Request.Host = u.Host

		proxy.ServeHTTP(c.Writer, c.Request)
	})

	if err := server.Run(":8081"); err != nil {
		log.Fatalf("error while starting server: %+v", err)
	}
}
