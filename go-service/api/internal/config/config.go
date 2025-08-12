package config

import "time"

const (
	HealthCheckCooldown = 5 * time.Second

	// How often the ProcessorManager checks failing processors
	HealthCheckInterval = 2 * time.Second // Increased from 500ms to reduce overhead

	// Latency threshold in milliseconds - use fallback if default is this much slower
	LatencyThresholdMs = 100

	// Standardized date format for consistency across all components
	DateTimeFormat = "2006-01-02T15:04:05.000Z"
)
