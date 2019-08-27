package tracing

import "gitlab.alipay-inc.com/sigma/eavesdropping/pkg/opentracing"

const (
	// Service is the tracing service name.
	Service = "kubelet"
)

var (
	// Tracer is a tracer for opentracing.
	Tracer = opentracing.NewTracer(Service)
	// FlatTracker is a helper for tracing.
	// Default to silent that we don't need to handle tracing errors.
	FlatTracker = opentracing.NewFlatTracker(Tracer, true)
)
