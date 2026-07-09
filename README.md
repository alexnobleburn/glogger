# Go Logger Library

A flexible, high-performance logging library for Go applications with support for multiple log publishers, structured logging, and context-aware logging.

## Features

- **High Performance**: Buffered channels and worker pools for efficient log processing
- **Pluggable Publishers**: Support for multiple log publishers (Zap included, easily extensible)
- **Structured Logging**: Rich field support (int, float, string, bool, object)
- **Context-Aware**: Extract metadata from context automatically
- **Stack Traces**: Optional stack trace capture for errors
- **Non-Blocking**: Asynchronous log processing with configurable timeouts
- **Thread-Safe**: Safe for concurrent use
- **Configurable**: Functional options for service tuning

## Installation

```bash
go get github.com/alexnobleburn/glogger
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/alexnobleburn/glogger/glog/zap"
)

func main() {
	// Initialize logger service
	service := glog.NewLoggerService()
	defer service.Stop() // Graceful shutdown

	// Add Zap publisher
	service.AddLogger("zap", zap.NewZapLogger("my-app", "production"))

	// Start the service
	service.Start()

	// Create logger
	log := service.NewLogger()

	// Use it!
	ctx := context.Background()
	log.Info(ctx, "Application started")
	log.Error(ctx, fmt.Errorf("something went wrong"),
		models.WithComponent("main"),
		models.WithStackTrace())
}
```

## Usage Examples

### Basic Logging

```go
ctx := context.Background()

// Info logging
log.Info(ctx, "User logged in",
    models.WithStringField("user_id", "12345"),
    models.WithComponent("auth"))

// Warning logging
log.Warning(ctx, "API rate limit approaching",
    models.WithIntField("remaining_calls", 10))

// Error logging
err := someOperation()
if err != nil {
    log.Error(ctx, err,
        models.WithComponent("operation"),
        models.WithStackTrace())
}

// Debug logging
log.Debug(ctx, "Cache miss",
    models.WithStringField("key", "user:12345"))
```

### Structured Fields

```go
// Integer field
log.Info(ctx, "Request processed",
    models.WithIntField("status_code", 200),
    models.WithIntField("duration_ms", 45))

// Float field
log.Info(ctx, "Performance metric",
    models.WithFloatField("response_time", 0.234))

// String field
log.Info(ctx, "User action",
    models.WithStringField("action", "login"),
    models.WithStringField("ip", "192.168.1.1"))

// Bool field
log.Info(ctx, "Feature flag",
    models.WithBoolField("enabled", true))

// Object field
log.Info(ctx, "Request details",
    models.WithObjectField("request", req))
```

### Context-Aware Logging

```go
// Add metadata to context
ctx := context.WithValue(context.Background(), models.AppID, "custom-app-id")
ctx = context.WithValue(ctx, models.EnvName, "staging")

// Logger will automatically extract these values
log.Info(ctx, "Processing request")
```

### Multiple Errors

```go
errs := []error{
    fmt.Errorf("database connection failed"),
    fmt.Errorf("cache unavailable"),
}

log.Errors(ctx, errs,
    models.WithComponent("initialization"))
```

### Custom Log Publisher

Implement the `LogPublisher` interface to add your own publisher:

```go
type CustomPublisher struct{}

func (p *CustomPublisher) SendMsg(data *models.LogData) {
    // Your custom logging implementation
    fmt.Printf("[%s] %s\n", data.Level.String(), data.Msg)
}

// Add to service
service.AddLogger("custom", &CustomPublisher{})
```

Publishers can be removed at runtime:

```go
service.RemoveLogger("custom")
```

## Service Configuration

`NewLoggerService` accepts functional options for tuning:

```go
service := glog.NewLoggerService(
    glog.WithInputBufferSize(200),   // Input channel buffer (default: 100)
    glog.WithJobBufferSize(2000),    // Job channel buffer (default: 1000)
    glog.WithNumWorkers(8),          // Worker pool size (default: 4)
    glog.WithSendTimeout(200 * time.Millisecond), // Publisher timeout (default: 100ms)
    glog.WithErrorHandler(func(err error) {       // Custom error handler
        sentry.CaptureException(err)
    }),
)
```

### Defaults

| Parameter | Default |
|-----------|---------|
| Input buffer | 100 messages |
| Job buffer | 1000 jobs |
| Worker count | 4 |
| Send timeout | 100ms |

## Architecture

```
┌─────────────┐
│   Logger    │ ──► Input Channel ──► ┌──────────────────┐
└─────────────┘                        │  LoggerService   │
                                       │  (Main Worker)   │
                                       └────────┬─────────┘
                                                │
                                       ┌────────▼─────────┐
                                       │   Job Channel    │
                                       └────────┬─────────┘
                                                │
                        ┌───────────────────────┼───────────────────────┐
                        │                       │                       │
                   ┌────▼─────┐           ┌────▼─────┐           ┌────▼─────┐
                   │ Worker 1 │           │ Worker 2 │           │ Worker N │
                   └────┬─────┘           └────┬─────┘           └────┬─────┘
                        │                      │                      │
                   ┌────▼─────┐           ┌───▼──────┐          ┌────▼─────┐
                   │  Zap     │           │  Sentry  │          │  Custom  │
                   │ Publisher│           │ Publisher│          │ Publisher│
                   └──────────┘           └──────────┘          └──────────┘
```

## Log Levels

```go
models.DebugLevel   // Verbose debugging information
models.InfoLevel    // General informational messages
models.WarnLevel    // Warning messages
models.ErrorLevel   // Error messages
models.DPanicLevel  // Development panic
models.PanicLevel   // Panic level
models.FatalLevel   // Fatal errors (calls os.Exit)
```

## Performance Considerations

- **Non-blocking**: Log sends drop messages when the channel is full rather than blocking the caller
- **Buffered channels**: Configurable buffer sizes for high-volume logging
- **Worker pool**: Parallel processing of log messages across publishers
- **Timeout protection**: Prevents slow publishers from blocking the worker pool
- **Panic recovery**: Workers recover from publisher panics without crashing

## Best Practices

1. **Always use context**: Pass meaningful context for better observability
2. **Use components**: Tag logs with component names for easier filtering
3. **Structured fields**: Use typed fields instead of formatting strings
4. **Stack traces sparingly**: Only enable for errors that need debugging
5. **Graceful shutdown**: Always call `service.Stop()` (or use `defer`) to flush all pending logs

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details
