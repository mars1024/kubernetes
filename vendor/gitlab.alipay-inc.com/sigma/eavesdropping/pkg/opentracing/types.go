package opentracing

import "time"

// Trace describes a trace record with the details of the latest span.
type Trace struct {
	// ID is trace id.
	ID string `json:"id"`
	// Service is the service name of a trace.
	Service string `json:"service"`
	// CreationTimestamp is the first time this trace record is created.
	// This field should not be changed when updating the trace record.
	CreationTimestamp time.Time `json:"creationTimestamp"`
	// CompletionTimestamp is the final time when this trace record finishes without error.
	// This filed can be changed if the when updating the trace record.
	CompletionTimestamp time.Time `json:"completionTimestamp"`
	// ExecutionCount records the execution count before the trace record succeeds.
	ExecutionCount int64 `json:"executionCount"`
	// Span is the latest span of this trace.
	Span *Span `json:"span"`
}

// Span records the call stack of a trace.
type Span struct {
	// Opeartion is the opeartion name. This should be unique in a service.
	Opeartion string `json:"operation"`
	// Success indicates if the procedure of this span finishes without error.
	Success bool `json:"success"`
	// StartTimestamp is the start timestamp of this span.
	StartTimestamp time.Time `json:"startTimestamp"`
	// EndTimestamp is the end timestamp of this span.
	EndTimestamp time.Time `json:"endTimestamp"`
	// Tags contains key-value pairs of this span.
	Tags map[string]string `json:"tags,omitempty"`
	// Logs contains all logs of this span.
	Logs []Log `json:"logs,omitempty"`
	// Children contains all subspans derived from this span.
	Children []*Span `json:"children,omitempty"`
}

// Log contains an item of a log.
type Log struct {
	Time   time.Time `json:"time"`
	Fields []Field   `json:"fields"`
}

// Field is a  log field.
type Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
