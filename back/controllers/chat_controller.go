package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"back/services"
)

// TODO: ファイル切り分ける
func getDB() (*sql.DB, error) {
	dsn := os.Getenv("POSTGRES_URI")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func HandleChat(c *gin.Context) {
	var request struct {
		Message string `json:"message" binding:"required"`
		UserID  string `json:"user_id" binding:"required"`
	}

	// JSONバインド
	if err := c.BindJSON(&request); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message and UserID are required"})
		return
	}

	if _, err := services.SaveMessage(request.UserID, "user", request.Message); err != nil {
		log.Printf("Error saving user message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user message"})
		return
	}

	// ---------- RAGサービスによるプロンプト拡張部分 START ----------
	db, err := getDB() 
	if err != nil {
		log.Printf("Error getting DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect DB"})
		return
	}
	defer db.Close()

	ragService := services.NewRAGService(db, os.Getenv("OPENAI_API_KEY"))

	// RAG で拡張プロンプトを作成
	enhancedPrompt, err := ragService.EnhancePrompt(request.UserID, request.Message)
	if err != nil {
		log.Printf("Error enhancing prompt: %v", err)
		// エラー時はとりあえず通常の入力を使用
		enhancedPrompt = request.Message
	}
	// ---------- RAGサービスによるプロンプト拡張部分 END ----------

	replyContent, err := services.CallOpenAI(request.UserID, enhancedPrompt)
	if err != nil {
		log.Printf("Error calling OpenAI: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reply, err := services.SaveMessage(request.UserID, "assistant", replyContent)
	if err != nil {
		log.Printf("Error saving bot reply: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save bot reply"})
		return
	}

	// 必要な情報を含むレスポンスを返す
	c.JSON(http.StatusOK, gin.H{
		"reply":     reply.Content,
		"id":        reply.ID,
		"timestamp": reply.Timestamp.Format(time.RFC3339),
	})
}

func UpdateMessageFlag(c *gin.Context) {
	type RequestBody struct {
		UserID     string `json:"userId" binding:"required"`
		Timestamp  string `json:"timestamp" binding:"required"`
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
	userID := c.Query("userId")
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
	userID := c.Query("userId")
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
		"reply":     reply.Content,
		"id":        reply.ID,
		"timestamp": reply.Timestamp.Format(time.RFC3339),
	})
}
