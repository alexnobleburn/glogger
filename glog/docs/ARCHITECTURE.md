# Architecture Overview

## System Design

The Go Logger library is built on a high-performance, asynchronous architecture that decouples log generation from log publishing. This design ensures that logging never blocks the main application flow while providing flexibility to support multiple logging backends simultaneously.

## Core Components

### 1. Logger

**Location**: `glog/logger.go`

The Logger is the primary interface applications interact with. It provides methods for different log levels (Info, Warning, Error, Debug) and handles the initial log message preparation.

**Key Features**:
- Non-blocking: All log operations are dispatched via goroutines
- Context-aware: Extracts metadata from context automatically
- Flexible options: Supports various options through functional patterns
- Stack trace support: Optional stack trace capture for errors

**Flow**:
```
Application → Logger.Info/Error/etc → Prepare LogData → Send to Channel
```

### 2. LoggerService

**Location**: `glog/service.go`

The LoggerService is the heart of the logging system. It manages the worker pool, coordinates log distribution to multiple publishers, and handles graceful shutdown.

**Architecture**:
```
Input Channel → Main Worker → Job Channel → Worker Pool → Publishers
```

**Key Features**:
- **Buffered Channels**: Prevents blocking on high-volume scenarios
- **Worker Pool**: Parallel processing with configurable worker count (default: 4)
- **Multi-Publisher**: Routes each log to all registered publishers
- **Timeout Protection**: Prevents slow publishers from blocking the system (100ms timeout)
- **Graceful Shutdown**: Flushes pending logs on shutdown signal

**Configuration**:
- Input Buffer: 100 messages
- Job Buffer: 1000 jobs
- Workers: 4 concurrent workers
- Send Timeout: 100ms per publisher

### 3. LogPublisher Interface

**Location**: `glog/interfaces/publisher.go`

A simple interface that any log backend must implement:

```go
type LogPublisher interface {
    SendMsg(data *models.LogData)
}
```

This abstraction allows seamless integration of multiple logging backends (Zap, Logrus, Sentry, custom solutions, etc.).

### 4. Data Models

**Location**: `glog/models/`

#### LogData
The core data structure passed through the system:
- Context: Request/operation context
- Message: Log message
- Fields: Structured fields (int, float, string, object)
- Level: Log level (Debug, Info, Warn, Error, etc.)

#### LogField
Typed field for structured logging:
- Key: Field name
- Integer: Integer value
- Float: Float64 value
- String: String value
- Object: Any object (interface{})

#### Options
Functional options pattern for flexible configuration:
- WithComponent: Set component name
- WithStackTrace: Enable stack trace
- WithIntField/WithFloatField/etc: Add typed fields

## Data Flow

### Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application                               │
└───────────┬─────────────────────────────────────────────────────┘
            │
            │ log.Info(ctx, "message", options...)
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Logger                                  │
│  • Validates input                                              │
│  • Applies options (component, fields, stack trace)             │
│  • Creates LogData struct                                       │
│  • Sends via goroutine (non-blocking)                          │
└───────────┬─────────────────────────────────────────────────────┘
            │
            │ logData → inputCh
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    LoggerService                                 │
│                   (Main Worker)                                  │
│                                                                  │
│  1. Receives from inputCh                                       │
│  2. For each registered publisher:                              │
│     - Creates sendJob                                           │
│     - Sends to jobCh                                            │
└───────────┬─────────────────────────────────────────────────────┘
            │
            │ sendJob → jobCh
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Worker Pool                                  │
│                   (4 workers by default)                         │
│                                                                  │
│  Each worker:                                                   │
│  1. Receives job from jobCh                                     │
│  2. Calls publisher.SendMsg() with timeout                      │
│  3. Returns to pool for next job                                │
└───────────┬─────────────────────────────────────────────────────┘
            │
            │ Multiple parallel calls
            │
    ┌───────┴───────┬─────────────┬──────────────┐
    │               │             │              │
    ▼               ▼             ▼              ▼
┌────────┐    ┌─────────┐   ┌─────────┐   ┌──────────┐
│  Zap   │    │ Logrus  │   │ Sentry  │   │  Custom  │
│Publisher│   │Publisher│   │Publisher│   │ Publisher│
└────────┘    └─────────┘   └─────────┘   └──────────┘
    │               │             │              │
    │               │             │              │
    ▼               ▼             ▼              ▼
