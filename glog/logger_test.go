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

func setupTestLogger() (*Logger, *mockPublisher, chan struct{}) {
	stopCh := make(chan struct{})
	loggerService := NewLoggerService(stopCh)

	mock := &mockPublisher{logs: make([]*models.LogData, 0)}
	loggerService.AddLogger("mock", mock)
	loggerService.Start()

	logger := NewLogger(loggerService.GetInputChan())

	return logger, mock, stopCh
}

func TestLogger_Info(t *testing.T) {
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	message := "test info message"

	logger.Info(ctx, message)
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
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
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	err := fmt.Errorf("test error")

	logger.Error(ctx, err)
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
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
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	err := fmt.Errorf("test error with stack")

	logger.Error(ctx, err, models.WithStackTrace())
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	// Check if filename field exists
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
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	message := "test warning"

	logger.Warning(ctx, message)
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Level != models.WarnLevel {
		t.Errorf("expected WarnLevel, got %v", logs[0].Level)
	}
}

func TestLogger_Debug(t *testing.T) {
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	message := "test debug"

	logger.Debug(ctx, message)
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Level != models.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", logs[0].Level)
	}
}

func TestLogger_Errors(t *testing.T) {
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	errs := []error{
		fmt.Errorf("error 1"),
		fmt.Errorf("error 2"),
		fmt.Errorf("error 3"),
	}

	logger.Errors(ctx, errs)
	time.Sleep(100 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != len(errs) {
		t.Fatalf("expected %d logs, got %d", len(errs), len(logs))
	}

	// Check that all error messages are present (order not guaranteed due to async)
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

	// Verify all expected messages were found
	for msg, found := range expectedMessages {
		if !found {
			t.Errorf("expected message %q not found in logs", msg)
		}
	}
}

func TestLogger_WithComponent(t *testing.T) {
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	component := "test-component"

	logger.Info(ctx, "test message", models.WithComponent(component))
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
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
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()

	logger.Info(ctx, "test message",
		models.WithIntField("int_field", 42),
		models.WithFloatField("float_field", 3.14),
		models.WithStringField("string_field", "test"),
		models.WithObjectField("object_field", map[string]string{"key": "value"}))

	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	fields := logs[0].Fields
	if len(fields) < 4 {
		t.Errorf("expected at least 4 fields, got %d", len(fields))
	}

	// Verify fields exist
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
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.WithValue(context.Background(), models.AppID, "test-app")
	ctx = context.WithValue(ctx, models.EnvName, "test-env")

	logger.Info(ctx, "test message")
	time.Sleep(50 * time.Millisecond)

	logs := mock.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	// Context values should be passed through
	if logs[0].Ctx == nil {
		t.Error("expected context to be set")
	}
}

func TestLogger_ConcurrentLogging(t *testing.T) {
	logger, mock, stopCh := setupTestLogger()
	defer close(stopCh)

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
	time.Sleep(100 * time.Millisecond)

	logs := mock.GetLogs()
	expectedLogs := numGoroutines * logsPerGoroutine
	if len(logs) != expectedLogs {
		t.Errorf("expected %d logs, got %d", expectedLogs, len(logs))
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger, _, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message")
	}
}

func BenchmarkLogger_ErrorWithStackTrace(b *testing.B) {
	logger, _, stopCh := setupTestLogger()
	defer close(stopCh)

	ctx := context.Background()
	err := fmt.Errorf("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error(ctx, err, models.WithStackTrace())
	}
}

func BenchmarkLogger_WithMultipleFields(b *testing.B) {
	logger, _, stopCh := setupTestLogger()
	defer close(stopCh)

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
