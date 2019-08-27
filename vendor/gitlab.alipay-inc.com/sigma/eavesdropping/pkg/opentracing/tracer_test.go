package opentracing

import (
	"fmt"
	"os"
	"testing"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	os.Setenv("TZ", "GMT")
	os.Exit(m.Run())
}

type mockStruct struct {
	metav1.Object
	annotations map[string]string
}

func newMockStruct(traceID string) *mockStruct {
	return &mockStruct{
		annotations: map[string]string{
			sigmak8sapi.AnnotationKeyTraceID: traceID,
		},
	}
}

func (m *mockStruct) GetAnnotations() map[string]string {
	return m.annotations
}

func (m *mockStruct) SetAnnotations(annotations map[string]string) {
	m.annotations = annotations
}

func TestTracerWithSuccessfulSpan(t *testing.T) {
	obj := newMockStruct("1234")
	tracer := NewTracer("my-svc")
	sc, err := tracer.Extract(MetaObject, obj)
	if err != nil {
		t.Fatal(err)
	}

	startUnix := int64(1000000000)
	getTime := func() time.Time {
		t := time.Unix(startUnix, 0)
		startUnix++
		return t
	}

	span := tracer.StartSpan("o1", opentracing.ChildOf(sc), opentracing.StartTime(getTime()))
	span.SetTag("t1", "2222")
	span.SetTag("t2", "3333")
	span.SetTag("t3", "4444")

	if yy := span.SetBaggageItem("kk", "yy").BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if span.Tracer() != tracer {
		t.Logf("Got wrong tracer: %v", span.Tracer())
	}

	o2 := tracer.StartSpan("o2", opentracing.ChildOf(span.Context()), opentracing.StartTime(getTime()))
	if yy := o2.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if yy := o2.SetBaggageItem("o2", "o2").BaggageItem("o2"); yy != "o2" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}

	o22 := o2.Tracer().StartSpan("o2-2", opentracing.ChildOf(o2.Context()), opentracing.StartTime(getTime()))

	o22.SetTag("o3", "33")

	if yy := o22.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if yy := o22.BaggageItem("o2"); yy != "o2" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}

	o22.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
	})

	o2.SetTag("o22", "222")

	o2.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
	})

	o3 := tracer.StartSpan("o3", opentracing.ChildOf(span.Context()), opentracing.StartTime(getTime()))
	if yy := o3.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}
	o3.SetTag("o22", "222")
	o3.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
	})

	if yy := span.BaggageItem("o2"); yy != "" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}

	if yy := span.BaggageItem("o3"); yy != "" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}
	span.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
		LogRecords: []opentracing.LogRecord{
			{
				Timestamp: getTime(),
				Fields: []log.Field{
					log.String("log1", "log1"),
					log.String("log2", "log2"),
				},
			},
			{
				Timestamp: getTime(),
				Fields: []log.Field{
					log.String("log4", "log4"),
					log.String("log2", "log2"),
				},
			},
		},
	})

	err = tracer.Inject(sc, MetaObject, obj)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"id":"1234","service":"my-svc","creationTimestamp":"2001-09-09T01:46:40Z","completionTimestamp":"2001-09-09T01:46:47Z","executionCount":1,"span":{"operation":"o1","success":true,"startTimestamp":"2001-09-09T01:46:40Z","endTimestamp":"2001-09-09T01:46:47Z","tags":{"t1":"2222","t2":"3333","t3":"4444"},"logs":[{"time":"2001-09-09T01:46:48Z","fields":[{"key":"log1","value":"log1"},{"key":"log2","value":"log2"}]},{"time":"2001-09-09T01:46:49Z","fields":[{"key":"log4","value":"log4"},{"key":"log2","value":"log2"}]}],"children":[{"operation":"o2","success":true,"startTimestamp":"2001-09-09T01:46:41Z","endTimestamp":"2001-09-09T01:46:44Z","tags":{"o22":"222"},"children":[{"operation":"o2-2","success":true,"startTimestamp":"2001-09-09T01:46:42Z","endTimestamp":"2001-09-09T01:46:43Z","tags":{"o3":"33"}}]},{"operation":"o3","success":true,"startTimestamp":"2001-09-09T01:46:45Z","endTimestamp":"2001-09-09T01:46:46Z","tags":{"o22":"222"}}]}}`
	if obj.annotations["pod.beta1.sigma.ali/trace-my-svc"] != expected {
		t.Fatalf("Expected %s but got %s", expected, obj.annotations["pod.beta1.sigma.ali/trace-my-svc"])
	}
}

func TestTracerWithFailedSpan(t *testing.T) {
	obj := newMockStruct("1234")
	tracer := NewTracer("my-svc")
	sc, err := tracer.Extract(MetaObject, obj)
	if err != nil {
		t.Fatal(err)
	}

	startUnix := int64(1000000000)
	getTime := func() time.Time {
		t := time.Unix(startUnix, 0)
		startUnix++
		return t
	}

	span := tracer.StartSpan("o1", opentracing.ChildOf(sc), opentracing.StartTime(getTime()))
	span.SetTag("t1", "2222")
	span.SetTag("t2", "3333")
	span.SetTag("t3", "4444")

	if yy := span.SetBaggageItem("kk", "yy").BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if span.Tracer() != tracer {
		t.Logf("Got wrong tracer: %v", span.Tracer())
	}

	o2 := tracer.StartSpan("o2", opentracing.ChildOf(span.Context()), opentracing.StartTime(getTime()))
	if yy := o2.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if yy := o2.SetBaggageItem("o2", "o2").BaggageItem("o2"); yy != "o2" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}

	o22 := o2.Tracer().StartSpan("o2-2", opentracing.ChildOf(o2.Context()), opentracing.StartTime(getTime()))

	o22.SetTag("o3", "33")

	if yy := o22.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}

	if yy := o22.BaggageItem("o2"); yy != "o2" {
		t.Logf("Got wrong baggage item %s: %s", "o2", yy)
	}

	o22.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
		LogRecords: []opentracing.LogRecord{
			{
				Timestamp: getTime(),
				Fields: []log.Field{
					log.Object("log1", fmt.Errorf("an error")),
				},
			},
		},
	})

	o2.SetTag("o22", "222")

	o2.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
	})

	o3 := tracer.StartSpan("o3", opentracing.ChildOf(span.Context()), opentracing.StartTime(getTime()))
	if yy := o3.BaggageItem("kk"); yy != "yy" {
		t.Logf("Got wrong baggage item %s: %s", "kk", yy)
	}
	o3.SetTag("o22", "222")
	o3.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
	})

	span.FinishWithOptions(opentracing.FinishOptions{
		FinishTime: getTime(),
		LogRecords: []opentracing.LogRecord{
			{
				Timestamp: getTime(),
				Fields: []log.Field{
					log.String("log1", "log1"),
					log.String("log2", "log2"),
				},
			},
			{
				Timestamp: getTime(),
				Fields: []log.Field{
					log.String("log4", "log4"),
					log.String("log2", "log2"),
				},
			},
		},
	})

	err = tracer.Inject(sc, MetaObject, obj)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"id":"1234","service":"my-svc","creationTimestamp":"2001-09-09T01:46:40Z","completionTimestamp":"0001-01-01T00:00:00Z","executionCount":1,"span":{"operation":"o1","success":false,"startTimestamp":"2001-09-09T01:46:40Z","endTimestamp":"2001-09-09T01:46:48Z","tags":{"t1":"2222","t2":"3333","t3":"4444"},"logs":[{"time":"2001-09-09T01:46:49Z","fields":[{"key":"log1","value":"log1"},{"key":"log2","value":"log2"}]},{"time":"2001-09-09T01:46:50Z","fields":[{"key":"log4","value":"log4"},{"key":"log2","value":"log2"}]}],"children":[{"operation":"o2","success":false,"startTimestamp":"2001-09-09T01:46:41Z","endTimestamp":"2001-09-09T01:46:45Z","tags":{"o22":"222"},"children":[{"operation":"o2-2","success":false,"startTimestamp":"2001-09-09T01:46:42Z","endTimestamp":"2001-09-09T01:46:43Z","tags":{"o3":"33"},"logs":[{"time":"2001-09-09T01:46:44Z","fields":[{"key":"log1","value":"an error"}]}]}]},{"operation":"o3","success":true,"startTimestamp":"2001-09-09T01:46:46Z","endTimestamp":"2001-09-09T01:46:47Z","tags":{"o22":"222"}}]}}`
	if obj.annotations["pod.beta1.sigma.ali/trace-my-svc"] != expected {
		t.Fatalf("Expected %s but got %s", expected, obj.annotations["pod.beta1.sigma.ali/trace-my-svc"])
	}
}
