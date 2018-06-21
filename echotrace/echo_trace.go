package echotrace

import (
	"github.com/labstack/echo"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/trace-go"
)

type settings struct {
	start  bool
	corGen func() *trace.Correlation
}

type Option func(*settings)

// Middleware return middleware which will extract the trace context from headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// If the Start option is passed to the method, it will start a new root span, without adding the 'span.missing' tag.
// It will also add correlation and http related tags, like the http method, status code etc..
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func Middleware(tracer opentracing.Tracer, logger trace.Logger, options ...Option) echo.MiddlewareFunc {
	s := &settings{}
	for _, opt := range options {
		opt(s)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(eCtx echo.Context) error {
			req := eCtx.Request()
			ctx := req.Context()
			h := req.Header

			cor := &trace.Correlation{
				CorrelationID: h.Get(trace.CorrelationIDHeader),
				WorkflowID:    h.Get(trace.WorkflowIDHeader),
			}
			if cor.CorrelationID == "" && s.start {
				cor = s.corGen()
			}
			ctx = trace.WithCorrelation(ctx, cor)

			msg := "HTTP in: [" + req.Method + "] " + req.URL.Path
			opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
			spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(h))
			if err == nil {
				opts = append(opts, opentracing.ChildOf(spanCtx))
			}

			span := tracer.StartSpan(msg, opts...)
			defer span.Finish()

			if s.start && err == nil {
				err = errors.New("Trace found, ignoring Start: " + msg)
				logger.Error(ctx, err.Error(), "error", err)
				span.SetTag(trace.StartIgnoredTag, true)
			} else if !s.start && err != nil {
				err = errors.Wrap(err, "No trace: "+msg)
				logger.Error(ctx, err.Error(), "error", err)
				span.SetTag(trace.SpanMissingTag, true)
			}

			trace.AddCorrelationTags(span, cor)
			ext.HTTPMethod.Set(span, req.Method)
			ext.HTTPUrl.Set(span, req.URL.String())

			ctx = opentracing.ContextWithSpan(ctx, span)
			eCtx.Request().WithContext(ctx)

			defer func() {
				if err := recover(); err != nil {
					trace.Error(span, errors.Errorf("panic during request handling: %+v", err))
					panic(err)
				}
			}()

			err = next(eCtx)

			ext.HTTPStatusCode.Set(span, uint16(eCtx.Response().Status))
			if err != nil {
				trace.Error(span, err)
			}

			return err
		}
	}
}

// Start option can be passed to Middleware to start the tracing
// instead of following it from a previous trace based on the http headers.
// A generator function is required for generation the Correlation object when starting a trace.
func Start(gen func() *trace.Correlation) Option {
	return func(opts *settings) {
		opts.start = true
		opts.corGen = gen
	}
}
