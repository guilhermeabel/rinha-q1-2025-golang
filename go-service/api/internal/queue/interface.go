package queue

import (
	"go-service/internal/dtos"
)

type PaymentQueueInterface interface {
	Enqueue(p *dtos.Payment) error
	Dequeue() (*dtos.Payment, error)
	RequeueWithBackoff(p *dtos.Payment) error
	Acknowledge(p *dtos.Payment) error
	Summary(f dtos.GetPaymentsSummaryFiltersJson) (*dtos.GetPaymentSummaryResponse, error)
	Clear() error
}
