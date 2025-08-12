package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-service/internal/config"
	"go-service/internal/dtos"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 2 * time.Second, // Reduced from 3s for faster operations
	Transport: &http.Transport{
		MaxIdleConns:        100, // Reduced from 200 to reduce contention
		MaxIdleConnsPerHost: 25,  // Reduced from 50 to reduce contention
		IdleConnTimeout:     10 * time.Second,
		DisableCompression:  true,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
	},
}

type Processor struct {
	url string
}

func NewPaymentProcessor(url string) *Processor {
	return &Processor{
		url: url,
	}
}

func (pp *Processor) Healthcheck() (failing bool, minResponseTime int, err error) {
	url := fmt.Sprintf("%s/payments/service-health", pp.url)

	resp, err := httpClient.Get(url)
	if err != nil {
		return true, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(resp.Body)
		return false, 100, fmt.Errorf("too many requests to payment processor: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return true, 0, fmt.Errorf("payment processor returned status %d: %s", resp.StatusCode, string(body))
	}

	var health dtos.HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return true, 0, errors.New("failed to decode body")
	}

	return health.Failing, health.MinResponseTime, nil
}

type PaymentProcessorRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"`
}

type PaymentProcessorResponse struct {
	Message string `json:"message"`
}

func (pp *Processor) Process(payment dtos.Payment) error {

	url := fmt.Sprintf("%s/payments", pp.url)

	request := PaymentProcessorRequest{
		CorrelationId: payment.CorrelationId,
		Amount:        payment.Amount,
		RequestedAt:   payment.RequestedAt.UTC().Format(config.DateTimeFormat),
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal payment request: %w", err)
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send payment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("payment processor returned status %d: %s", resp.StatusCode, string(body))
	}

	var response PaymentProcessorResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("Warning: failed to parse response: %v\n", err)
	}

	return nil
}

func (pp *Processor) Clear() error {
	url := fmt.Sprintf("%s/admin/purge-payments", pp.url)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(nil))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rinha-Token", "123")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending purge request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed payments purge")
	}

	return nil
}