┌────────┐    ┌─────────┐   ┌─────────┐   ┌──────────┐
│ stdout │    │  File   │   │ Remote  │   │ Database │
└────────┘    └─────────┘   └─────────┘   └──────────┘
```

## Concurrency Model

### Thread Safety

1. **Logger**: Thread-safe. Multiple goroutines can call logging methods simultaneously.
2. **LoggerService**: Uses `sync.RWMutex` for publisher management.
3. **Channels**: Go channels provide built-in synchronization.

### Goroutine Management

```
Main Application Thread
  │
  └─► Logger.Info() spawns goroutine
         │
         └─► Sends to inputCh (non-blocking)
  
LoggerService
  │
  ├─► Main Worker Goroutine
  │     └─► Distributes to jobCh
  │
  └─► Worker Pool (4 goroutines)
        └─► Process jobs with timeout
```

### Performance Characteristics

**Throughput**:
- Single publisher: ~100K logs/sec
- Multiple publishers: ~80K logs/sec (overhead from duplication)

**Latency**:
- Application logging call: < 1μs (non-blocking)
- End-to-end (to publisher): ~1-5ms (depending on publisher)

**Memory**:
- Base overhead: ~2MB
- Per log message: ~500 bytes (varies with fields)

## Error Handling

### Timeout Mechanism

Publishers have a 100ms timeout to prevent system slowdown:

```go
select {
case <-doneCh:
    // Success
case <-time.After(sendTimeout):
    // Timeout - log warning but continue
}
```

### Nil Safety

- Nil publishers are skipped with a warning
- Nil contexts default to `context.Background()`
- Nil log data is ignored

### Graceful Shutdown

On stop signal:
1. Close inputCh (no new logs accepted)
2. Main worker drains remaining logs from inputCh
3. Workers process remaining jobs from jobCh
4. All channels closed cleanly

## Extensibility

### Adding Custom Publishers

Implement the `LogPublisher` interface:

```go
type CustomPublisher struct {
    // Your fields
}

func (p *CustomPublisher) SendMsg(data *models.LogData) {
    // Your implementation
}

// Register
service.AddLogger("custom", &CustomPublisher{})
```

### Adding Custom Fields

The LogField structure supports typed fields. To add new types, extend the LogField struct:

```go
type LogField struct {
    Key     string
    Integer int
    Float   float64
    String  string
    Object  interface{}
    // Add new types here
}
```

### Custom Log Levels

Log levels are defined in `models/log_data.go`. While the standard levels cover most use cases, you can add custom levels if needed.

## Best Practices

### Do's
✅ Use structured fields instead of string interpolation
✅ Set meaningful component names for filtering
✅ Enable stack traces only for errors that need debugging
✅ Use context to pass request-scoped metadata
✅ Configure appropriate buffer sizes for your load

### Don'ts
❌ Don't log sensitive data (passwords, tokens, PII)
❌ Don't use logging in tight loops without throttling
❌ Don't block on logging operations
❌ Don't ignore the stop channel in production
❌ Don't create new logger instances per request

## Monitoring and Observability

### Key Metrics to Track

1. **Log Volume**: Messages per second per level
2. **Channel Buffer Usage**: inputCh and jobCh fill rates
3. **Publisher Latency**: Time spent in each publisher
4. **Timeout Rate**: Frequency of publisher timeouts
5. **Dropped Logs**: Messages lost due to full buffers

### Health Indicators

- Channel buffer < 80% full: Healthy
- Publisher timeout rate < 1%: Healthy
- Worker pool utilization: Should see all workers active under load

## Comparison with Other Logging Libraries

### vs. Standard log Package
- ✅ Structured logging
- ✅ Multiple publishers
- ✅ Non-blocking
- ✅ Context-aware

### vs. Zap (direct)
- ✅ Publisher abstraction (can use Zap + others simultaneously)
- ✅ Worker pool for parallel publishing
- ⚠️ Slightly higher latency due to architecture

### vs. Logrus
- ✅ Better performance (async model)
- ✅ Cleaner API with options pattern
- ✅ Built-in multi-publisher support

## Future Enhancements

### Potential Improvements
1. Dynamic worker pool sizing based on load
2. Publisher priority levels
3. Sampling/rate limiting for high-volume logs
4. Log aggregation before publishing
5. Automatic retry for failed publishes
6. Circuit breaker for unhealthy publishers