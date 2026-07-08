package glog

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog/models"
	"sync"
	"testing"
	"time"
)

// mockPublisher implements interfaces/publisher for testing
type mockPublisher struct {
	mu       sync.Mutex
	logs     []*models.LogData
	sendFunc func(*models.LogData)
}

func (m *mockPublisher) SendMsg(data *models.LogData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendFunc != nil {
		m.sendFunc(data)
	}
	m.logs = append(m.logs, data)
}

func (m *mockPublisher) GetLogs() []*models.LogData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*models.LogData{}, m.logs...)
}

func (m *mockPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = nil
}

func setupTestLogger() (*Logger, *mockPublisher, *LoggerService) {
	loggerService := NewLoggerService()

	mock := &mockPublisher{logs: make([]*models.LogData, 0)}
	loggerService.AddLogger("mock", mock)
	loggerService.Start()

	logger := loggerService.NewLogger()

	return logger, mock, loggerService
}

// waitForLogs polls until the mock has at least n logs or timeout.
func waitForLogs(mock *mockPublisher, n int, timeout time.Duration) []*models.LogData {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		logs := mock.GetLogs()
		if len(logs) >= n {
			return logs
		}
		time.Sleep(5 * time.Millisecond)
	}
	return mock.GetLogs()
}

func TestLogger_Info(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	message := "test info message"

	logger.Info(ctx, message)
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Msg != message {
		t.Errorf("expected message %q, got %q", message, logs[0].Msg)
	}

	if logs[0].Level != models.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", logs[0].Level)
	}
}

func TestLogger_Error(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	err := fmt.Errorf("test error")

	logger.Error(ctx, err)
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Msg != err.Error() {
		t.Errorf("expected message %q, got %q", err.Error(), logs[0].Msg)
	}

	if logs[0].Level != models.ErrorLevel {
		t.Errorf("expected ErrorLevel, got %v", logs[0].Level)
	}
}

func TestLogger_ErrorWithStackTrace(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	err := fmt.Errorf("test error with stack")

	logger.Error(ctx, err, models.WithStackTrace())
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	hasFilename := false
	for _, field := range logs[0].Fields {
		if field.Key == models.FieldFilenameKey {
			hasFilename = true
			if field.String == "" {
				t.Error("expected filename field to have a value")
			}
		}
	}

	if !hasFilename {
		t.Error("expected filename field in log with stack trace")
	}
}

func TestLogger_Warning(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	message := "test warning"

	logger.Warning(ctx, message)
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Level != models.WarnLevel {
		t.Errorf("expected WarnLevel, got %v", logs[0].Level)
	}
}

func TestLogger_Debug(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	message := "test debug"

	logger.Debug(ctx, message)
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Level != models.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", logs[0].Level)
	}
}

func TestLogger_Errors(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	errs := []error{
		fmt.Errorf("error 1"),
		fmt.Errorf("error 2"),
		fmt.Errorf("error 3"),
	}

	logger.Errors(ctx, errs)
	logs := waitForLogs(mock, len(errs), time.Second)

	if len(logs) != len(errs) {
		t.Fatalf("expected %d logs, got %d", len(errs), len(logs))
	}

	expectedMessages := make(map[string]bool)
	for _, err := range errs {
		expectedMessages[err.Error()] = false
	}

	for _, log := range logs {
		if _, exists := expectedMessages[log.Msg]; exists {
			expectedMessages[log.Msg] = true
		} else {
			t.Errorf("unexpected log message: %q", log.Msg)
		}
	}

	for msg, found := range expectedMessages {
		if !found {
			t.Errorf("expected message %q not found in logs", msg)
		}
	}
}

