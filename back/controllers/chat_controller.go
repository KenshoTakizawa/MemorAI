package controllers

import (
	"back/services"
	"net/http"
	"time"

	"log"

	"github.com/gin-gonic/gin"
)

func HandleChat(c *gin.Context) {
    var request struct {
        Message string `json:"message" binding:"required"`
        UserID  string `json:"user_id" binding:"required"`
    }

    if err := c.BindJSON(&request); err != nil {
        log.Printf("Error binding JSON: %v", err)
        c.JSON(400, gin.H{"error": "Message and UserID are required"})
        return
    }

    // ユーザーからのメッセージを保存
    _, err := services.SaveMessage(request.UserID, "user", request.Message)
    if err != nil {
        log.Printf("Error saving message: %v", err)
        c.JSON(500, gin.H{"error": "Failed to save user message"})
        return
    }

    // OpenAIからの返信を取得
    replyContent, err := services.CallOpenAI(request.UserID, request.Message)
	log.Printf("replyContent: %+v", replyContent)
    if err != nil {
        log.Printf("Error calling OpenAI: %v", err)
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 返信を保存
    reply, err := services.SaveMessage(request.UserID, "assistant", replyContent)
    if err != nil {
        log.Printf("Error saving bot reply: %v", err)
        c.JSON(500, gin.H{"error": "Failed to save bot reply"})
        return
    }

	log.Printf("reply: %+v", reply)

    // 必要な情報を含むレスポンスを返す
    c.JSON(200, gin.H{
        "reply": reply.Content,       // OpenAI の返信内容
        "id": reply.ID,               // 保存されたメッセージの ID
        "timestamp": reply.Timestamp.Format(time.RFC3339), // タイムスタンプを返す
    })
}

func UpdateMessageFlag(c *gin.Context) {
	type RequestBody struct {
		UserID     string `json:"userId" binding:"required"`    // UserIDは必須
		Timestamp  string `json:"timestamp" binding:"required"` // Timestampを追加
		IsLiked    *bool  `json:"isLiked"`
		IsDisliked *bool  `json:"isDisliked"`
	}

	var requestBody RequestBody
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateMessageFlag(requestBody.UserID, requestBody.Timestamp, requestBody.IsLiked, requestBody.IsDisliked)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message flag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message updated successfully"})
}

func GetConversations(c *gin.Context) {
	userID := c.Query("userId") // クエリパラメータからuserIdを取得
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	conversations, err := services.GetAllConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}
