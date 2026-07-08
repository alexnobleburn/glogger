package zap

import (
	"context"
	"github.com/alexnobleburn/glogger/glog/models"
	"io"
	"testing"
)

func TestNewZapLogger(t *testing.T) {
	appID := "test-app"
	env := "test-env"

	logger := NewZapLogger(appID, env)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	if logger.appID != appID {
		t.Errorf("expected appID %q, got %q", appID, logger.appID)
	}

	if logger.env != env {
		t.Errorf("expected env %q, got %q", env, logger.env)
	}

	if logger.zl == nil {
		t.Error("expected non-nil zap.Logger")
	}
}

func TestZapLogger_SendMsg_InfoLevel(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test info message",
		Level: models.InfoLevel,
		Fields: []*models.LogField{
			{Key: "test_field", Type: models.FieldTypeString, String: "test_value"},
		},
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_ErrorLevel(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test error message",
		Level: models.ErrorLevel,
		Fields: []*models.LogField{
			{Key: "error", Type: models.FieldTypeString, String: "something went wrong"},
		},
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_WarnLevel(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test warning message",
		Level: models.WarnLevel,
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_DebugLevel(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test debug message",
		Level: models.DebugLevel,
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_WithFields(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test message with fields",
		Level: models.InfoLevel,
		Fields: []*models.LogField{
			{Key: "int_field", Type: models.FieldTypeInt, Integer: 42},
			{Key: "float_field", Type: models.FieldTypeFloat, Float: 3.14},
			{Key: "string_field", Type: models.FieldTypeString, String: "test"},
			{Key: "bool_field", Type: models.FieldTypeBool, Bool: true},
		},
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_NilContext(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   nil,
		Msg:   "test message",
		Level: models.InfoLevel,
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_ContextValues(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	ctx := context.WithValue(context.Background(), models.AppID, "custom-app")
	ctx = context.WithValue(ctx, models.EnvName, "custom-env")

	logData := &models.LogData{
		Ctx:   ctx,
		Msg:   "test message with context",
		Level: models.InfoLevel,
	}

	logger.SendMsg(logData)
}

func TestZapLogger_SendMsg_EmptyContextValues(t *testing.T) {
	logger := NewZapLoggerWithWriter("default-app", "default-env", io.Discard)

	ctx := context.WithValue(context.Background(), models.AppID, "")
	ctx = context.WithValue(ctx, models.EnvName, "")

	logData := &models.LogData{
		Ctx:   ctx,
		Msg:   "test message",
		Level: models.InfoLevel,
	}

	logger.SendMsg(logData)
}

func TestZapLogger_GetPayloadFields(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test",
		Level: models.InfoLevel,
		Fields: []*models.LogField{
			{Key: "int_field", Type: models.FieldTypeInt, Integer: 42},
			{Key: "float_field", Type: models.FieldTypeFloat, Float: 3.14},
			{Key: "string_field", Type: models.FieldTypeString, String: "test"},
		},
	}

	fields := logger.getPayloadFields(logData)

	// Should have namespace + 3 fields
	if len(fields) < 4 {
		t.Errorf("expected at least 4 fields (namespace + 3), got %d", len(fields))
	}
}

func TestZapLogger_GetPayloadFields_EmptyFields(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:    context.Background(),
		Msg:    "test",
		Level:  models.InfoLevel,
		Fields: []*models.LogField{},
	}

	fields := logger.getPayloadFields(logData)

	if len(fields) < 1 {
		t.Error("expected at least namespace field")
	}
}

func TestZapLogger_GetPayloadFields_ZeroValues(t *testing.T) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "test",
		Level: models.InfoLevel,
		Fields: []*models.LogField{
			{Key: "int_field", Type: models.FieldTypeInt, Integer: 0},
			{Key: "float_field", Type: models.FieldTypeFloat, Float: 0.0},
			{Key: "string_field", Type: models.FieldTypeString, String: ""},
		},
	}

	fields := logger.getPayloadFields(logData)

	if len(fields) != 4 {
		t.Errorf("expected 4 fields (namespace + 3 zero-value fields), got %d fields", len(fields))
	}
}

func BenchmarkZapLogger_SendMsg(b *testing.B) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "benchmark message",
		Level: models.InfoLevel,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.SendMsg(logData)
	}
}

func BenchmarkZapLogger_SendMsg_WithFields(b *testing.B) {
	logger := NewZapLoggerWithWriter("test-app", "test", io.Discard)

	logData := &models.LogData{
		Ctx:   context.Background(),
		Msg:   "benchmark message",
		Level: models.InfoLevel,
		Fields: []*models.LogField{
			{Key: "field1", Type: models.FieldTypeString, String: "value1"},
			{Key: "field2", Type: models.FieldTypeInt, Integer: 42},
			{Key: "field3", Type: models.FieldTypeFloat, Float: 3.14},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.SendMsg(logData)
	}
}
