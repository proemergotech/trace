package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	CorrelationIDHeader  = "X-Correlation-Id"
	WorkflowIDHeader     = "X-Workflow-Id"
	CorrelationIDField   = "x_correlation_id"
	CorrelationIDMissing = "x_correlation_id.missing"
	WorkflowIDField      = "x_workflow_id"
	WorkflowIDMissing    = "x_workflow_id.missing"
	SpanMissingTag       = "span.missing"
	StartIgnoredTag      = "start.ignored"
)

type correlationKey struct{}
type contextMapper struct{}

type Correlation struct {
	CorrelationID string
	WorkflowID    string
}

// Logger is a simplified interface for logging, mainly used to decouple tracing from logging.
type Logger interface {
	Error(ctx context.Context, msg string, keysAndValues ...interface{})
}

// ContextMapper used to extract values from a context.
type ContextMapper interface {
	Values(ctx context.Context) map[string]string
}

var defaultContextMapper = contextMapper{}

// Mapper returns a ContextMapper which will extract Correlation-Id and Workflow-Id from context.
func Mapper() ContextMapper {
	return defaultContextMapper
}

func (cl contextMapper) Values(ctx context.Context) map[string]string {
	cor := CorrelationFrom(ctx)

	return map[string]string{
		CorrelationIDField: cor.CorrelationID,
		WorkflowIDField:    cor.WorkflowID,
	}
}

// NewCorrelation returns a new Correlation object with generated CorrelationID and empty WorkflowID.
func NewCorrelation() *Correlation {
	return &Correlation{
		CorrelationID: NewCorrelationID(),
	}
}

// NewCorrelationID generates a new correlation id consisting of 32 random hexadecimal characters.
func NewCorrelationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

// WithCorrelation create a new context with the passed correlation in it.
// If the ctx parameter is nil, context.Background() used instead,
// but it's a good practice to never pass nil context, use context.Background() instead.
func WithCorrelation(ctx context.Context, c *Correlation) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, correlationKey{}, c)
}

// CorrelationFrom returns the Correlation from context or an empty Correlation if not exists.
func CorrelationFrom(ctx context.Context) *Correlation {
	c, ok := ctx.Value(correlationKey{}).(*Correlation)
	if !ok {
		return &Correlation{}
	}

	return c
}

// AddCorrelationTags add Correlation related tags to the span.
// It will add Correlation-Id and Workflow-Id if they are exists
// or a related "*.missing" tag if they don't.
func AddCorrelationTags(span opentracing.Span, cor *Correlation) {
	span.SetTag(CorrelationIDField, cor.CorrelationID)
	span.SetTag(WorkflowIDField, cor.WorkflowID)

	if cor.CorrelationID == "" {
		span.SetTag(CorrelationIDMissing, true)
	}

	if cor.WorkflowID == "" {
		span.SetTag(WorkflowIDMissing, true)
	}
}

// Error marks the span as failed and set 'sampling.priority' to 1.
// Also collect the fields from the error and log them to the span, along with the error itself.
// The error will be logged under the 'error.object' tag.
func Error(span opentracing.Span, err error) {
	ext.Error.Set(span, true)
	ext.SamplingPriority.Set(span, 1)
	span.LogKV(fields(err)...)
}

func fields(err error) []interface{} {
	type causer interface {
		Cause() error
	}

	type fielder interface {
		Fields() []interface{}
	}

	f := []interface{}{"error.object", err.Error()}
	for err != nil {
		if fErr, ok := err.(fielder); ok {
			f = append(f, fErr.Fields()...)
		}

		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}

	return f
}
