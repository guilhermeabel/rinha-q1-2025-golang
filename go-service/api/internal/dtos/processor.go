package dtos

import "time"

type HealthCheckResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type Payment struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
	Processor     string    `json:"processor,omitempty"`
	RetryCount    int       `json:"retryCount,omitempty"`
}
