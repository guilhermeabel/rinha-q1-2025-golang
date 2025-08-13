package services

import (
	"errors"
	"go-service/internal/dtos"
	internalErrors "go-service/internal/errors"
	"go-service/internal/gateway"
	"time"

	"github.com/go-redis/redis/v8"
)

type ProcessorStatus struct {
	Failing         bool      `json:"failing"`
	MinResponseTime int       `json:"minResponseTime"`
	LastChecked     time.Time `json:"lastChecked"`
}

type ProcessorManager struct {
	rc  *redis.Client
	ppp gateway.PaymentProcessorInterface
	spp gateway.PaymentProcessorInterface
}

func NewProcessorManager(
	redisClient *redis.Client,
	ppp gateway.PaymentProcessorInterface,
	spp gateway.PaymentProcessorInterface,
) *ProcessorManager {
	return &ProcessorManager{
		rc:  redisClient,
		ppp: ppp,
		spp: spp,
	}
}

type ProcessorChoice struct {
	Name                 string
	ExpectedResponseTime int
}

func (pm *ProcessorManager) Dispatch(payment *dtos.Payment) (string, error) {
	err := pm.ppp.Process(*payment)
	if err == nil {
		return "default", nil
	}

	err = pm.spp.Process(*payment)
	if err == nil {
		return "fallback", nil
	}

	return "", internalErrors.ErrNoPaymentProcessorAvailable

}

func (pm *ProcessorManager) Clear() error {
	err := pm.ppp.Clear()
	if err != nil {
		return errors.New("cannot purge primary processor payments: " + err.Error())
	}

	err = pm.spp.Clear()
	if err != nil {
		return errors.New("cannot purge secondary processor payments: " + err.Error())
	}

	return nil
}
