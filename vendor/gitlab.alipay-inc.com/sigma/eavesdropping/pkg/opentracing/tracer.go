package opentracing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewTracer creates a memory tracer.
func NewTracer(service string) opentracing.Tracer {
	return &tracer{
		service: service,
	}
}

// The tracker only supports carrier with type MetaObject.
type tracer struct {
	service string
}

var _ opentracing.Tracer = &tracer{}

// StartSpan starts a new span. It only supports one parent span context.
func (t *tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	options := &opentracing.StartSpanOptions{}
	for _, opt := range opts {
		opt.Apply(options)
	}

	var parentSC *spanContext
	for _, ref := range options.References {
		// Only support child ref.
		if ref.Type == opentracing.ChildOfRef {
			if sc, ok := ref.ReferencedContext.(*spanContext); ok {
				parentSC = sc
				break
			}
		}
	}

	if parentSC == nil {
		panic("must have a parent span context")
	}

	s := &span{
		tracer:    t,
		operation: operationName,
		tags:      map[string]string{},
		logs:      []Log{},
		children:  []*span{},
	}

	if parentSC.span == nil {
		// The span context has no related span. It's the root context.
		// Make this span as the root span.
		s.ctx = parentSC
	} else {
		s.ctx = &spanContext{
			ctx:     parentSC,
			traceID: parentSC.traceID,
			service: t.service,
			values:  map[string]string{},
		}
		// Append current span to its parent.
		parentSC.span.children = append(parentSC.span.children, s)
	}
	s.ctx.span = s

	if !options.StartTime.IsZero() {
		s.startTime = options.StartTime
	} else {
		s.startTime = time.Now()
	}
	for k, v := range options.Tags {
		s.SetTag(k, v)
	}

	return s
}

// Inject injects data to carrier. It only supports MetaObject.
func (t *tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	sc, ok := sm.(*spanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}

	if sc.span == nil {
		return opentracing.ErrInvalidSpanContext
	}

	f, ok := format.(ExtentionFormat)
	if !ok || f != MetaObject {
		return opentracing.ErrInvalidCarrier
	}

	obj, ok := carrier.(metav1.Object)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	span := sc.span.record()

	trace := sc.lastTrace
	if trace == nil || !trace.CompletionTimestamp.IsZero() {
		trace = &Trace{
			ID:                sc.traceID,
			Service:           sc.service,
			CreationTimestamp: span.StartTimestamp,
			ExecutionCount:    0,
		}
	}
	trace.Span = span
	trace.ExecutionCount++
	if span.Success {
		trace.CompletionTimestamp = span.EndTimestamp
	}

	data, err := json.Marshal(trace)
	if err != nil {
		return opentracing.ErrInvalidSpanContext
	}

	annos := obj.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[keyForService(t.service)] = string(data)
	obj.SetAnnotations(annos)

	return nil
}

// Extract extracts data from a carrier. It only supports MetaObject.
func (t *tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	f, ok := format.(ExtentionFormat)
	if !ok || f != MetaObject {
		return nil, opentracing.ErrInvalidCarrier
	}

	obj, ok := carrier.(metav1.Object)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	traceID := ""
	spans := ""
	if annos := obj.GetAnnotations(); annos != nil {
		traceID = annos[keyForTraceID()]
		spans = annos[keyForService(t.service)]
	}
	if traceID == "" {
		return nil, opentracing.ErrInvalidCarrier
	}

	lastTrace := &Trace{}
	if spans != "" {
		if err := json.Unmarshal([]byte(spans), lastTrace); err != nil {
			return nil, opentracing.ErrInvalidCarrier
		}
	}
	sc := &spanContext{
		traceID: traceID,
		service: t.service,
		values:  map[string]string{},
	}
	if lastTrace.ID == traceID {
		sc.lastTrace = lastTrace
	}
	return sc, nil
}

type spanContext struct {
	ctx       *spanContext
	traceID   string
	service   string
	values    map[string]string
	lastTrace *Trace
	span      *span
}

var _ opentracing.SpanContext = &spanContext{}

