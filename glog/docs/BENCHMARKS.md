# Performance Benchmarks

## Running Benchmarks

```bash
# Run all benchmarks with memory stats
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkLogger_Info -benchmem ./glog/

# Run with custom duration
go test -bench=. -benchtime=10s -benchmem ./...

# Save results for comparison
go test -bench=. -benchmem -count=5 ./... > bench_old.txt
# Make changes
go test -bench=. -benchmem -count=5 ./... > bench_new.txt
benchstat bench_old.txt bench_new.txt
```

## Latest Results

**Environment**: Windows 11, 12th Gen Intel Core i7-12700H (20 threads), Go 1.21

### Logger (end-to-end via mock publisher)

```
BenchmarkLogger_Info-20                    10019863     100.8 ns/op     183 B/op    2 allocs/op
BenchmarkLogger_ErrorWithStackTrace-20       365102    3209   ns/op    2332 B/op   32 allocs/op
BenchmarkLogger_WithMultipleFields-20       3033979     499.5 ns/op     833 B/op   13 allocs/op
```

### Zap Publisher (direct SendMsg, io.Discard)

```
BenchmarkZapLogger_SendMsg-20               2519918     489.5 ns/op     449 B/op    3 allocs/op
BenchmarkZapLogger_SendMsg_WithFields-20    1456318     856.9 ns/op     962 B/op    5 allocs/op
```

### Interpretation

| Operation | ns/op | Throughput | Notes |
|-----------|-------|-----------|-------|
| Info (mock) | ~101 | ~9.9M ops/sec | Non-blocking select-send to channel |
| Error + stack trace | ~3209 | ~312K ops/sec | `runtime.Callers` overhead |
| Info + 4 fields | ~500 | ~2.0M ops/sec | Component + int + string + float |
| Zap SendMsg | ~490 | ~2.0M ops/sec | JSON encoding to io.Discard |
| Zap + 3 fields | ~857 | ~1.2M ops/sec | Field processing overhead |

## Performance Characteristics

### Throughput

| Scenario | Messages/Second | Notes |
|----------|----------------|-------|
| Logger.Info (mock publisher) | ~9,900,000 | Channel send only |
| Single Zap publisher | ~2,000,000 | End-to-end with JSON encoding |
| With stack traces | ~312,000 | runtime.Callers overhead |
| Concurrent (10 goroutines) | scales linearly | Channel provides synchronization |

### Memory

| Component | Memory | Notes |
|-----------|--------|-------|
| Logger instance | ~24B | Channel pointer + atomic pointer |
| LogData (Info, no fields) | ~183B | 2 allocs |
| LogData (4 fields) | ~833B | 13 allocs |
| Stack trace capture | ~2.3KB | 32 allocs |
| Zap SendMsg (no fields) | ~449B | 3 allocs |

## Optimization Tips

### 1. Avoid Stack Traces in Hot Paths

Stack traces add ~3100ns per call. Use only for unexpected errors:

```go
// Only for unexpected errors
if isUnexpected(err) {
    log.Error(ctx, err, models.WithStackTrace())
} else {
    log.Error(ctx, err)
}
```

### 2. Group Related Fields

```go
// 5 individual fields = 5 allocations
log.Info(ctx, "request",
    models.WithStringField("method", method),
    models.WithStringField("path", path),
    models.WithIntField("status", status),
    models.WithStringField("ip", ip),
    models.WithFloatField("duration", dur))

// 1 object field = 1 allocation
log.Info(ctx, "request",
    models.WithObjectField("details", map[string]any{
        "method": method, "path": path, "status": status,
    }))
```

### 3. Configure Buffers for Your Load

```go
// High-throughput service
service := glog.NewLoggerService(
    glog.WithInputBufferSize(1000),
    glog.WithJobBufferSize(5000),
    glog.WithNumWorkers(8),
)

// Low-latency service
service := glog.NewLoggerService(
    glog.WithInputBufferSize(50),
    glog.WithNumWorkers(2),
    glog.WithSendTimeout(50 * time.Millisecond),
)
```

### 4. Monitor Dropped Messages

```go
var dropped atomic.Int64

service := glog.NewLoggerService(
    glog.WithErrorHandler(func(err error) {
        dropped.Add(1)
        slog.Warn("glogger error", "err", err)
    }),
)
```

## Profiling

```bash
# CPU profile
go test -bench=BenchmarkLogger_Info -cpuprofile=cpu.prof -benchmem ./glog/
go tool pprof cpu.prof

# Memory profile
go test -bench=BenchmarkLogger_Info -memprofile=mem.prof -benchmem ./glog/
go tool pprof mem.prof

# Trace
go test -bench=BenchmarkLogger_Info -trace=trace.out ./glog/
go tool trace trace.out
```

## Load Test Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/alexnobleburn/glogger/glog"
    "github.com/alexnobleburn/glogger/glog/models"
    "github.com/alexnobleburn/glogger/glog/zap"
    "sync"
    "time"
)

func main() {
    service := glog.NewLoggerService(
        glog.WithInputBufferSize(1000),
        glog.WithNumWorkers(8),
    )
    zapLogger := zap.NewZapLogger("load-test", "production")
    service.AddLogger("zap", zapLogger)
    service.Start()
    defer service.Stop()

    log := service.NewLogger()

    numGoroutines := 100
    logsPerGoroutine := 10000

    start := time.Now()
    var wg sync.WaitGroup
    wg.Add(numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer wg.Done()
            ctx := context.Background()
            for j := 0; j < logsPerGoroutine; j++ {
                log.Info(ctx, "load test message",
                    models.WithIntField("goroutine_id", id),
                    models.WithIntField("iteration", j))
            }
        }(i)
    }

    wg.Wait()
    elapsed := time.Since(start)

    totalLogs := numGoroutines * logsPerGoroutine
    throughput := float64(totalLogs) / elapsed.Seconds()

    fmt.Printf("Total logs: %d\n", totalLogs)
    fmt.Printf("Duration: %v\n", elapsed)
    fmt.Printf("Throughput: %.0f logs/sec\n", throughput)
}
```
