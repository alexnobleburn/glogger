package main

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/alexnobleburn/glogger/glog/zap"
	"time"
)

func main() {
	fmt.Println("=== Basic Logger Example ===")

	// Step 1: Create stop channel for graceful shutdown
	stopCh := make(chan struct{})
	defer func() {
		fmt.Println("\nShutting down logger...")
		close(stopCh)
		time.Sleep(200 * time.Millisecond) // Allow logs to flush
		fmt.Println("Logger shutdown complete")
	}()

	// Step 2: Initialize logger service
	loggerService := glog.NewLoggerService(stopCh)

	// Step 3: Add Zap publisher (outputs JSON to stdout)
	zapLogger := zap.NewZapLogger("basic-example", "development")
	loggerService.AddLogger("zap", zapLogger)

	// Step 4: Start the service (starts worker pool)
	loggerService.Start()

	// Step 5: Create logger instance for your application
	log := glog.NewLogger(loggerService.GetInputChan())

	// Step 6: Start logging!
	ctx := context.Background()

	// Basic info log
	fmt.Println("1. Basic Info Log:")
	log.Info(ctx, "Application started successfully")
	time.Sleep(100 * time.Millisecond)

	// Info log with fields
	fmt.Println("\n2. Info Log with Structured Fields:")
	log.Info(ctx, "User logged in",
		models.WithStringField("username", "john_doe"),
		models.WithIntField("user_id", 12345),
		models.WithStringField("ip_address", "192.168.1.100"))
	time.Sleep(100 * time.Millisecond)

	// Warning log
	fmt.Println("\n3. Warning Log:")
	log.Warning(ctx, "API rate limit approaching",
		models.WithIntField("remaining_calls", 10),
		models.WithIntField("limit", 100))
	time.Sleep(100 * time.Millisecond)

	// Error log
	fmt.Println("\n4. Error Log:")
	err := fmt.Errorf("connection timeout")
	log.Error(ctx, err,
		models.WithComponent("database"),
		models.WithStringField("host", "localhost"),
		models.WithIntField("port", 5432))
	time.Sleep(100 * time.Millisecond)

	// Debug log
	fmt.Println("\n5. Debug Log:")
	log.Debug(ctx, "Cache lookup performed",
		models.WithStringField("key", "user:12345"),
		models.WithStringField("result", "hit"),
		models.WithIntField("ttl_seconds", 3600))
	time.Sleep(100 * time.Millisecond)

	// Log with component
	fmt.Println("\n6. Log with Component Tag:")
	log.Info(ctx, "Payment processed successfully",
		models.WithComponent("payment-service"),
		models.WithFloatField("amount", 99.99),
		models.WithStringField("currency", "USD"),
		models.WithStringField("transaction_id", "txn_abc123"))
	time.Sleep(100 * time.Millisecond)

	// Multiple fields of different types
	fmt.Println("\n7. Complex Log with Multiple Field Types:")
	log.Info(ctx, "HTTP request completed",
		models.WithComponent("api"),
		models.WithStringField("method", "POST"),
		models.WithStringField("path", "/api/users"),
		models.WithIntField("status_code", 201),
		models.WithIntField("response_size_bytes", 1024),
		models.WithFloatField("duration_seconds", 0.234))
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n=== Example Complete ===")
}