func (b *spanContext) ForeachBaggageItem(fn func(k, v string) bool) {
	for k, v := range b.values {
		if ctn := fn(k, v); !ctn {
			return
		}
	}
	if b.ctx != nil {
		b.ctx.ForeachBaggageItem(fn)
	}
}

type logItem struct {
	time  time.Time
	key   string
	value string
}

type span struct {
	lock       sync.RWMutex
	tracer     *tracer
	ctx        *spanContext
	operation  string
	tags       map[string]string
	errors     int
	logs       []Log
	children   []*span
	startTime  time.Time
	finishTime time.Time
}

var _ opentracing.Span = &span{}

func (s *span) Context() opentracing.SpanContext {
	return s.ctx
}

func (s *span) SetBaggageItem(key, val string) opentracing.Span {
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.ctx.values[key] = val
	return s
}

func (s *span) BaggageItem(key string) string {
	s.lock.Lock()
	defer s.lock.Unlock()
	val := ""
	s.ctx.ForeachBaggageItem(func(k, v string) bool {
		if k == key {
			val = v
			return false
		}
		return true
	})
	return val
}

func (s *span) marshal(obj interface{}) string {
	switch o := obj.(type) {
	case error:
		return o.Error()
	}
	typ := reflect.TypeOf(obj)
	switch typ.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return fmt.Sprint(obj)
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprint(obj)
	}
	return string(data)
}

func (s *span) SetTag(key string, value interface{}) opentracing.Span {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.tags[key] = s.marshal(value)
	return s
}

func (s *span) log(t time.Time, fields ...log.Field) {
	item := Log{
		Time: t,
	}
	errorCount := 0
	for _, field := range fields {
		value := field.Value()
		if _, ok := value.(error); ok {
			errorCount++
		}
		item.Fields = append(item.Fields, Field{
			Key:   field.Key(),
			Value: s.marshal(value),
		})
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.errors += errorCount
	s.logs = append(s.logs, item)
}

func (s *span) LogFields(fields ...log.Field) {
	s.log(time.Now(), fields...)
}

func (s *span) LogKV(keyVals ...interface{}) {
	if len(keyVals) <= 0 {
		return
	}
	if len(keyVals)%2 != 0 {
		panic("parameters of LogKV must be in pairs")
	}
	fields := []log.Field{}
	for i := 0; i < len(keyVals); i += 2 {
		key, ok := keyVals[i].(string)
		if !ok {
			panic("key of LogKV must be string")
		}
		value := keyVals[i+1]
		fields = append(fields, log.Object(key, value))
	}
	s.LogFields(fields...)
}

func (s *span) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s *span) FinishWithOptions(opts opentracing.FinishOptions) {
	s.lock.Lock()
	s.finishTime = opts.FinishTime
	if opts.FinishTime.IsZero() {
		s.finishTime = time.Now()
	}
	s.lock.Unlock()

	for _, record := range opts.LogRecords {
		if len(record.Fields) > 0 {
			s.log(record.Timestamp, record.Fields...)
		}
	}
}

func (s *span) SetOperationName(operationName string) opentracing.Span {
	s.operation = operationName
	return s
}

func (s *span) Tracer() opentracing.Tracer {
	return s.tracer
}

// Deprecated: use LogFields or LogKV
func (s *span) LogEvent(event string) {
	panic("LogEvent: Not Implement")
}

// Deprecated: use LogFields or LogKV
func (s *span) LogEventWithPayload(event string, payload interface{}) {
	panic("LogEventWithPayload: Not Implement")
}

// Deprecated: use LogFields or LogKV
func (s *span) Log(data opentracing.LogData) {
	panic("Log: Not Implement")
}

func (s *span) record() *Span {
	s.lock.RLock()
	defer s.lock.RUnlock()
	span := &Span{
		Operation:      s.operation,
		Success:        s.errors == 0,
		StartTimestamp: s.startTime,
		EndTimestamp:   s.finishTime,
		Tags:           s.tags,
		Logs:           s.logs,
	}
	for _, sp := range s.children {
		r := sp.record()
		if !r.Success {
			span.Success = false
		}
		span.Children = append(span.Children, r)
	}
	return span
}
