package zap

import (
	"context"
	"github.com/alexnobleburn/glogger/glog/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type CtxLoggerKey string

const (
	timeTag = "timestamp"
)

type Logger struct {
	*zap.Logger
	appID string
	env   string
}

func NewZapLogger(appID, env string) *Logger {
	config := getEncoderConfig()
	coreConsole := zapcore.NewCore(zapcore.NewJSONEncoder(config), os.Stdout, getAllLevelFunc())
	zapLogger := zap.New(zapcore.NewTee(coreConsole))

	return &Logger{
		Logger: zapLogger,
		appID:  appID,
		env:    env,
	}
}

func (l *Logger) SendMsg(logData *models.LogData) {
	if logData.Ctx == nil {
		logData.Ctx = context.Background()
	}
	appID, ok := logData.Ctx.Value(models.AppID).(string)
	if !ok || appID == "" {
		appID = l.appID
	}
	env, ok := logData.Ctx.Value(models.EnvName).(string)
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
		l.Error(logData.Msg, fields...)
	case models.WarnLevel:
		l.Warn(logData.Msg, fields...)
	case models.InfoLevel:
		l.Info(logData.Msg, fields...)
	case models.DebugLevel:
		l.Debug(logData.Msg, fields...)
	case models.FatalLevel:
		l.Fatal(logData.Msg, fields...)
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
