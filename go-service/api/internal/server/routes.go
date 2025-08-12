package server

import (
	"net/http"
)

func (s *HttpServer) loadRoutes(mux *http.ServeMux) http.HandlerFunc {
	mux.HandleFunc("GET /payments-summary", s.paymentsSummary)
	mux.HandleFunc("POST /purge-payments", s.purgePayments)
	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /healthcheck", s.healthCheck)

	return mux.ServeHTTP
}
