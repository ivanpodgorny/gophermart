package entity

import "time"

type Transaction struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type TransactionType string

const TransactionTypeOut TransactionType = "OUT"