func TestLogger_WithComponent(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	component := "test-component"

	logger.Info(ctx, "test message", models.WithComponent(component))
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	hasComponent := false
	for _, field := range logs[0].Fields {
		if field.Key == models.FieldComponentKey && field.String == component {
			hasComponent = true
			break
		}
	}

	if !hasComponent {
		t.Errorf("expected component field with value %q", component)
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()

	logger.Info(ctx, "test message",
		models.WithIntField("int_field", 42),
		models.WithFloatField("float_field", 3.14),
		models.WithStringField("string_field", "test"),
		models.WithObjectField("object_field", map[string]string{"key": "value"}))

	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	fields := logs[0].Fields
	if len(fields) < 4 {
		t.Errorf("expected at least 4 fields, got %d", len(fields))
	}

	fieldMap := make(map[string]*models.LogField)
	for _, field := range fields {
		fieldMap[field.Key] = field
	}

	if field, ok := fieldMap["int_field"]; !ok || field.Integer != 42 {
		t.Error("expected int_field with value 42")
	}

	if field, ok := fieldMap["float_field"]; !ok || field.Float != 3.14 {
		t.Error("expected float_field with value 3.14")
	}

	if field, ok := fieldMap["string_field"]; !ok || field.String != "test" {
		t.Error("expected string_field with value 'test'")
	}

	if _, ok := fieldMap["object_field"]; !ok {
		t.Error("expected object_field")
	}
}

func TestLogger_ContextValues(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.WithValue(context.Background(), models.AppID, "test-app")
	ctx = context.WithValue(ctx, models.EnvName, "test-env")

	logger.Info(ctx, "test message")
	logs := waitForLogs(mock, 1, time.Second)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Ctx == nil {
		t.Error("expected context to be set")
	}
}

func TestLogger_ConcurrentLogging(t *testing.T) {
	logger, mock, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	numGoroutines := 10
	logsPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info(ctx, fmt.Sprintf("message from goroutine %d-%d", id, j))
			}
		}(i)
	}

	wg.Wait()

	expectedLogs := numGoroutines * logsPerGoroutine
	logs := waitForLogs(mock, expectedLogs, 2*time.Second)

	if len(logs) != expectedLogs {
		t.Errorf("expected %d logs, got %d", expectedLogs, len(logs))
	}
}

func TestLogger_GracefulShutdown(t *testing.T) {
	loggerService := NewLoggerService()
	mock := &mockPublisher{logs: make([]*models.LogData, 0)}
	loggerService.AddLogger("mock", mock)
	loggerService.Start()
	logger := loggerService.NewLogger()

	ctx := context.Background()
	n := 50
	for i := 0; i < n; i++ {
		logger.Info(ctx, fmt.Sprintf("message %d", i))
	}

	loggerService.Stop()

	logs := mock.GetLogs()
	if len(logs) != n {
		t.Errorf("expected %d logs after graceful shutdown, got %d", n, len(logs))
	}
}

func TestLogger_DoubleStop(t *testing.T) {
	loggerService := NewLoggerService()
	mock := &mockPublisher{logs: make([]*models.LogData, 0)}
	loggerService.AddLogger("mock", mock)
	loggerService.Start()

	// Double Stop should not panic
	loggerService.Stop()
	loggerService.Stop()
}

func TestLogger_WriteAfterStop(t *testing.T) {
	loggerService := NewLoggerService()
	mock := &mockPublisher{logs: make([]*models.LogData, 0)}
	loggerService.AddLogger("mock", mock)
	loggerService.Start()
	logger := loggerService.NewLogger()

	loggerService.Stop()

	// Writing after Stop should not panic — message is silently dropped
	logger.Info(context.Background(), "after stop")
}

func TestLogger_PublisherPanic(t *testing.T) {
	loggerService := NewLoggerService(WithErrorHandler(func(err error) {}))
	panicPublisher := &mockPublisher{
		sendFunc: func(data *models.LogData) {
			panic("publisher crashed")
		},
	}
	loggerService.AddLogger("panic", panicPublisher)
	loggerService.Start()
	logger := loggerService.NewLogger()

	logger.Info(context.Background(), "trigger panic")

	// Stop should not deadlock despite publisher panic
	loggerService.Stop()
}

func BenchmarkLogger_Info(b *testing.B) {
	logger, _, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message")
	}
}

func BenchmarkLogger_ErrorWithStackTrace(b *testing.B) {
	logger, _, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()
	err := fmt.Errorf("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error(ctx, err, models.WithStackTrace())
	}
}

func BenchmarkLogger_WithMultipleFields(b *testing.B) {
	logger, _, service := setupTestLogger()
	defer service.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message",
			models.WithComponent("benchmark"),
			models.WithIntField("iteration", i),
			models.WithStringField("type", "benchmark"),
			models.WithFloatField("value", 3.14))
	}
}
