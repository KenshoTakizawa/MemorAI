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

	// TODO: RAGサービスを使用してプロンプトを強化
    // ragService := services.NewRAGService(os.Getenv("POSTGRES_URI"))
    // enhancedPrompt, err := ragService.EnhancePrompt(request.UserID, request.Message)
    // if err != nil {
    //     log.Printf("Error enhancing prompt: %v", err)
    //     // エラー時は通常のプロンプトを使用
    //     enhancedPrompt = request.Message
    // }

    // 強化されたプロンプトでOpenAIを呼び出し
    // eplyContent, err := services.CallOpenAI(request.UserID, enhancedPrompt)

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

func HandleResearchAI(c *gin.Context) {
	userID := c.Query("userId") // クエリからUserIDを取得
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	// AIの話題をリサーチ
	topic, err := services.ResearchAITopic(c)
	if err != nil {
		log.Printf("Error researching AI topic: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to research AI topic"})
		return
	}

	// リサーチ結果をDynamoDBに保存
	reply, err := services.SaveMessage(userID, "assistant", topic)
	if err != nil {
		log.Printf("Error saving AI research topic: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save AI research topic"})
		return
	}

	// 保存した内容を返す
	c.JSON(http.StatusOK, gin.H{
		"reply":     reply.Content,                     // リサーチ結果の内容
		"id":        reply.ID,                          // メッセージID
		"timestamp": reply.Timestamp.Format(time.RFC3339), // タイムスタンプ
	})
}
