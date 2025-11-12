package interfaces

import (
	"context"
	"github.com/alexnobleburn/glogger/glog/models"
)

type Logger interface {
	Error(ctx context.Context, err error, options ...models.Option)
	Errors(ctx context.Context, errs []error, options ...models.Option)
	Info(ctx context.Context, message string, options ...models.Option)
	Warning(ctx context.Context, message string, options ...models.Option)
	Debug(ctx context.Context, message string, options ...models.Option)
}
