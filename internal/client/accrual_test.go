package client

import (
	"context"
	"encoding/json"
	"github.com/imroc/req/v3"
	"github.com/ivanpodgorny/gophermart/internal/entity"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestAccrual_GetAccrual(t *testing.T) {
	var (
		ctx        = context.Background()
		order      = "116322550058324"
		errOrder   = "655770442208670"
		wrongOrder = "711388585544181"
		status     = "PROCESSED"
		accrual    = 500.0
		addr       = "https://accrual.loc"
		getURL     = func(n string) string {
			return addr + "/api/orders/" + n
		}
		r = req.C().SetBaseURL(addr)
	)

	httpmock.ActivateNonDefault(r.GetClient())
	defer httpmock.DeactivateAndReset()

	b, _ := json.Marshal(&struct {
		Order   string  `json:"order"`
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual"`
	}{
		Order:   order,
		Status:  status,
		Accrual: accrual,
	})
	httpmock.RegisterResponder(
		"GET",
		getURL(order),
		httpmock.NewBytesResponder(http.StatusOK, b),
	)
	httpmock.RegisterResponder(
		"GET",
		getURL(errOrder),
		httpmock.NewStringResponder(http.StatusInternalServerError, ""),
	)
	httpmock.RegisterResponder(
		"GET",
		getURL(wrongOrder),
		httpmock.NewStringResponder(http.StatusNoContent, ""),
	)
	client := Accrual{
		req: r,
	}

	s, a, err := client.GetAccrual(ctx, order)
	assert.NoError(t, err, "успешное получение данных о начислении")
	assert.Equal(t, statusMap[status], s, "успешное получение данных о начислении")
	assert.Equal(t, accrual, a, "успешное получение данных о начислении")

	_, _, err = client.GetAccrual(ctx, errOrder)
	assert.Error(t, err, "ответ сервиса с ошибкой")

	s, _, err = client.GetAccrual(ctx, wrongOrder)
	assert.NoError(t, err, "незарегистрированный номер заказа")
	assert.Equal(t, entity.OrderStatusInvalid, s, "незарегистрированный номер заказа")
}
