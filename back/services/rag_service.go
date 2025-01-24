package services

import (
    "back/models"
    "database/sql"
    "fmt"
    "strings"
)

// RAGService 構造体の定義
type RAGService struct {
    db *sql.DB
    openAIKey string
}

// NewRAGService コンストラクタ
func NewRAGService(db *sql.DB, openAIKey string) *RAGService {
    return &RAGService{
        db: db,
        openAIKey: openAIKey,
    }
}

// 類似度の高い会話を検索する関数
func (rs *RAGService) findSimilarConversations(userID string, queryVector []float64) ([]models.ConversationSummary, error) {
    // PostgreSQLでコサイン類似度を計算して類似の会話を検索
    query := `
        SELECT id, user_id, summary, vector, start_time, end_time, created_at
        FROM conversation_summaries
        WHERE user_id = $1
        ORDER BY vector <=> $2
        LIMIT 3
    `

    rows, err := rs.db.Query(query, userID, queryVector)
    if err != nil {
        return nil, fmt.Errorf("similarity search failed: %v", err)
    }
    defer rows.Close()

    var conversations []models.ConversationSummary
    for rows.Next() {
        var conv models.ConversationSummary
        err := rows.Scan(
            &conv.ID,
            &conv.UserID,
            &conv.Summary,
            &conv.Vector,
            &conv.StartTime,
            &conv.EndTime,
            &conv.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("row scan failed: %v", err)
        }
        conversations = append(conversations, conv)
    }

    return conversations, nil
}

// プロンプトを生成する関数
func (rs *RAGService) buildPromptWithContext(query string, conversations []models.ConversationSummary) string {
    var contextBuilder strings.Builder

    // システムプロンプトの作成
    contextBuilder.WriteString("以下は関連する過去の会話の要約です：\n\n")

    // 過去の会話コンテキストを追加
    for _, conv := range conversations {
        contextBuilder.WriteString(fmt.Sprintf("- %s\n", conv.Summary))
    }

    // 最終的なプロンプトの構築
    contextBuilder.WriteString("\n上記の過去の会話を踏まえて、以下の質問に答えてください：\n")
    contextBuilder.WriteString(query)

    return contextBuilder.String()
}

// EnhancePromptのエラーハンドリングを改善した版
func (rs *RAGService) EnhancePrompt(userID string, query string) (string, error) {
    // クエリをベクトル化
    queryVector, err := rs.vectorizeText(query)
    if err != nil {
        return query, fmt.Errorf("vectorization failed: %v", err) // 元のクエリを返す
    }

    // 類似度の高い過去の会話を検索
    similarConversations, err := rs.findSimilarConversations(userID, queryVector)
    if err != nil {
        return query, fmt.Errorf("similar conversation search failed: %v", err) // 元のクエリを返す
    }

    // 類似の会話が見つからない場合は元のクエリを返す
    if len(similarConversations) == 0 {
        return query, nil
    }

    // プロンプトを生成
    enhancedPrompt := rs.buildPromptWithContext(query, similarConversations)
    
    return enhancedPrompt, nil
}