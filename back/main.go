package main

import (
	"back/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// デバッグモードを有効化
	gin.SetMode(gin.DebugMode)

	// CORSの設定
	router := routes.SetupRouter()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// エラーハンドリングを追加
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithWriter(os.Stdout))

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
