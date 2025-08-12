package server

import (
	"context"
	"fmt"
	"go-service/internal/services"
	"net/http"
	"strconv"
	"time"
)

type HttpServer struct {
	ps     services.PaymentsInterface
	port   string
	server *http.Server
}

func NewServer(port string, ps services.PaymentsInterface) *HttpServer {
	return &HttpServer{
		ps:   ps,
		port: port,
	}
}

func (s *HttpServer) ListenAndServe() error {
	portNum, err := strconv.Atoi(s.port)
	if err != nil {
		portNum = 8080
	}

	s.server = s.createHTTPServer(portNum)
	return s.server.ListenAndServe()
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *HttpServer) createHTTPServer(port int) *http.Server {
	router := s.loadRoutes(http.NewServeMux())
	middlewareChain := NewChain(
		s.recoverPanic,
		s.noCache,
		s.enableCors,
	)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      middlewareChain(router),
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}
