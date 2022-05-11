package broker

import "runtime"

type Traceable interface {
	trace() error
	GetTrace() []TraceInfo
}

type TraceInfo struct {
	File string
	Line int
}

type traceableError struct {
	errorRequest
	Traces []TraceInfo `json:"traces"`
}

func (t *traceableError) trace() error {
	if _, file, line, ok := runtime.Caller(2); ok {
		if t.Traces == nil {
			t.Traces = make([]TraceInfo, 0)
		}
		t.Traces = append(t.Traces, TraceInfo{
			File: file,
			Line: line,
		})
	}
	return t
}

func (t *traceableError) GetTrace() []TraceInfo {
	return t.Traces
}

func errToTraceable(err error) *traceableError {
	var valid bool
	var brokerErr *errorRequest
	if err != nil {
		brokerErr, valid = err.(*errorRequest)
	}

	if !valid {
		if brokerErr, valid = InternalError(err).(*errorRequest); !valid {
			panic("invalid error")
		}
	}

	return &traceableError{
		errorRequest: *brokerErr,
	}
}

func Trace(err error) error {
	var valid bool
	var t Traceable
	t, valid = err.(Traceable)
	if !valid {
		t = errToTraceable(err)
	}
	return t.trace()
}
