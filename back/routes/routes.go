package routes

import (
    "back/controllers"

    "github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // チャットメッセージ送信
    r.POST("/chat", controllers.HandleChat)

    // メッセージのフラグ更新
    r.POST("/chat/update-flag", controllers.UpdateMessageFlag)

    // 過去の会話を取得
    r.GET("/chat/conversations", controllers.GetConversations)

    r.GET("/chat/research-ai", controllers.HandleResearchAI)

    return r
}
