package services

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
)

func CallOpenAI(userID string, message string) (string, error) {
	fmt.Printf("CallOpenAI: %+v", message)
	client := resty.New()
	apiKey := os.Getenv("OPENAI_API_KEY")
	openAIURL := "https://api.openai.com/v1/chat/completions"
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	// デバッグログを追加
	// fmt.Printf("Making request with userID: %s, message: %s\n", userID, message)

	recentConversations, err := GetRecentConversations(userID, 5)
	if err != nil {
		// fmt.Printf("Error getting recent conversations: %v\n", err)
		return "", err
	}

	// 会話履歴の初期化
	messages := []map[string]string{
		{
			"role":    "system",
			"content": "過去の会話を参考に、ユーザーの質問に答えてください。",
		},
	}

	// 会話履歴を追加
	for i := len(recentConversations) - 1; i >= 0; i-- {
		messages = append(messages, map[string]string{
			"role":    recentConversations[i].Role,
			"content": recentConversations[i].Content,
		})
	}
	

	// 新しいメッセージを追加
	// メッセージを追加
	// messages = append(messages, map[string]string{
		// "role":    "user",
		// "content": message,
	// })

	// `content`だけを改行で羅列して出力
	fmt.Println("Messages content:")
	for i, msg := range messages {
		fmt.Printf("%d: %s\n", i+1, msg["content"])
	}

	requestBody := map[string]interface{}{
		"model":    "gpt-4o-mini",
		"messages": messages,
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		Post(openAIURL)

	if err != nil {
		return "", err
	}

	// レスポンスをデバッグ出力
	fmt.Println("OpenAI Response:", resp.String())

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if ok && len(choices) > 0 {
		messageContent := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)

		// 応答を保存
		if messageContent != "" {
			assistantMessage, err := SaveMessage(userID, "assistant", messageContent)
			if err != nil {
				fmt.Printf("Failed to save assistant message: %v\n", err)
			} else {
				fmt.Printf("Assistant message saved: %+v\n", assistantMessage)
			}
		}
		

		return messageContent, nil
	}

	return "", nil
}
