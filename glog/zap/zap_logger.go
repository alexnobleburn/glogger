package zap

import (
	"context"
	"github.com/alexnobleburn/glogger/glog/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
)

type CtxLoggerKey string

const (
	timeTag = "timestamp"
)

type Logger struct {
	zl    *zap.Logger
	appID string
	env   string
}

func NewZapLogger(appID, env string) *Logger {
	return newLogger(appID, env, os.Stdout)
}

// NewZapLoggerWithWriter creates a Logger that writes to the given writer (useful for tests).
func NewZapLoggerWithWriter(appID, env string, w io.Writer) *Logger {
	return newLogger(appID, env, zapcore.AddSync(w))
}

func newLogger(appID, env string, ws zapcore.WriteSyncer) *Logger {
	config := getEncoderConfig()
	core := zapcore.NewCore(zapcore.NewJSONEncoder(config), ws, getAllLevelFunc())
	zapLogger := zap.New(zapcore.NewTee(core))

	return &Logger{
		zl:    zapLogger,
		appID: appID,
		env:   env,
	}
}

func (l *Logger) SendMsg(logData *models.LogData) {
	ctx := logData.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	appID, ok := ctx.Value(models.AppID).(string)
	if !ok || appID == "" {
		appID = l.appID
	}
	env, ok := ctx.Value(models.EnvName).(string)
	if !ok || env == "" {
		env = l.env
	}

	fields := []zapcore.Field{
		zap.String("service_name", appID),
		zap.String("env", env),
	}

	resFields := l.getPayloadFields(logData)
	fields = append(fields, resFields...)

	switch logData.Level {
	case models.ErrorLevel:
		l.zl.Error(logData.Msg, fields...)
	case models.WarnLevel:
		l.zl.Warn(logData.Msg, fields...)
	case models.InfoLevel:
		l.zl.Info(logData.Msg, fields...)
	case models.DebugLevel:
		l.zl.Debug(logData.Msg, fields...)
	case models.DPanicLevel:
		l.zl.DPanic(logData.Msg, fields...)
	case models.PanicLevel:
		l.zl.Panic(logData.Msg, fields...)
	case models.FatalLevel:
		l.zl.Fatal(logData.Msg, fields...)
	default:
		l.zl.Info(logData.Msg, fields...)
	}
}

func (l *Logger) getPayloadFields(logData *models.LogData) []zap.Field {
	var resFields []zap.Field
	resFields = append(resFields, zap.Namespace("payload"))
	for _, f := range logData.Fields {
		switch f.Type {
		case models.FieldTypeInt:
			resFields = append(resFields, zap.Int(f.Key, f.Integer))
		case models.FieldTypeString:
			resFields = append(resFields, zap.String(f.Key, f.String))
		case models.FieldTypeFloat:
			resFields = append(resFields, zap.Float64(f.Key, f.Float))
		case models.FieldTypeObject:
			resFields = append(resFields, zap.Any(f.Key, f.Object))
		case models.FieldTypeBool:
			resFields = append(resFields, zap.Bool(f.Key, f.Bool))
		}
	}
	return resFields
}

func getEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = timeTag
	config.EncodeTime = zapcore.RFC3339TimeEncoder
	return config
}

func getAllLevelFunc() zap.LevelEnablerFunc {
	return func(l zapcore.Level) bool { return true }
}
