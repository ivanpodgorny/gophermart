package entity

import "time"

type Order struct {
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual"`
	UploadedAt time.Time   `json:"uploaded_at"`
}

type StatusCheckJob struct {
	Num    string
	Status OrderStatus
}

type StatusCheckResult struct {
	Num     string
	Status  OrderStatus
	Accrual float64
}

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

func NewStatusCheckJob(num string) StatusCheckJob {
	return StatusCheckJob{
		Num:    num,
		Status: OrderStatusNew,
	}
}
