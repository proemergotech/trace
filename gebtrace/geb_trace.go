package gebtrace

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/geb-client-go/geb"
	"gitlab.com/proemergotech/trace-go"
)

type settings struct {
	start  bool
	corGen func() *trace.Correlation
}

type Option func(*settings)

// PublishMiddleware return middleware which will extract the trace context from event headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// It will also add correlation related tags.
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func PublishMiddleware(tracer opentracing.Tracer, logger trace.Logger) geb.Middleware {
	return func(e *geb.Event, next func(*geb.Event) error) (err error) {
		ctx := e.Context()
		h := e.Headers()

		cor := trace.CorrelationFrom(ctx)
		h[trace.CorrelationIDField] = cor.CorrelationID
		h[trace.WorkflowIDField] = cor.WorkflowID

		msg := "GEB out:" + e.EventName()
		opts := []opentracing.StartSpanOption{ext.SpanKindProducer}
		if parent := opentracing.SpanFromContext(ctx); parent == nil {
			err := errors.New("No trace: " + msg)
			logger.Error(ctx, err.Error(), "error", err)

			opts = append(opts, opentracing.Tag{
				Key:   trace.SpanMissingTag,
				Value: true,
			})
		} else {
			opts = append(opts, opentracing.ChildOf(parent.Context()))
		}

		span := tracer.StartSpan(
			msg,
			opts...,
		)
		defer span.Finish()

		trace.AddCorrelationTags(span, cor)

		ctx = opentracing.ContextWithSpan(ctx, span)
		err = tracer.Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(h))
		if err != nil {
			err = errors.Wrap(err, "Trace inject failed: "+msg)
			logger.Error(ctx, err.Error(), "error", err)
		}

		e.SetHeaders(h)
		e.SetContext(ctx)

		defer func() {
			if err != nil {
				trace.Error(span, err)
			}

			if err := recover(); err != nil {
				trace.Error(span, errors.Errorf("panic during publish: %+v", err))
				panic(err)
			}
		}()

		return next(e)
	}
}

// OnEventMiddleware return middleware which will extract the trace context from event headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// If the Start option is passed to the method, it will start a new root span, without adding the 'span.missing' tag.
// It will also add correlation related tags.
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func OnEventMiddleware(tracer opentracing.Tracer, logger trace.Logger, options ...Option) geb.Middleware {
	s := &settings{}
	for _, opt := range options {
		opt(s)
	}

	return func(e *geb.Event, next func(*geb.Event) error) (err error) {
		ctx := e.Context()
		h := e.Headers()

		cor := &trace.Correlation{
			CorrelationID: h[trace.CorrelationIDField],
			WorkflowID:    h[trace.WorkflowIDField],
		}
		if cor.CorrelationID == "" && s.start {
			cor = s.corGen()
		}
		ctx = trace.WithCorrelation(ctx, cor)

		msg := "GEB in: " + e.EventName()
		opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
		spanCtx, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(h))
		if err == nil {
			opts = append(opts, opentracing.FollowsFrom(spanCtx))
		}

		span := tracer.StartSpan(
			msg,
			opts...,
		)
		defer span.Finish()

		if s.start && err == nil {
			err = errors.New("Trace found, ignoring Start: "+msg)
			logger.Error(ctx, err.Error(), "error", err)
			trace.Error(span, err)
		} else if !s.start && err != nil{
			err = errors.Wrap(err, "No trace: "+msg)
			logger.Error(ctx, err.Error(), "error", err)
			span.SetTag(trace.SpanMissingTag, true)
			trace.Error(span, err)
		}

		trace.AddCorrelationTags(span, cor)

		ctx = opentracing.ContextWithSpan(ctx, span)

		e.SetHeaders(h)
		e.SetContext(ctx)

		defer func() {
			if err != nil {
				trace.Error(span, err)
			}

			if err := recover(); err != nil {
				trace.Error(span, errors.Errorf("panic during onEvent: %+v", err))
				panic(err)
			}
		}()

		return next(e)
	}
}

// Start option can be passed to OnEventMiddleware to start the tracing
// instead of following it from a previous trace based on the geb headers.
// A generator function is required for generation the Correlation object when starting a trace.
func Start(gen func() *trace.Correlation) Option {
	return func(opts *settings) {
		opts.start = true
		opts.corGen = gen
	}
}
