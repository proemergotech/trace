package gintrace

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/trace-go"
	"gitlab.com/proemergotech/trace-go/internal"
)

type settings struct {
	genCor   bool
	genCorFn func() *trace.Correlation
	trace    trace.Option
}

type Option func(*settings)

// Middleware return middleware which will extract the trace context from headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// If the Start option is passed to the method, it will start a new root span, without adding the 'span.missing' tag.
// It will also add correlation and http related tags, like the http method, status code etc..
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func Middleware(tracer opentracing.Tracer, logger trace.Logger, options ...Option) gin.HandlerFunc {
	s := &settings{
		trace: trace.StartWithWarning,
	}
	for _, opt := range options {
		opt(s)
	}

	return func(gCtx *gin.Context) {
		req := gCtx.Request
		ctx := req.Context()
		h := req.Header

		cor := &trace.Correlation{
			CorrelationID: h.Get(trace.CorrelationIDHeader),
			WorkflowID:    h.Get(trace.WorkflowIDHeader),
		}

		if cor.CorrelationID == "" {
			if s.genCor {
				cor = s.genCorFn()
			} else {
				httpErr := internal.CorrelationIDMissing()
				logger.Error(ctx, httpErr.Error.Error(), "error", errors.WithStack(&httpErr.Error), "method", req.Method, "url", req.URL.String())
				gCtx.AbortWithStatusJSON(http.StatusBadRequest, httpErr)
				return
			}
		}
		ctx = trace.WithCorrelation(ctx, cor)

		spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(h))

		// no parent trace, but no need to start, so just ignore tracing at all
		if err != nil && s.trace == trace.Ignore {
			gCtx.Next()
			return
		}

		opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
		if err == nil {
			opts = append(opts, opentracing.ChildOf(spanCtx))
		}

		msg := "HTTP in: [" + req.Method + "] " + req.URL.Path
		span := tracer.StartSpan(msg, opts...)
		defer span.Finish()

		if err != nil && s.trace == trace.StartWithWarning {
			err = errors.New("No trace: " + msg)
			logger.Warn(ctx, err.Error(), "error", err)
			span.SetTag(trace.StartIgnoredTag, true)
		}

		trace.AddCorrelationTags(span, cor)
		ext.HTTPMethod.Set(span, req.Method)
		ext.HTTPUrl.Set(span, req.URL.String())

		ctx = opentracing.ContextWithSpan(ctx, span)
		req = req.WithContext(ctx)
		gCtx.Request = req

		defer func() {
			ext.HTTPStatusCode.Set(span, uint16(gCtx.Writer.Status()))
			if len(gCtx.Errors) != 0 {
				// at least add the first error, others will be logged
				trace.Error(span, gCtx.Errors[1])
			}

			if err := recover(); err != nil {
				trace.Error(span, errors.Errorf("panic during request handling: %+v", err))
				panic(err)
			}
		}()

		gCtx.Next()
	}
}

// GenerateCorrelation option can be passed to Middleware to generate correlation when it's missing.
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
