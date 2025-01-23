package models

import (
	"time"
)

type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
} 