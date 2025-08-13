package main

import (
	"context"
	"go-service/internal/gateway"
	"go-service/internal/queue"
	"go-service/internal/server"
	"go-service/internal/services"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	_ "go.uber.org/automaxprocs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		cancel()
	}()

	appPort := "9999"

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	slog.SetDefault(slog.New(logHandler))

	redisClient := redis.NewClient(&redis.Options{
		Addr:         "redis:6379",
		MinIdleConns: 20,
		MaxRetries:   1,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		IdleTimeout:  2 * time.Minute,
	})

	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		slog.Error("failed to connect to redis: %w", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("Failed to close processor manager", "error", err)
		}
	}()

	primaryPaymentProcessor := gateway.NewPaymentProcessor(os.Getenv("PAYMENT_PROCESSOR_URL_DEFAULT"))
	secondaryPaymentProcessor := gateway.NewPaymentProcessor(os.Getenv("PAYMENT_PROCESSOR_URL_FALLBACK"))
	processorManager := services.NewProcessorManager(redisClient, primaryPaymentProcessor, secondaryPaymentProcessor)

	q, err := queue.NewPaymentQueue(redisClient)
	if err != nil {
		slog.Error("Failed to create processor manager", "error", err)
		os.Exit(1)
	}

	paymentsService := services.NewPaymentService(
		processorManager,
		q,
	)

	server := server.NewServer(appPort, paymentsService)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		paymentsService.StartWorker(ctx)
	}()

	slog.Info("READY", "port", appPort)
	<-ctx.Done()
	os.Exit(0)
}
