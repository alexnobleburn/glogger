package examples

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/alexnobleburn/glogger/glog/zap"
)

func main() {
	// Initialize logger service
	loggerService := glog.NewLoggerService()
	defer loggerService.Stop() // Gracefully shutdown and flush all logs

	// Add Zap logger publisher
	zapLogger := zap.NewZapLogger("example-app", "development")
	loggerService.AddLogger("zap", zapLogger)

	// Start the service
	loggerService.Start()

	// Create logger instance
	log := glog.NewLogger(loggerService.GetInputChan())

	// Example 1: Basic info logging
	ctx := context.Background()
	log.Info(ctx, "Application started successfully")

	// Example 2: Logging with component
	log.Info(ctx, "Database connection established",
		models.WithComponent("database"),
		models.WithStringField("host", "localhost"),
		models.WithIntField("port", 5432))

	// Example 3: Warning with metrics
	log.Warning(ctx, "High memory usage detected",
		models.WithComponent("monitor"),
		models.WithFloatField("memory_usage_gb", 7.8),
		models.WithIntField("threshold_gb", 8))

	// Example 4: Error logging with stack trace
	err := simulateError()
	if err != nil {
		log.Error(ctx, err,
			models.WithComponent("worker"),
			models.WithStackTrace(),
			models.WithStringField("operation", "data_processing"))
	}

	// Example 5: Debug logging
	log.Debug(ctx, "Cache lookup",
		models.WithComponent("cache"),
		models.WithStringField("key", "user:12345"),
		models.WithStringField("result", "miss"))

	// Example 6: Context-aware logging
	ctxWithMetadata := context.WithValue(ctx, models.AppID, "custom-service")
	ctxWithMetadata = context.WithValue(ctxWithMetadata, models.EnvName, "staging")
	log.Info(ctxWithMetadata, "Request processed with custom metadata")

	// Example 7: Multiple errors
	errors := []error{
		fmt.Errorf("validation error: invalid email"),
		fmt.Errorf("validation error: missing required field"),
	}
	log.Errors(ctx, errors,
		models.WithComponent("validation"))

	// Example 8: Structured logging with object
	requestData := map[string]interface{}{
		"method": "POST",
		"path":   "/api/users",
		"status": 201,
	}
	log.Info(ctx, "HTTP request completed",
		models.WithComponent("http"),
		models.WithObjectField("request", requestData),
		models.WithIntField("duration_ms", 45))

	fmt.Println("\nAll examples logged successfully!")
}

func simulateError() error {
	return fmt.Errorf("failed to process data: connection timeout")
}
