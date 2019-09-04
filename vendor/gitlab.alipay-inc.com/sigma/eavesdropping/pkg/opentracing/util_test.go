package opentracing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

func TestLayerTracker(t *testing.T) {
	tracker := NewLayerTracker(NewTracer("my-svc"), true)

	obj := newMockStruct("124")

	mo := NewMiniObject("my-svc", obj)

	err := tracker.Track(context.Background(), mo, "create", func(ctx context.Context) error {
		return tracker.Track(ctx, nil, "update", func(ctx context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(mo.annotations) != 2 {
		t.Fatalf("MoniObject must have 2 keys but got %d", len(mo.annotations))
	}

	if v := mo.annotations[keyForTraceID()]; v != "124" {
		t.Fatalf("Expected trace id %s but got %s", "124", v)
	}

	v := mo.annotations[keyForService("my-svc")]
	cases := []struct {
		patches  string
		expected string
	}{
		{
			patches:  `{"metadata":{"name":"kkk"}}`,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q},"name":"kkk"}}`, keyForService("my-svc"), v),
		},
		{
			patches:  ``,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q}}}`, keyForService("my-svc"), v),
		},
		{
			patches:  `{}`,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q}}}`, keyForService("my-svc"), v),
		},
		{
			patches:  `{"spec":{"a":"bbb"}}`,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q}},"spec":{"a":"bbb"}}`, keyForService("my-svc"), v),
		},
		{
			patches:  `{"metadata":{"annotations":{"kkk":"nbbb"},"name":"kkk"}}`,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"kkk":"nbbb","pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q},"name":"kkk"}}`, keyForService("my-svc"), v),
		},
		{
			patches:  `{"metadata":{"annotations":{"pod.beta1.sigma.ali/trace-my-svc":"111"},"name":"kkk"}}`,
			expected: fmt.Sprintf(`{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"%s":%q},"name":"kkk"}}`, keyForService("my-svc"), v),
		},
	}

	for _, c := range cases {
		result := mo.StrategicMergePatch(c.patches)
		if result != c.expected {
			t.Fatalf("Expected patches %s but got %s", c.expected, result)
		}
	}

}

func TestLayerTrackerWithError(t *testing.T) {
	tracker := NewLayerTracker(NewTracer("my-svc"), false)

	obj := newMockStruct("")

	err := tracker.Track(context.Background(), obj, "create", func(ctx context.Context) error {
		return tracker.Track(ctx, nil, "update", func(ctx context.Context) error {
			return nil
		})
	})
	if err != opentracing.ErrInvalidCarrier {
		t.Fatal(err)
	}
}

func TestFlatTracker(t *testing.T) {
	tracker := NewFlatTracker(NewTracer("my-svc"), true)

	obj := newMockStruct("124")

	mo := NewMiniObject("my-svc", obj)

	ctx, tk, err := tracker.Track(context.Background(), mo, "test-flat")
	if err != nil {
		t.Fatal(err)
	}
	// stage 0
	ctx, _ = tk.Stage(ctx, "stage0")
	SetTag(ctx, "tag", "value")

	// stage 1
	tk.Stage(ctx, "stage1")

	// Finish
	tk.Finish(ctx)

	_, value := mo.Value()

	t.Log(value)

	trace := &Trace{}

	err = json.Unmarshal([]byte(value), trace)
	if err != nil {
		t.Fatal(err)
	}

	if trace.Span.Operation != "test-flat" {
		t.Fatalf("Expected root operation %s but got %s", "test-flat", trace.Span.Operation)
	}
	if len(trace.Span.Children) != 2 {
		t.Fatalf("Expected %d children but got %d", 2, len(trace.Span.Children))
	}
	if trace.Span.Children[0].Operation != "stage0" {
		t.Fatalf("Expected stage operation %s but got %s", "stage0", trace.Span.Children[0].Operation)
	}
	if trace.Span.Children[0].Tags["tag"] != "value" {
		t.Fatalf("Expected stage tag value %s but got %s", "value", trace.Span.Children[0].Tags["tag"])
	}
	if trace.Span.Children[1].Operation != "stage1" {
		t.Fatalf("Expected stage operation %s but got %s", "stage0", trace.Span.Children[1].Operation)
	}
}

