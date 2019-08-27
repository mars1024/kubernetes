package opentracing

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LayerTracker creates a layer tracker to record child spans.
//
// Usage:
//
//  const service = "my-component"
//  tracker := NewLayerTracker(NewTracer(service), true)
//
//  func DoSomething() error {
//      mo := NewMiniObject(service, obj)
//      tracker.Track(context.Background(), mo, "DoSomething", func(ctx context.Context) error {
//          // Do something.
//          AnotherFunc(ctx)
//          // Return
//      })
//      // Patch obj via the methods of MiniObject.
//      // Return
//  }
//
//  func AnotherFunc(ctx) error {
//      return tracker.Track(ctx, nil, "AnotherFunc", func(ctx context.Context) error {
//      	return nil
//      })
//  }
//
//
//  Result:
//    DoSomething [Root]
//    |- AnotherFunc
type LayerTracker struct {
	tracer opentracing.Tracer
	silent bool
}

// NewLayerTracker creates a layer tracker.
func NewLayerTracker(tracer opentracing.Tracer, silent bool) *LayerTracker {
	return &LayerTracker{
		tracer: tracer,
		silent: silent,
	}
}

// Tracer returns the tracer in this tracker.
func (t *LayerTracker) Tracer() opentracing.Tracer {
	return t.tracer
}

// Track tracks the fn and injects trace data to the obj if this is root span.
// If the obj is nil, that means it won't create a root span and only inherit parent span from the ctx.
// If the tracker is silent, all tracing error are ignored and the returned error is from fn.
func (t *LayerTracker) Track(ctx context.Context, obj metav1.Object, operation string, fn func(ctx context.Context) error, opts ...opentracing.StartSpanOption) error {
	parentContext, isRoot, err := extractParentSpanContext(ctx, t.tracer, obj)
	if !t.silent && err != nil {
		return err
	}
	if parentContext == nil {
		// If there is no parent context, call fn directly.
		return fn(ctx)
	}

	// Build span.
	opts = append(opts, opentracing.ChildOf(parentContext))
	span := t.tracer.StartSpan(operation, opts...)
	ctx = opentracing.ContextWithSpan(ctx, span)

	resutlErr := fn(ctx)
	if resutlErr != nil {
		// Record the error to the span.
		LogError(ctx, resutlErr)
	}

	span.Finish()
	if isRoot {
		// If current span is root, inject tracing data to the object.
		if err := t.tracer.Inject(parentContext, MetaObject, obj); !t.silent && err != nil {
			return err
		}
	}
	return resutlErr
}

// FlatTracker creates a flat tracker to record sibling spans.
//
// Usage:
//
//  const service = "my-component"
//  tracker := NewFlatTracker(NewTracer(service), true)
//
//  func DoSomething() error {
//      mo := NewMiniObject(service, obj)
//      ctx, t, err := tracker.Track(context.Background(), mo, "DoSomething")
//
//      t.Stage(ctx, "stage0")
//      // Do something for stage 0.
//
//      ctx, _ = t.Stage(ctx, "stage1")
//      AnotherFunc(ctx)
//
//      t.Stage(ctx, "stage2")
//      // Do something for stage 2.
//
//      // Patch obj via the methods of MiniObject.
//      // Return
//  }
//
//  func AnotherFunc(ctx) error {
//      ctx, t, err := tracker.Track(ctx, nil, "AnotherFunc")
//
//      t.Stage(ctx, "stageX")
//      // Do something for stage x.
//  }
//
//
//  Result:
//    DoSomething [Root]
//    |- stage0
//    |- stage1
//      |- AnotherFunc
//        |- stageX
//    |- stage2
type FlatTracker struct {
	tracer opentracing.Tracer
	silent bool
}

// NewFlatTracker creates a flat tracker.
func NewFlatTracker(tracer opentracing.Tracer, silent bool) *FlatTracker {
	return &FlatTracker{
		tracer: tracer,
		silent: silent,
	}
}

