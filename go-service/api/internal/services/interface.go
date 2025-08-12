package services

import (
	"go-service/internal/dtos"
	"go-service/internal/entities"
)

type PaymentsInterface interface {
	RequestProcessing(correlationId string, amount float64) error
	GetSummary(filters dtos.GetPaymentsSummaryFilters) (*entities.PaymentsSummary, error)
	Process() error
	Clear() error
}

type ProcessorManagerInterface interface {
	Dispatch(payment *dtos.Payment) (string, error)
	Clear() error
}
