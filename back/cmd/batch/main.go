// cmd/batch/main.go
package main

import (
	"back/services"
	"log"
	"time"
)

func main() {
	// 明示的な接続文字列の指定
	postgresURI := "host=localhost port=5432 user=postgres password=postgres dbname=memorai sslmode=disable"

	// DynamoDBクライアントの取得
	dynamoClient := services.GetDynamoDBClient()

	// 数回リトライを試みる
	var processor *services.BatchProcessor
	var err error

	for i := 0; i < 3; i++ {
		processor, err = services.NewBatchProcessor(postgresURI, dynamoClient)
		if err == nil {
			break
		}
		log.Printf("Attempt %d: Failed to create batch processor: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to create batch processor after retries: %v", err)
	}

	log.Println("Starting batch processing service...")

	// 初回実行
	if err := processor.ProcessConversations(); err != nil {
		log.Printf("Error in initial processing: %v", err)
	}

	// 定期実行の設定
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Starting scheduled batch processing...")
		if err := processor.ProcessConversations(); err != nil {
			log.Printf("Error processing conversations: %v", err)
		}
		log.Println("Batch processing completed")
	}
}
