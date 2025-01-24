package models

import (
	"time"

	"github.com/lib/pq"
)

type ConversationSummary struct {
    ID        string         `json:"id"`
    UserID    string         `json:"user_id"`
    Summary   string         `json:"summary"`
    Vector    pq.Float64Array `json:"vector"`
    StartTime time.Time      `json:"start_time"`
    EndTime   time.Time      `json:"end_time"`
    CreatedAt time.Time      `json:"created_at"`
}