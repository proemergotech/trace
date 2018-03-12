package gintrace

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/trace-go"
)

// Middleware return middleware which will extract the trace context from headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// It will also add correlation and http related tags, like the http method, status code etc..
// If an error happens or one of the middleware panics, it will mark the span as failed and continue panicking.
func Middleware(tracer opentracing.Tracer, logger trace.Logger) gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := gCtx.Request
		ctx := req.Context()
		h := req.Header

		cor := &trace.Correlation{
			CorrelationID: h.Get(trace.CorrelationIDHeader),
			WorkflowID:    h.Get(trace.WorkflowIDHeader),
		}
		ctx = trace.WithCorrelation(ctx, cor)

		msg := "HTTP in: [" + req.Method + "] " + req.URL.Path
		opts := []opentracing.StartSpanOption{ext.SpanKindConsumer}
		if spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(h)); err != nil {
			err = errors.Wrap(err, "No trace: "+msg)
			logger.Error(ctx, err.Error(), "error", err)

			opts = append(opts, opentracing.Tag{
				Key:   trace.SpanMissingTag,
				Value: true,
			})
		} else {
			opts = append(opts, opentracing.ChildOf(spanCtx))
		}

		span := tracer.StartSpan(
			msg,
			opts...,
		)
		defer span.Finish()

		trace.AddCorrelationTags(span, cor)
		ext.HTTPMethod.Set(span, req.Method)
		ext.HTTPUrl.Set(span, req.URL.String())

		ctx = opentracing.ContextWithSpan(ctx, span)
		req = req.WithContext(ctx)
		gCtx.Request = req

		defer func() {
			ext.HTTPStatusCode.Set(span, uint16(gCtx.Writer.Status()))

			for _, e := range gCtx.Errors {
				trace.Error(span, e.Err)
			}

			if err := recover(); err != nil {
				trace.Error(span, errors.Errorf("panic during request handling: %+v", err))
				panic(err)
			}
		}()

		gCtx.Next()
	}
}
