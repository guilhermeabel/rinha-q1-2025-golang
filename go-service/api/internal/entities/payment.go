package entities

type Payment struct {
	CorrelationId string
	Amount        float64
	RequestedAt   string
}

type PaymentsSummary struct {
	Default  PaymentStats
	Fallback PaymentStats
}

type PaymentStats struct {
	TotalRequests int64
	TotalAmount   float64
}
