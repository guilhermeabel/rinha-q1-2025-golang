package dtos

import "time"

type CreatePaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type GetPaymentsSummaryFilters struct {
	From time.Time
	To   time.Time
}

type GetPaymentsSummaryFiltersJson struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type GetPaymentSummaryResponse struct {
	Default  PaymentSummary `json:"default"`
	Fallback PaymentSummary `json:"fallback"`
}

type PaymentSummary struct {
	TotalRequests int64   `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}