// Track tracks the fn and injects trace data to the obj if this is root span.
// If the tracker is silent, you can safely ignore the error returned from this method.
func (t *FlatTracker) Track(ctx context.Context, obj metav1.Object, operation string, opts ...opentracing.StartSpanOption) (context.Context, *DisposableFlatTracker, error) {
	parentContext, isRoot, err := extractParentSpanContext(ctx, t.tracer, obj)
	if !t.silent && err != nil {
		return ctx, nil, err
	}

	var span opentracing.Span
	if parentContext != nil {
		opts = append(opts, opentracing.ChildOf(parentContext))
		span = t.tracer.StartSpan(operation, opts...)
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	return ctx, &DisposableFlatTracker{
		tracker:  t,
		parent:   span,
		object:   obj,
		isRoot:   isRoot,
		finished: false,
	}, nil
}

// DisposableFlatTracker returns a flat tracker for recording a trace.
// Don't reuse this tracker. Don't use this tracker in parallel. It's not thread-safe.
type DisposableFlatTracker struct {
	tracker  *FlatTracker
	isRoot   bool
	parent   opentracing.Span
	object   metav1.Object
	lastSpan opentracing.Span
	finished bool
}

// Stage starts a new span. This will also finish last span if it's existing.
// If the tracker is silent, you can safely ignore the error returned from this method.
func (t *DisposableFlatTracker) Stage(ctx context.Context, operation string, opts ...opentracing.StartSpanOption) (context.Context, error) {
	if t.parent == nil {
		return ctx, opentracing.ErrInvalidSpanContext
	}
	err := t.finish(ctx)
	if err != nil && !t.tracker.silent {
		// Return the error if it's not silent.
		return ctx, err
	}
	opts = append(opts, opentracing.ChildOf(t.parent.Context()))
	t.lastSpan = t.tracker.tracer.StartSpan(operation, opts...)
	return opentracing.ContextWithSpan(ctx, t.lastSpan), nil
}

func (t *DisposableFlatTracker) finish(ctx context.Context) error {
	if t.lastSpan != nil {
		t.lastSpan.Finish()
	}
	return nil
}

// Finish finishes the last span. This method can be called many times but
// only the first time is valid.
// If the tracker is silent, you can safely ignore the error returned from this method.
func (t *DisposableFlatTracker) Finish(ctx context.Context) error {
	if t.finished {
		return nil
	}
	t.finish(ctx)
	if t.parent != nil {
		t.parent.Finish()
		if t.isRoot {
			if err := t.tracker.tracer.Inject(t.parent.Context(), MetaObject, t.object); !t.tracker.silent && err != nil {
				return err
			}
		}
	}

	t.finished = true
	return nil
}

// extractParentSpanContext extracts parentContext from the ctx or the obj.
//
// There are only two cases that the parentContext is non-nil:
// 1. ctx has a parent span.
// 2. obj is non-nil and carries tracing data.
func extractParentSpanContext(ctx context.Context, tracer opentracing.Tracer, obj metav1.Object) (parentContext opentracing.SpanContext, isRoot bool, err error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		parentContext = span.Context()
	} else {
		if obj == nil {
			err = opentracing.ErrInvalidCarrier
		} else {
			// Extract context from the object.
			parentContext, err = tracer.Extract(MetaObject, obj)
			if parentContext != nil {
				isRoot = true
			}
		}
	}
	return
}

// MiniObject implements metav1.Object and saving data from tracer.
type MiniObject struct {
	metav1.Object
	service     string
	annotations map[string]string
}

// NewMiniObject creates a mini object for obj.
func NewMiniObject(service string, obj metav1.Object) *MiniObject {
	mo := &MiniObject{
		service:     service,
		annotations: map[string]string{},
	}
	annos := obj.GetAnnotations()
	keys := []string{
		keyForTraceID(),
		keyForService(service),
	}
	for _, key := range keys {
		if v, ok := annos[key]; ok {
			mo.annotations[key] = v
		}
	}

	key := keyForService(service)
	if mo.annotations[key] == "" {
		// Try to parse compressed trace data.
		compressedValue := annos[compressedKeyForService(service)]
		if compressedValue != "" {
			data, err := uncompress(compressedValue)
			if err == nil && json.Valid(data) {
				mo.annotations[key] = string(data)
			}
		}
	}
	return mo
}

// GetAnnotations returns annotations from this object.
func (m *MiniObject) GetAnnotations() map[string]string {
	return m.annotations
}

// SetAnnotations sets annotations to this object.
func (m *MiniObject) SetAnnotations(annotations map[string]string) {
	m.annotations = annotations
}

