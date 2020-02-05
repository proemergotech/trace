package gebtrace

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/geb-client-go/v2/geb"
	"gitlab.com/proemergotech/trace-go/v2"
	"gitlab.com/proemergotech/trace-go/v2/internal"
)

type settings struct {
	genCor   bool
	genCorFn func() *trace.Correlation
	trace    trace.Option
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
		if cor.CorrelationID == "" {
			err := errors.New(internal.MissingFromContext)
			logger.Error(ctx, err.Error(), "error", err, "event_name", e.EventName())

			return err
		}

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
	s := &settings{
		trace: trace.StartWithWarning,
	}
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

		if cor.CorrelationID == "" {
			if s.genCor {
				cor = s.genCorFn()
			} else {
				err := errors.New(internal.MissingGebHeader)
				logger.Error(ctx, err.Error(), "error", err, "event_name", e.EventName())

				return err
			}
		}
		ctx = trace.WithCorrelation(ctx, cor)

		// no parent trace, but no need to start, so just ignore tracing at all
		if err != nil && s.trace == trace.Ignore {
			return next(e)
		}

		opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
		spanCtx, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(h))
		if err == nil {
			opts = append(opts, opentracing.FollowsFrom(spanCtx))
		}

		msg := "GEB in: " + e.EventName()
		span := tracer.StartSpan(
			msg,
			opts...,
		)
		defer span.Finish()

		// if we don't have a trace and ignore is false, we expected a parent trace, so log that we don't found one
		if err != nil && s.trace == trace.StartWithWarning {
			err = errors.New("No trace: " + msg)
			logger.Warn(ctx, err.Error(), "error", err)
			span.SetTag(trace.StartIgnoredTag, true)
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

// GenerateCorrelation option can be passed to OnEventMiddleware to generate correlation when it's missing.
// A generator function is required for generation the Correlation object when starting a trace.
// Usually just use trace.NewCorrelation.
func GenerateCorrelation(gen func() *trace.Correlation) Option {
	return func(opts *settings) {
		opts.genCor = true
		opts.genCorFn = gen
	}
}

// Trace option can be passed to Middleware to handle cases when a parent trace not found.
// Check the possible options for more information.
func Trace(option trace.Option) (Option, error) {
	if err := trace.ValidateOption(option); err != nil {
		return nil, err
	}

	return func(opts *settings) {
		opts.trace = option
	}, nil
}
