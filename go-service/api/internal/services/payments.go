package services

import (
	"context"
	"errors"
	"go-service/internal/config"
	"go-service/internal/dtos"
	"go-service/internal/entities"
	internalErrors "go-service/internal/errors"
	"go-service/internal/queue"
	"log/slog"
	"time"
)

type PaymentService struct {
	pm ProcessorManagerInterface
	q  queue.PaymentQueueInterface
}

func NewPaymentService(
	pm ProcessorManagerInterface,
	q queue.PaymentQueueInterface,
) *PaymentService {
	return &PaymentService{
		pm: pm,
		q:  q,
	}
}

func (ps *PaymentService) StartWorker(ctx context.Context) {
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		go ps.worker(ctx, i)
	}

	<-ctx.Done()
}

func (ps *PaymentService) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := ps.Process(); err != nil {
				if !errors.Is(err, internalErrors.ErrNoPaymentsInQueue) &&
					!errors.Is(err, internalErrors.ErrNoPaymentProcessorAvailable) &&
					!errors.Is(err, context.DeadlineExceeded) {
					slog.Error("Worker error", "workerID", workerID, "err", err)
				}
			}

		}
	}
}

func (ps *PaymentService) Process() error {
	payment, err := ps.q.Dequeue()
	if err != nil {
		return err
	}

	processorName, err := ps.pm.Dispatch(payment)
	if err != nil {
		ps.q.RequeueWithBackoff(payment)
		return err
	}

	payment.Processor = processorName

	return ps.q.Acknowledge(payment)
}

func (ps *PaymentService) Clear() error {
	err := ps.q.Clear()
	if err != nil {
		return errors.New("cannot purge payments queue: " + err.Error())
	}

	return nil
}

func (ps *PaymentService) GetSummary(filters dtos.GetPaymentsSummaryFilters) (*entities.PaymentsSummary, error) {
	var fromStr, toStr string

	if !filters.From.IsZero() {
		fromStr = filters.From.UTC().Format(config.DateTimeFormat)
	}

	if !filters.To.IsZero() {
		toStr = filters.To.UTC().Format(config.DateTimeFormat)
	}

	summary, err := ps.q.Summary(dtos.GetPaymentsSummaryFiltersJson{
		From: fromStr,
		To:   toStr,
	})
	if err != nil {
		return nil, err
	}

	return &entities.PaymentsSummary{
		Default: entities.PaymentStats{
			TotalRequests: summary.Default.TotalRequests,
			TotalAmount:   summary.Default.TotalAmount,
		},
		Fallback: entities.PaymentStats{
			TotalRequests: summary.Fallback.TotalRequests,
			TotalAmount:   summary.Fallback.TotalAmount,
		},
	}, nil
}

func (ps *PaymentService) RequestProcessing(correlationId string, amount float64) error {
	payment := &dtos.Payment{
		CorrelationId: correlationId,
		Amount:        amount,
		RequestedAt:   time.Now().UTC(),
	}

	return ps.q.Enqueue(payment)
}
