package queue

import (
	"context"
	"encoding/json"
	"errors"
	"go-service/internal/config"
	"go-service/internal/dtos"
	internalErrors "go-service/internal/errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type PaymentQueue struct {
	rc *redis.Client
}

func NewPaymentQueue(rc *redis.Client) (*PaymentQueue, error) {
	return &PaymentQueue{
		rc: rc,
	}, nil
}

func (pq *PaymentQueue) Close() error {
	return pq.rc.Close()
}

func (pq *PaymentQueue) Enqueue(p *dtos.Payment) error {
	ctx := context.Background()

	key := "payments:queue"
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}

	pipe := pq.rc.Pipeline()
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(p.RequestedAt.UnixMilli()),
		Member: payload,
	})
	_, err = pipe.Exec(ctx)
	return err
}

func (pq *PaymentQueue) Dequeue() (*dtos.Payment, error) {
	ctx := context.Background()
	key := "payments:queue"

	pipe := pq.rc.Pipeline()
	popCmd := pipe.ZPopMin(ctx, key, 1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.Nil {
			return nil, internalErrors.ErrNoPaymentsInQueue
		}
		return nil, err
	}

	result := popCmd.Val()
	if len(result) == 0 {
		return nil, internalErrors.ErrNoPaymentsInQueue
	}

	var payment dtos.Payment
	memberStr, ok := result[0].Member.(string)
	if !ok {
		return nil, errors.New("invalid member type in queue")
	}
	if err := json.Unmarshal([]byte(memberStr), &payment); err != nil {
		return nil, err
	}

	return &payment, nil
}

func (pq *PaymentQueue) getQueueKey() string {
	return "payments:queue"
}

func (pq *PaymentQueue) RequeueWithBackoff(p *dtos.Payment) error {
	ctx := context.Background()
	key := pq.getQueueKey()
	p.RetryCount++

	delayMs := int64(30 * (1 << p.RetryCount))
	if delayMs > 5000 {
		delayMs = 5000
	}

	score := float64(time.Now().UnixMilli() + delayMs)

	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}

	pipe := pq.rc.Pipeline()
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: payload,
	})
	_, err = pipe.Exec(ctx)
	return err
}

func (pq *PaymentQueue) Acknowledge(p *dtos.Payment) error {
	ctx := context.Background()

	queueKey := pq.getQueueKey()
	processedKey := "payments:processed"

	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}

	pipe := pq.rc.TxPipeline()

	pipe.ZRem(ctx, queueKey, payload)

	pipe.ZAdd(ctx, processedKey, &redis.Z{
		Score:  float64(p.RequestedAt.UnixMilli()),
		Member: payload,
	})

	results, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	if len(results) != 2 {
		return errors.New("unexpected number of transaction results")
	}

	return nil
}

func (pq *PaymentQueue) Clear() error {
	ctx := context.TODO()

	return pq.rc.FlushDB(ctx).Err()
}

func (pq *PaymentQueue) Summary(f dtos.GetPaymentsSummaryFiltersJson) (*dtos.GetPaymentSummaryResponse, error) {
	ctx := context.Background()
	key := "payments:processed"

	now := time.Now()
	var minScore, maxScore float64 = float64(0), float64(now.UnixMilli())

	if f.From != "" {
		if fromTime, err := time.Parse(config.DateTimeFormat, f.From); err == nil {
			minScore = float64(fromTime.UnixMilli())
		}
	}

	if f.To != "" {
		if toTime, err := time.Parse(config.DateTimeFormat, f.To); err == nil {
			maxScore = float64(toTime.UnixMilli())
		}
	}

	var defaultSummary, fallbackSummary dtos.PaymentSummary
	const batchSize = 10000
	var offset int64 = 0

	for {
		payments, err := pq.rc.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min:    strconv.FormatFloat(minScore, 'f', 0, 64),
			Max:    strconv.FormatFloat(maxScore, 'f', 0, 64),
			Offset: offset,
			Count:  batchSize,
		}).Result()

		if err != nil {
			return nil, err
		}

		if len(payments) == 0 {
			break
		}

		for _, paymentStr := range payments {
			var payment dtos.Payment
			if err := json.Unmarshal([]byte(paymentStr), &payment); err != nil {
				continue
			}

			processor := payment.Processor
			if processor == "" {
				processor = "default"
			}

			switch processor {
			case "default":
				defaultSummary.TotalRequests++
				defaultSummary.TotalAmount += payment.Amount
			case "fallback":
				fallbackSummary.TotalRequests++
				fallbackSummary.TotalAmount += payment.Amount
			}
		}

		offset += int64(len(payments))

		if len(payments) < batchSize {
			break
		}
	}

	return &dtos.GetPaymentSummaryResponse{
		Default:  defaultSummary,
		Fallback: fallbackSummary,
	}, nil
}
