package gateway

import (
	"go-service/internal/dtos"
)

type PaymentProcessorInterface interface {
	Healthcheck() (failing bool, minResponseTime int, err error)
	Process(payment dtos.Payment) error
	Clear() error
}
