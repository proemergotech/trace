package gentlemantrace

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"gitlab.com/proemergotech/trace-go/v2"
	"gitlab.com/proemergotech/trace-go/v2/internal"
	"gopkg.in/h2non/gentleman.v2"
	gcontext "gopkg.in/h2non/gentleman.v2/context"
	"gopkg.in/h2non/gentleman.v2/plugin"
)

type settings struct {
	trace trace.Option
}

type Option func(*settings)

// Middleware return middleware which will extract the trace context from headers and starts a new child span.
// If there's no parent context, it will start a new root span and adds the 'span.missing' tag to the span.
// It will also add correlation and http related tags, like the http method, status code etc..
func Middleware(tracer opentracing.Tracer, logger trace.Logger, options ...Option) plugin.Plugin {
	s := &settings{
		trace: trace.StartWithWarning,
	}
	for _, opt := range options {
		opt(s)
	}

	before := func(gCtx *gcontext.Context, handler gcontext.Handler) {
		req := gCtx.Request
		ctx := req.Context()
		h := req.Header

		cor := trace.CorrelationFrom(ctx)
		if cor.CorrelationID == "" {
			err := errors.New(internal.MissingFromContext)
			logger.Error(ctx, err.Error(), "error", err, "method", req.Method, "url", req.URL.String())
			handler.Error(gCtx, err)
			return
		}

		h.Add(trace.CorrelationIDHeader, cor.CorrelationID)
		h.Add(trace.WorkflowIDHeader, cor.WorkflowID)

		host := req.Host
		if host == "" {
			host = req.URL.Host
		}

		msg := "HTTP out: [" + req.Method + "] " + host
		opts := []opentracing.StartSpanOption{ext.SpanKindRPCClient}
		if parent := opentracing.SpanFromContext(ctx); parent == nil {
			if s.trace == trace.Ignore {
				handler.Next(gCtx)
				return
			}
			if s.trace == trace.StartWithWarning {
				err := errors.New("No trace: " + msg)
				logger.Warn(ctx, err.Error(), "error", err)
			}

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

		trace.AddCorrelationTags(span, cor)
		ext.HTTPMethod.Set(span, req.Method)
		ext.HTTPUrl.Set(span, req.URL.String())

		ctx = opentracing.ContextWithSpan(ctx, span)
		err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(h))
		if err != nil {
			err = errors.Wrap(err, "Trace inject failed: "+msg)
			logger.Error(ctx, err.Error(), "error", err)
		}

		req = req.WithContext(ctx)
		gCtx.Request = req

		handler.Next(gCtx)
	}
	after := func(gCtx *gcontext.Context, handler gcontext.Handler) {
		defer handler.Next(gCtx)

		span := opentracing.SpanFromContext(gCtx.Request.Context())
		if span == nil {
			return
		}
		defer span.Finish()

		ext.HTTPStatusCode.Set(span, uint16(gCtx.Response.StatusCode))

		err := gCtx.Error
		if err != nil {
			trace.Error(span, err)
		}
	}

	return &plugin.Layer{Handlers: plugin.Handlers{
		"before dial": before,
		"response":    after,
		"error":       after,
	}}
}

// WithContext creates a new Request, combining req.Context and stdlib ctx.
// It will copy gentleman Context values, but also keep the values stored in the stdlib context.
// The already set deadline/timeout and cancel on the stdlib context will work too.
func WithContext(ctx context.Context, req *gentleman.Request) *gentleman.Request {
	ctx = context.WithValue(ctx, gcontext.Key, req.Context.GetAll())
	req.Context.Request = req.Context.Request.WithContext(ctx)

	return req
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
