# Architecture Overview

## System Design

The Go Logger library is built on a high-performance, asynchronous architecture that decouples log generation from log publishing. This design ensures that logging never blocks the main application flow while providing flexibility to support multiple logging backends simultaneously.

## Core Components

### 1. Logger

**Location**: `glog/logger.go`

The Logger is the primary interface applications interact with. It provides methods for different log levels (Info, Warning, Error, Debug) and handles the initial log message preparation.

**Key Features**:
- Non-blocking: sends to a buffered channel with `select`/`default` — if the buffer is full, the message is dropped silently
- Safe after shutdown: checks `atomic.Bool stopped` before sending, never panics
- Context-aware: extracts metadata from context automatically
- Flexible options: supports various options through functional patterns
- Stack trace support: optional stack trace capture for errors (only when requested via `WithStackTrace()`)

**Flow**:
```
Application -> Logger.Info/Error/etc -> Prepare LogData -> select-send to inputCh
```

### 2. LoggerService

**Location**: `glog/service.go`

The LoggerService is the heart of the logging system. It manages the worker pool, coordinates log distribution to multiple publishers, and handles graceful shutdown.

**Architecture**:
```
Input Channel -> Main Worker -> Job Channel -> Worker Pool -> Publishers
```

**Key Features**:
- **Buffered Channels**: prevents blocking on high-volume scenarios
- **Worker Pool**: parallel processing with configurable worker count
- **Multi-Publisher**: routes each log to all registered publishers
- **Timeout Protection**: prevents slow publishers from blocking the system (`time.NewTimer` + `Stop()`)
- **Panic Recovery**: `recover()` inside publisher goroutine — a panicking publisher never crashes a worker
- **Graceful Shutdown**: `sync.Once` for safe double-Stop, `atomic.Bool` for write-after-Stop protection
- **Configurable ErrorHandler**: all internal errors go through `func(error)` — no `fmt.Println`

**Default Configuration**:

| Parameter | Default | Option |
|-----------|---------|--------|
| Input Buffer | 100 messages | `WithInputBufferSize(n)` |
| Job Buffer | 1000 jobs | `WithJobBufferSize(n)` |
| Workers | 4 | `WithNumWorkers(n)` |
| Send Timeout | 100ms | `WithSendTimeout(d)` |
| Error Handler | `fmt.Println` | `WithErrorHandler(fn)` |

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
- Context: request/operation context
- Message: log message
- Fields: structured fields (int, float, string, bool, object)
- Level: log level (Debug, Info, Warn, Error, DPanic, Panic, Fatal)

#### LogField
Typed field for structured logging:
- Key: field name
- Type: field type enum (`FieldTypeString`, `FieldTypeInt`, `FieldTypeFloat`, `FieldTypeBool`, `FieldTypeObject`)
- Integer, Float, String, Bool, Object: type-specific value storage

#### Options
Functional options pattern for flexible configuration:
- `WithComponent`: set component name
- `WithStackTrace`: enable stack trace
- `WithIntField`/`WithFloatField`/`WithStringField`/`WithBoolField`/`WithObjectField`: add typed fields

## Data Flow

### Complete Flow Diagram

```
+-------------------------------------------------------------------+
|                        Application                                 |
+----------+--------------------------------------------------------+
           |
           |  log.Info(ctx, "message", options...)
           v
+-------------------------------------------------------------------+
|                          Logger                                    |
|  * Applies options (component, fields, stack trace)                |
|  * Creates LogData struct                                          |
|  * select-send to inputCh (non-blocking, drop if full)             |
+----------+--------------------------------------------------------+
           |
           |  logData -> inputCh (buffered, default 100)
           v
+-------------------------------------------------------------------+
|                    LoggerService (Main Worker)                      |
|                                                                    |
|  1. Ranges over inputCh                                            |
|  2. RLock: snapshots registered publishers                         |
|  3. Creates sendJob per publisher, sends to jobCh                  |
+----------+--------------------------------------------------------+
           |
           |  sendJob -> jobCh (buffered, default 1000)
           v
+-------------------------------------------------------------------+
|                     Worker Pool (default 4)                        |
|                                                                    |
|  Each worker:                                                      |
|  1. Ranges over jobCh                                              |
|  2. Spawns goroutine for publisher.SendMsg()                       |
|  3. Waits with time.NewTimer (configurable timeout)                |
|  4. recover() catches publisher panics                             |
+----------+--------------------------------------------------------+
           |
           |  Multiple parallel calls
           |
   +-------+-------+-----------+------------+
   |               |           |            |
   v               v           v            v
+--------+   +---------+  +---------+  +----------+
|  Zap   |   | Logrus  |  | Sentry  |  |  Custom  |
+--------+   +---------+  +---------+  +----------+
```

