package services

import "time"

// GetCurrentTimestamp は現在のタイムスタンプをISO8601形式で返します
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
