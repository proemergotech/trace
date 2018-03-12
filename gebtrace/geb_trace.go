package gebtrace

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/geb-client-go/geb"
	"gitlab.com/proemergotech/trace-go"
)

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
// It will also add correlation related tags.
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func OnEventMiddleware(tracer opentracing.Tracer, logger trace.Logger) geb.Middleware {
	return func(e *geb.Event, next func(*geb.Event) error) (err error) {
		ctx := e.Context()
		h := e.Headers()

		cor := &trace.Correlation{
			CorrelationID: h[trace.CorrelationIDField],
			WorkflowID:    h[trace.WorkflowIDField],
		}
		ctx = trace.WithCorrelation(ctx, cor)

		msg := "GEB in: " + e.EventName()
		opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
		if spanCtx, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(h)); err != nil {
			err = errors.Wrap(err, "No trace: "+msg)
			logger.Error(ctx, err.Error(), "error", err)

			opts = append(opts, opentracing.Tag{
				Key:   trace.SpanMissingTag,
				Value: true,
			})
		} else {
			opts = append(opts, opentracing.FollowsFrom(spanCtx))
		}

		span := tracer.StartSpan(
			msg,
			opts...,
		)
		defer span.Finish()

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
