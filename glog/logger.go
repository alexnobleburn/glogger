package glog

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog/interfaces"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/pkg/errors"
	"strings"
)

type Logger struct {
	logChan chan<- *models.LogData
}

func NewLogger(logChan chan<- *models.LogData) *Logger {
	return &Logger{logChan: logChan}
}

func (l *Logger) Error(ctx context.Context, err error, options ...models.Option) {
	opts := &models.Options{}
	for _, opt := range options {
		opt(opts)
	}
	l.error(ctx, err, opts)
}

func (l *Logger) Errors(ctx context.Context, errs []error, options ...models.Option) {
	opts := &models.Options{}
	for _, opt := range options {
		opt(opts)
	}
	for _, err := range errs {
		l.error(ctx, err, opts)
	}
}

func (l *Logger) error(ctx context.Context, err error, opts *models.Options) {
	extendedErr := errors.WithStack(err)
	logData := &models.LogData{
		Ctx:    ctx,
		Msg:    extendedErr.Error(),
		Fields: []*models.LogField{},
		Level:  models.ErrorLevel,
	}

	if opts.WithStackTrace() {
		var fileNames []string
		if stackTracerErr, ok := extendedErr.(interfaces.StackTracer); ok {
			stacktrace := stackTracerErr.StackTrace()
			if len(stacktrace) > 0 {
				for i := 1; i < len(stacktrace); i++ {
					fileNames = append(fileNames, fmt.Sprintf("%s:%d", stacktrace[i], stacktrace[i]))
				}
			}
		}
		logData.Fields = append(logData.Fields,
			&models.LogField{Key: models.FieldFilenameKey, String: strings.Join(fileNames, " <- ")})
	}

	if len(opts.GetFields()) > 0 {
		logData.Fields = append(logData.Fields, opts.GetFields()...)
	}
	if opts.GetComponent() != "" {
		logData.Fields = append(logData.Fields,
			&models.LogField{Key: models.FieldComponentKey, String: opts.GetComponent()})
	}

	go l.sendData(logData)
}

func (l *Logger) Warning(ctx context.Context, message string, options ...models.Option) {
	l.logMsg(ctx, models.WarnLevel, message, options...)
}

func (l *Logger) Info(ctx context.Context, message string, options ...models.Option) {
	l.logMsg(ctx, models.InfoLevel, message, options...)
}

func (l *Logger) Debug(ctx context.Context, message string, options ...models.Option) {
	l.logMsg(ctx, models.DebugLevel, message, options...)
}

func (l *Logger) logMsg(ctx context.Context, level models.LogLevel, message string, options ...models.Option) {
	opts := &models.Options{}
	for _, opt := range options {
		opt(opts)
	}

	logMsg := models.NewLogMessage(level, message).
		SetComponent(opts.GetComponent()).
		SetFields(opts.GetFields()...)
	logData := &models.LogData{
		Ctx: ctx,
		Msg: logMsg.Message,
		Fields: append(
			logMsg.Fields,
			&models.LogField{Key: models.FieldComponentKey, String: logMsg.Component},
		),
		Level: logMsg.Level,
	}

	go l.sendData(logData)
}

func (l *Logger) sendData(logData *models.LogData) {
	l.logChan <- logData
}
