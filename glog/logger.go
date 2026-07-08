package glog

import (
	"context"
	"fmt"
	"github.com/alexnobleburn/glogger/glog/interfaces"
	"github.com/alexnobleburn/glogger/glog/models"
	"github.com/pkg/errors"
	"strings"
	"sync/atomic"
)

// Compile-time check that Logger implements interfaces.Logger.
var _ interfaces.Logger = (*Logger)(nil)

type Logger struct {
	logChan chan<- *models.LogData
	stopped *atomic.Bool
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
	logData := &models.LogData{
		Ctx:    ctx,
		Msg:    err.Error(),
		Fields: []*models.LogField{},
		Level:  models.ErrorLevel,
	}

	if opts.WithStackTrace() {
		extendedErr := errors.WithStack(err)
		var fileNames []string
		if stackTracerErr, ok := extendedErr.(interfaces.StackTracer); ok {
			stacktrace := stackTracerErr.StackTrace()
			if len(stacktrace) > 0 {
				for i := 1; i < len(stacktrace); i++ {
					fileNames = append(fileNames, fmt.Sprintf("%+v", stacktrace[i]))
				}
			}
		}
		logData.Fields = append(logData.Fields,
			&models.LogField{Key: models.FieldFilenameKey, Type: models.FieldTypeString, String: strings.Join(fileNames, " <- ")})
	}

	if len(opts.GetFields()) > 0 {
		logData.Fields = append(logData.Fields, opts.GetFields()...)
	}
	if opts.GetComponent() != "" {
		logData.Fields = append(logData.Fields,
			&models.LogField{Key: models.FieldComponentKey, Type: models.FieldTypeString, String: opts.GetComponent()})
	}

	l.sendData(logData)
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

	logData := &models.LogData{
		Ctx:    ctx,
		Msg:    message,
		Fields: opts.GetFields(),
		Level:  level,
	}

	if opts.GetComponent() != "" {
		logData.Fields = append(logData.Fields,
			&models.LogField{Key: models.FieldComponentKey, Type: models.FieldTypeString, String: opts.GetComponent()})
	}

	l.sendData(logData)
}

func (l *Logger) sendData(logData *models.LogData) {
	if l.stopped != nil && l.stopped.Load() {
		return
	}
	select {
	case l.logChan <- logData:
	default:
		// Channel full — drop the message to maintain non-blocking guarantee.
	}
}
