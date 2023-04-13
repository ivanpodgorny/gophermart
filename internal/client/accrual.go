package client

import (
	"context"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"net/http"
	"time"
)

type Accrual struct {
	req *req.Client
}

var statusMap = map[string]entity.OrderStatus{
	"REGISTERED": entity.OrderStatusNew,
	"INVALID":    entity.OrderStatusInvalid,
	"PROCESSING": entity.OrderStatusProcessing,
	"PROCESSED":  entity.OrderStatusProcessed,
}

func NewAccrual(addr string) *Accrual {
	return &Accrual{
		req: req.C().
			SetBaseURL(addr).
			SetTimeout(5 * time.Second),
	}
}

// GetAccrual отправляет запрос к сервису расчёта начислений баллов лояльности для получения
// информации о статусе расчёта начисления по заказу. При ответе сервиса с кодом 429 пытается
// выполнить повторный запрос через минуту.
func (c *Accrual) GetAccrual(ctx context.Context, order string) (entity.OrderStatus, float64, error) {
	respBody := struct {
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual"`
	}{}
	resp, err := c.req.R().
		SetContext(ctx).
		SetRetryCount(2).
		SetRetryFixedInterval(60*time.Second).
		SetRetryCondition(func(resp *req.Response, err error) bool {
			return err == nil && resp.StatusCode == http.StatusTooManyRequests
		}).
		SetSuccessResult(&respBody).
		SetPathParam("number", order).
		Get("/api/orders/{number}")
	if err != nil {
		return "", 0, err
	}

	if resp.IsErrorState() {
		return "", 0, fmt.Errorf("server responded with status code %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusNoContent {
		return entity.OrderStatusInvalid, 0, nil
	}

	return statusMap[respBody.Status], respBody.Accrual, err
}
