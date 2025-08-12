package server

import (
	"encoding/json"
	"go-service/internal/config"
	"go-service/internal/dtos"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type GetPaymentsSummaryFilters struct {
	From string
	To   string
}

func (s *HttpServer) paymentsSummary(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	filters := dtos.GetPaymentsSummaryFilters{}

	if from != "" {
		fromDate, err := time.Parse(config.DateTimeFormat, from)
		if err == nil {
			filters.From = fromDate
		} else {
			slog.Error("failed to parse from date", "date", from, "error", err)
		}
	}

	if to != "" {
		toDate, err := time.Parse(config.DateTimeFormat, to)
		if err == nil {
			filters.To = toDate
		} else {
			slog.Error("failed to parse to date", "date", to, "error", err)
		}
	}

	summary, err := s.ps.GetSummary(filters)
	if err != nil {
		slog.Error("error while fetching payments summary", "error", err)
		http.Error(w, "Error while fetching payments summary", http.StatusInternalServerError)
		return
	}

	summaryResponse := dtos.GetPaymentSummaryResponse{
		Default:  dtos.PaymentSummary(summary.Default),
		Fallback: dtos.PaymentSummary(summary.Fallback),
	}

	writeJSON(w, http.StatusOK, summaryResponse)
}

func (s *HttpServer) createPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("cannot read request body", "error", err)
		http.Error(w, "Cannot read request body", http.StatusUnprocessableEntity)
		return
	}

	defer r.Body.Close()

	var payment dtos.CreatePaymentRequest
	err = json.Unmarshal(body, &payment)
	if err != nil {
		slog.Error("cannot unmarshal request body", "error", err)
		http.Error(w, "Cannot unmarshal request body", http.StatusUnprocessableEntity)
		return
	}

	go func() {
		err = s.ps.RequestProcessing(payment.CorrelationId, payment.Amount)
		if err != nil {
			slog.Error("error requesting payment processing", "error", err)
		}
	}()

	writeJSON(w, http.StatusOK, struct{}{})
}

func (s *HttpServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	err := writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{
		Status: "all good",
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *HttpServer) purgePayments(w http.ResponseWriter, r *http.Request) {
	err := s.ps.Clear()
	if err != nil {
		http.Error(w, "Error when trying to purge payments: "+err.Error(), http.StatusInternalServerError)
	}
}
