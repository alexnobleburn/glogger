package main

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/alexnobleburn/glogger/glog/zap"
)

func main() {
	// Initialize and configure the service
	service := glog.NewLoggerService(
		glog.WithNumWorkers(4),
		glog.WithSendTimeout(200),
	)
	defer service.Stop()

	// Add Zap publisher
	service.AddLogger("zap", zap.NewZapLogger("example-app", "development"))
	service.Start()

	// Create logger
	log := service.NewLogger()
	ctx := context.Background()

	// Info
	log.Info(ctx, "Application started")

	// Structured fields
	log.Info(ctx, "Database connected",
		models.WithComponent("database"),
		models.WithStringField("host", "localhost"),
		models.WithIntField("port", 5432),
		models.WithBoolField("ssl", true))

	// Warning
	log.Warning(ctx, "High memory usage",
		models.WithComponent("monitor"),
		models.WithFloatField("memory_gb", 7.8))

	// Error with stack trace
	err := fmt.Errorf("connection timeout")
	log.Error(ctx, err,
		models.WithComponent("worker"),
		models.WithStackTrace())

	// Context-aware logging
	ctx = context.WithValue(ctx, models.AppID, "custom-service")
	ctx = context.WithValue(ctx, models.EnvName, "staging")
	log.Info(ctx, "Request processed with custom metadata")

	// Multiple errors
	log.Errors(ctx, []error{
		fmt.Errorf("invalid email"),
		fmt.Errorf("missing required field"),
	}, models.WithComponent("validation"))

	// Object field
	log.Info(ctx, "HTTP request completed",
		models.WithComponent("http"),
		models.WithObjectField("request", map[string]any{
			"method": "POST", "path": "/api/users", "status": 201,
		}),
		models.WithIntField("duration_ms", 45))
}