func TestMiniObject(t *testing.T) {
	data := `{"id":"124","service":"my-svc","creationTimestamp":"2019-08-26T06:50:53.342147163Z","completionTimestamp":"2019-08-26T06:50:53.342158379Z","executionCount":1,"span":{"operation":"test-flat","success":true,"startTimestamp":"2019-08-26T06:50:53.342147163Z","endTimestamp":"2019-08-26T06:50:53.342158379Z","children":[{"operation":"stage0","success":true,"startTimestamp":"2019-08-26T06:50:53.342155209Z","endTimestamp":"2019-08-26T06:50:53.342157238Z","tags":{"tag":"value"}},{"operation":"stage1","success":true,"startTimestamp":"2019-08-26T06:50:53.342158097Z","endTimestamp":"2019-08-26T06:50:53.342158266Z"}]}}`
	expected := `{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":null,"pod.beta1.sigma.ali/trace-my-svc":"{\"id\":\"124\",\"service\":\"my-svc\",\"creationTimestamp\":\"2019-08-26T06:50:53.342147163Z\",\"completionTimestamp\":\"2019-08-26T06:50:53.342158379Z\",\"executionCount\":1,\"span\":{\"operation\":\"test-flat\",\"success\":true,\"startTimestamp\":\"2019-08-26T06:50:53.342147163Z\",\"endTimestamp\":\"2019-08-26T06:50:53.342158379Z\",\"children\":[{\"operation\":\"stage0\",\"success\":true,\"startTimestamp\":\"2019-08-26T06:50:53.342155209Z\",\"endTimestamp\":\"2019-08-26T06:50:53.342157238Z\",\"tags\":{\"tag\":\"value\"}},{\"operation\":\"stage1\",\"success\":true,\"startTimestamp\":\"2019-08-26T06:50:53.342158097Z\",\"endTimestamp\":\"2019-08-26T06:50:53.342158266Z\"}]}}"}}}`
	compressed := `{"metadata":{"annotations":{"pod.beta1.sigma.ali/gzip-trace-my-svc":"H4sIAAAAAAAA/6TQu2r0MBDF8Xc5tf0hyfdpv1fYakMKIU8cg29oxiZh8bsHpQukiNn+/OHHeWDsQbCuRAbheIyBQZg/czkCMoTIXsd1uY0zi/p5A8EZ2+WmzV19MzVVhqriX1E6Wza2Lu4pWudt4r9mVVs0Xcr4g8Oeqv/rvijIZpDNL6AH1o3jtwMEZdH8bfKaxHsILALSuHMGUR/1EpWX/pIxvI9TH3kBvfxUifqBzROkqnKmu0JqXNGmvfpB0kfqBxAOP+2M88x+4dlneK3pmkuPubq+43w9z68AAAD//wY56MJkAgAA","pod.beta1.sigma.ali/trace-my-svc":null}}}`
	obj := MiniObject{
		service: "my-svc",
		annotations: map[string]string{
			keyForService("my-svc"): data,
		},
	}
	result := obj.StrategicMergePatch("{}")
	if result != expected {
		t.Fatalf("Expected data %s but got %s", expected, result)
	}

	result = obj.CompressedStrategicMergePatch("{}")
	t.Log(result)
	if result != compressed {
		t.Fatalf("Expected compressed data %s but got %s", compressed, result)
	}
}

func TestMiniObjectWithCompressData(t *testing.T) {
	origin := &mockStruct{
		annotations: map[string]string{
			sigmak8sapi.AnnotationKeyTraceID:        "124",
			"pod.beta1.sigma.ali/gzip-trace-my-svc": "H4sIAAAAAAAA/6TQu2r0MBDF8Xc5tf0hyfdpv1fYakMKIU8cg29oxiZh8bsHpQukiNn+/OHHeWDsQbCuRAbheIyBQZg/czkCMoTIXsd1uY0zi/p5A8EZ2+WmzV19MzVVhqriX1E6Wza2Lu4pWudt4r9mVVs0Xcr4g8Oeqv/rvijIZpDNL6AH1o3jtwMEZdH8bfKaxHsILALSuHMGUR/1EpWX/pIxvI9TH3kBvfxUifqBzROkqnKmu0JqXNGmvfpB0kfqBxAOP+2M88x+4dlneK3pmkuPubq+43w9z68AAAD//wY56MJkAgAA",
		},
	}

	mo := NewMiniObject("my-svc", origin)

	if len(mo.annotations) != 2 {
		t.Fatalf("MoniObject must have 2 keys but got %d", len(mo.annotations))
	}

	if v := mo.annotations[keyForTraceID()]; v != "124" {
		t.Fatalf("Expected trace id %s but got %s", "124", v)
	}

	expected := `{"id":"124","service":"my-svc","creationTimestamp":"2019-08-26T06:50:53.342147163Z","completionTimestamp":"2019-08-26T06:50:53.342158379Z","executionCount":1,"span":{"operation":"test-flat","success":true,"startTimestamp":"2019-08-26T06:50:53.342147163Z","endTimestamp":"2019-08-26T06:50:53.342158379Z","children":[{"operation":"stage0","success":true,"startTimestamp":"2019-08-26T06:50:53.342155209Z","endTimestamp":"2019-08-26T06:50:53.342157238Z","tags":{"tag":"value"}},{"operation":"stage1","success":true,"startTimestamp":"2019-08-26T06:50:53.342158097Z","endTimestamp":"2019-08-26T06:50:53.342158266Z"}]}}`

	if v := mo.annotations[keyForService("my-svc")]; v != expected {
		t.Fatalf("Expected trace data %s but got %s", expected, v)
	}
}