## Concurrency Model

### Thread Safety

1. **Logger**: thread-safe. Multiple goroutines can call logging methods simultaneously. Non-blocking `select`-send to channel.
2. **LoggerService**: uses `sync.RWMutex` for publisher management. `sync.Once` for Stop(). `atomic.Bool` for stopped flag.
3. **Channels**: Go channels provide built-in synchronization.

### Goroutine Management

```
LoggerService.Start()
  |
  +-> Main Worker Goroutine (1)
  |     Reads inputCh, distributes to jobCh
  |     Closes jobCh when inputCh is drained
  |
  +-> Worker Pool (N goroutines, default 4)
        Each worker reads jobCh
        Spawns short-lived goroutine per publisher call
        recover() inside goroutine catches panics
        time.NewTimer for timeout (properly stopped)
```

### Shutdown Sequence

```
Stop() called
  |
  +-> sync.Once: atomic.Bool stopped = true, close(inputCh)
  |
  +-> mainWg.Wait(): main worker drains inputCh, closes jobCh
  |
  +-> wg.Wait(): workers drain jobCh, all finish
```

## Error Handling

### Timeout Mechanism

Publishers have a configurable timeout (default 100ms):

```go
timer := time.NewTimer(ls.sendTimeout)
defer timer.Stop()

select {
case <-doneCh:
    // Success
case <-timer.C:
    // Timeout - report via ErrorHandler
}
```

Note: `time.NewTimer` + `defer timer.Stop()` is used instead of `time.After` to avoid memory leaks under high load.

### Panic Recovery

Publisher panics are caught inside the goroutine:

```go
go func() {
    defer close(doneCh)
    defer func() {
        if r := recover(); r != nil {
            ls.errorHandler(fmt.Errorf("panic in publisher: %v", r))
        }
    }()
    job.logger.SendMsg(job.logData)
}()
```

This ensures a panicking publisher never crashes a worker or deadlocks Stop().

### Safety Guarantees

- **Double Stop()**: safe, protected by `sync.Once`
- **Write after Stop()**: safe, `atomic.Bool` check + `select`/`default` — no panic
- **Publisher panic**: caught by `recover()`, worker continues processing
- **Channel full**: message silently dropped (non-blocking guarantee)
- **Nil publishers**: skipped with error via ErrorHandler
- **Nil contexts**: default to `context.Background()` in publisher (not mutating LogData)

## Extensibility

### Adding Custom Publishers

Implement the `LogPublisher` interface:

```go
type CustomPublisher struct{}

func (p *CustomPublisher) SendMsg(data *models.LogData) {
    // Your implementation — must return within sendTimeout
}

service.AddLogger("custom", &CustomPublisher{})
```

### Configuring the Service

```go
service := glog.NewLoggerService(
    glog.WithInputBufferSize(500),
    glog.WithJobBufferSize(5000),
    glog.WithNumWorkers(8),
    glog.WithSendTimeout(200 * time.Millisecond),
    glog.WithErrorHandler(func(err error) {
        slog.Error("glogger internal error", "err", err)
    }),
)
```

### Creating a Logger

```go
// Preferred: via service method
logger := service.NewLogger()

// Alternative: via channel (backward compatible)
logger := glog.NewLogger(service.GetInputChan())
```

## Best Practices

### Do's
- Use structured fields instead of string interpolation
- Set meaningful component names for filtering
- Enable stack traces only for errors that need debugging
- Use context to pass request-scoped metadata
- Configure buffer sizes for your expected load
- Always call `service.Stop()` for graceful shutdown
- Set a custom `ErrorHandler` in production to monitor dropped/timed-out messages

### Don'ts
- Don't log sensitive data (passwords, tokens, PII)
- Don't use logging in tight loops without throttling
- Don't create new logger instances per request
- Don't assume all messages are delivered — channel-full drops are possible under extreme load
