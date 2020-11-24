package internal

import (
	"github.com/proemergotech/trace/v2"
)

const (
	MissingFromContext = "correlation is missing from context"
	missingHttpHeader  = "'" + trace.CorrelationIDHeader + "'" + " header is missing"
	fieldPrefix        = "trace_"
)

type HTTPError struct {
	Error Error `json:"error"`
}

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func CorrelationIDMissing() *HTTPError {
	return &HTTPError{
		Error: Error{
			Message: missingHttpHeader,
			Code:    "ERR_CORRELATION_ID_MISSING",
		},
	}
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Fields() []interface{} {
	return []interface{}{
		fieldPrefix + "code", e.Code,
		fieldPrefix + "message", e.Message,
	}
}
