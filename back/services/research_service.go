package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type PerplexityResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func ResearchAITopic(c *gin.Context) (string, error) {
	client := resty.New()
	perplexityURL := "https://api.perplexity.ai/chat/completions"

	apiKey := os.Getenv("PERPLEXITY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("API key is not set")
	}

	requestBody := map[string]interface{}{
		"model": "sonar",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "Be precise and concise.",
			},
			{
				"role":    "user",
				"content": "最新のAIに関する話題を5つ以上教えてください。",
			},
		},
		"max_tokens":               8000,  // 必要に応じて調整
		"temperature":              0.2,
		"top_p":                    0.9,
		"search_domain_filter":     []string{"perplexity.ai"},
		"return_images":            false,
		"return_related_questions": false,
		"search_recency_filter":    "month",
		"top_k":                    0,
		"stream":                   false,
		"presence_penalty":         0,
		"frequency_penalty":        1,
		"response_format":          nil,
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		Post(perplexityURL)

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("failed to fetch AI topic, status: %d", resp.StatusCode())
	}

	var result PerplexityResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(result.Choices) > 0 && result.Choices[0].Message.Content != "" {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in response")
}
