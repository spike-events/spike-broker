package broker

import "runtime"

type Traceable interface {
	trace(stackPos int) error
	GetTrace() []TraceInfo
}

type TraceInfo struct {
	File string
	Line int
}

type traceableError struct {
	errorMessage
	Traces []TraceInfo `json:"traces"`
}

func (t *traceableError) trace(stackPos int) error {
	if _, file, line, ok := runtime.Caller(2 + stackPos); ok {
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
	var brokerErr *errorMessage
	if err != nil {
		brokerErr, valid = err.(*errorMessage)
	}

	if !valid {
		if brokerErr, valid = InternalError(err).(*errorMessage); !valid {
			panic("invalid error")
		}
	}

	return &traceableError{
		errorMessage: *brokerErr,
	}
}

func Trace(err error, stackPosition int) error {
	var valid bool
	var t Traceable
	t, valid = err.(Traceable)
	if !valid {
		t = errToTraceable(err)
	}
	return t.trace(stackPosition)
}