// ApplyTo applies the changes to an object.
func (m *MiniObject) ApplyTo(obj metav1.Object) {
	annos := obj.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	for k, v := range m.annotations {
		annos[k] = v
	}
	obj.SetAnnotations(annos)
}

// Value returns the value of the trace in this object.
func (m *MiniObject) Value() (key string, value string) {
	key = keyForService(m.service)
	return key, m.annotations[key]
}

// merge merges values to patches.metadata.annotations.
func (m *MiniObject) merge(patches string, values map[string]interface{}) string {
	if patches == "" {
		patches = "{}"
	}

	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(patches), &obj)
	if err != nil {
		// For normal case, it should not be here.
		panic(fmt.Sprintf("unmarshal patches %s with error %v", patches, err))
	}
	metadata, ok := obj["metadata"]
	if !ok {
		metadata = map[string]interface{}{}
	}
	md := metadata.(map[string]interface{})
	annotations, ok := md["annotations"]
	if !ok {
		annotations = map[string]interface{}{}
	}
	annos := annotations.(map[string]interface{})
	for key, value := range values {
		annos[key] = value
	}
	md["annotations"] = annotations
	obj["metadata"] = metadata
	data, err := json.Marshal(obj)
	if err != nil {
		// For normal case, it should not be here.
		panic(fmt.Sprintf("marshal patches %+v with error %v", obj, err))
	}
	return string(data)
}

// MergePatch merges the changes to patches which type is MergePatch.
func (m *MiniObject) MergePatch(patches string) string {
	key := keyForService(m.service)
	if len(m.annotations) <= 0 || m.annotations[key] == "" {
		return patches
	}
	return m.merge(patches, map[string]interface{}{
		key:                                m.annotations[key],
		compressedKeyForService(m.service): nil,
	})
}

// StrategicMergePatch merges the changes to patches which type is StrategicMergePatch.
func (m *MiniObject) StrategicMergePatch(patches string) string {
	return m.MergePatch(patches)
}

// CompressedMergePatch merges the compressed changes to patches which type is MergePatch.
// The method will back off to MergePatch if an error occurs.
func (m *MiniObject) CompressedMergePatch(patches string) string {
	key := keyForService(m.service)
	if len(m.annotations) <= 0 || m.annotations[key] == "" {
		return patches
	}

	value, err := compress([]byte(m.annotations[key]))
	if err != nil {
		// Back off to uncompressed version.
		return m.MergePatch(patches)
	}

	return m.merge(patches, map[string]interface{}{
		compressedKeyForService(m.service): value,
		key:                                nil,
	})
}

func compress(data []byte) (string, error) {
	buf := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buf)
	_, err := w.Write(data)
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func uncompress(data string) ([]byte, error) {
	gzipData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(gzipData))
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return result, err
}

// CompressedStrategicMergePatch merges the compressed changes to patches which type is StrategicMergePatch.
func (m *MiniObject) CompressedStrategicMergePatch(patches string) string {
	return m.CompressedMergePatch(patches)
}

// SetBaggageItem sets a baggage item to the span from a context.
// If there is no span in context, this call does nothing.
func SetBaggageItem(ctx context.Context, key string, value string) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.SetBaggageItem(key, value)
	}
}

// BaggageItem gets a baggage item form the span from a context.
// If there is no span in context, this call does nothing.
func BaggageItem(ctx context.Context, key string, value interface{}) string {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		return span.BaggageItem(key)
	}
	return ""
}

// SetTag sets a tag to the span from a context.
// If there is no span in context, this call does nothing.
func SetTag(ctx context.Context, key string, value interface{}) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.SetTag(key, value)
	}
}

// LogFields logs to a span from context.
// If there is no span in context, this call does nothing.
func LogFields(ctx context.Context, fields ...log.Field) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.LogFields(fields...)
	}
}

// LogKV logs to a span from context.
// If there is no span in context, this call does nothing.
func LogKV(ctx context.Context, keyVals ...interface{}) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.LogKV(keyVals...)
	}
}

// LogError logs an error to the context. This function records
// a log with key 'error'. If the err is nil, record nothing.
// If there is no span in context, this call does nothing.
func LogError(ctx context.Context, err error) {
	if span := opentracing.SpanFromContext(ctx); span != nil && err != nil {
		span.LogKV("error", err)
	}
}
