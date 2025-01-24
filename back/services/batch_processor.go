package services

import (
	"back/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
)

type BatchProcessor struct {
	postgresDB *sql.DB
	dynamoDB   *dynamodb.Client
}

func NewBatchProcessor(postgresURI string, dynamoClient *dynamodb.Client) (*BatchProcessor, error) {
	connStr := postgresURI
	if !strings.Contains(postgresURI, "sslmode=") {
		if strings.Contains(postgresURI, "?") {
			connStr += "&sslmode=disable"
		} else {
			connStr += "?sslmode=disable"
		}
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %v", err)
	}

	// 接続テスト
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %v", err)
	}

	return &BatchProcessor{
		postgresDB: db,
		dynamoDB:   dynamoClient,
	}, nil
}

// ProcessConversations は会話データの処理メインロジック
func (bp *BatchProcessor) ProcessConversations() error {
	now := time.Now()
	threeHoursAgo := now.Add(-3 * time.Hour)

	// アクティブユーザーを取得
	users, err := bp.getActiveUsers(threeHoursAgo)
	if err != nil {
		return fmt.Errorf("failed to get active users: %v", err)
	}

	for _, userID := range users {
		// 各ユーザーの会話を期間で取得
		conversations, err := bp.getConversationsInPeriod(userID, threeHoursAgo, now)
		if err != nil {
			log.Printf("Error getting conversations for user %s: %v", userID, err)
			continue
		}

		if len(conversations) == 0 {
			log.Printf("No conversations found for user %s", userID)
			continue
		}

		// 要約処理以下は変更なし
		summary, err := bp.summarizeConversations(conversations)
		if err != nil {
			log.Printf("Error summarizing conversations for user %s: %v", userID, err)
			continue
		}

		vector, err := bp.vectorizeText(summary)
		if err != nil {
			log.Printf("Error vectorizing text for user %s: %v", userID, err)
			continue
		}

		err = bp.saveToPostgres(userID, summary, vector, threeHoursAgo, now)
		if err != nil {
			log.Printf("Error saving to postgres for user %s: %v", userID, err)
			continue
		}

		log.Printf("Successfully processed conversations for user %s", userID)
	}

	return nil
}

func (bp *BatchProcessor) getActiveUsers(since time.Time) ([]string, error) {
	sinceStr := since.Format(time.RFC3339)

	// Scanを使用してアクティブユーザーを取得
	result, err := bp.dynamoDB.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName:        aws.String("Conversations"),
		FilterExpression: aws.String("#ts >= :ts"),
		ExpressionAttributeNames: map[string]string{
			"#ts": "Timestamp",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ts": &types.AttributeValueMemberS{Value: sinceStr},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan DynamoDB: %v", err)
	}

	// ユニークなユーザーIDを収集
	userMap := make(map[string]bool)
	for _, item := range result.Items {
		if userID, ok := item["UserID"].(*types.AttributeValueMemberS); ok {
			userMap[userID.Value] = true
		}
	}

	var users []string
	for userID := range userMap {
		users = append(users, userID)
	}

	return users, nil
}

// PostgreSQLから最後の要約時刻を取得
// func (bp *BatchProcessor) getLastSummaryTime() (time.Time, error) {
// 	var lastTime time.Time
// 	err := bp.postgresDB.QueryRow(`
//         SELECT COALESCE(MAX(end_time), NOW() - INTERVAL '3 hours')
//         FROM conversation_summaries
//     `).Scan(&lastTime)
// 	if err != nil {
// 		return time.Now().Add(-3 * time.Hour), err
// 	}
// 	return lastTime, nil
// }

// // 要約が既に存在するかチェック
// func (bp *BatchProcessor) summaryExists(userID string, start, end time.Time) (bool, error) {
// 	var exists bool
// 	err := bp.postgresDB.QueryRow(`
//         SELECT EXISTS(
//             SELECT 1 FROM conversation_summaries
//             WHERE user_id = $1
//             AND start_time = $2
//             AND end_time = $3
//         )
//     `, userID, start, end).Scan(&exists)
// 	return exists, err
// }

// 会話を要約
func (bp *BatchProcessor) summarizeConversations(conversations []models.Conversation) (string, error) {
	messages := []map[string]string{
		{
			"role":    "system",
			"content": "以下の会話を具体的な内容がわかるように要約してください。",
		},
	}

	for _, conv := range conversations {
		messages = append(messages, map[string]string{
			"role":    conv.Role,
			"content": conv.Content,
		})
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	var openAIMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
			Role:    msg["role"],
			Content: msg["content"],
		})
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4TurboPreview,
			Messages: openAIMessages,
		},
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (bp *BatchProcessor) saveToPostgres(userID string, summary string, vector []float64, startTime time.Time, endTime time.Time) error {
	query := `
        INSERT INTO conversation_summaries 
        (user_id, summary, vector, start_time, end_time)
        VALUES ($1, $2, $3::float8[], $4, $5)
        ON CONFLICT (user_id, start_time, end_time)
        DO UPDATE SET
            summary = EXCLUDED.summary,
            vector = EXCLUDED.vector
    `

	// float64スライスをpq.Float64Arrayに変換
	vectorArray := pq.Float64Array(vector)

	_, err := bp.postgresDB.Exec(query, userID, summary, vectorArray, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to save to postgres: %v", err)
	}

	log.Printf("Successfully saved summary for user %s with vector length %d", userID, len(vector))
	return nil
}

func (bp *BatchProcessor) vectorizeText(text string) ([]float64, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	resp, err := client.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.AdaEmbeddingV2,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("embedding creation failed: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings received")
	}

	embeddings := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embeddings[i] = float64(v)
	}
	log.Printf("Created embedding vector with length %d", len(embeddings))
	return embeddings, nil
}

func NewBatchProcessorWithDynamo(postgresURI string) (*BatchProcessor, error) {
	db := GetDynamoDBClient()
	return NewBatchProcessor(postgresURI, db)
}

func (bp *BatchProcessor) getConversationsInPeriod(userID string, start, end time.Time) ([]models.Conversation, error) {
	startStr := start.Format(time.RFC3339)
	endStr := end.Format(time.RFC3339)

	result, err := bp.dynamoDB.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String("Conversations"),
		KeyConditionExpression: aws.String("UserID = :uid AND #ts BETWEEN :start AND :end"),
		ExpressionAttributeNames: map[string]string{
			"#ts": "Timestamp",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid":   &types.AttributeValueMemberS{Value: userID},
			":start": &types.AttributeValueMemberS{Value: startStr},
			":end":   &types.AttributeValueMemberS{Value: endStr},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %v", err)
	}

	conversations := make([]models.Conversation, 0)
	for _, item := range result.Items {
		timestamp, _ := time.Parse(time.RFC3339, item["Timestamp"].(*types.AttributeValueMemberS).Value)
		conv := models.Conversation{
			ID:        item["ID"].(*types.AttributeValueMemberS).Value,
			UserID:    item["UserID"].(*types.AttributeValueMemberS).Value,
			Role:      item["Role"].(*types.AttributeValueMemberS).Value,
			Content:   item["Content"].(*types.AttributeValueMemberS).Value,
			Timestamp: timestamp,
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}
