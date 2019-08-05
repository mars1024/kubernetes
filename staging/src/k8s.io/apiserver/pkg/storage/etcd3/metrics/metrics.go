package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	etcdRequestLatenciesSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "etcd_request_latencies_summary",
			Help: "Etcd request latency summary in microseconds for each operation and object type.",
		},
		[]string{"operation", "type"},
	)
	// This metric is inaccurate because of the swift change of a channel.
	etcdWatcherChannelLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "etcd_watcher_channel_length",
			Help: "Etcd watcher channel length",
		},
		[]string{"key", "channel"},
	)
	etcdEventsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "etcd_watcher_received_events",
			Help: "Counter of events received from etcd broken by key",
		},
		[]string{"key"},
	)
	sendedEventsLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "etcd_watcher_sended_events_latency_milliseconds_bucket",
			Help:    "Latency bucket of watchers sended by key",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource"},
	)
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		prometheus.MustRegister(etcdRequestLatenciesSummary)
		prometheus.MustRegister(etcdWatcherChannelLength)
		prometheus.MustRegister(etcdEventsCounter)
		prometheus.MustRegister(sendedEventsLatency)
	})
}

func RecordEtcdRequestLatency(verb, resource string, startTime time.Time) {
	etcdRequestLatenciesSummary.WithLabelValues(verb, resource).Observe(float64(time.Since(startTime) / time.Microsecond))
}

func RecordEtcdWatcherChannelLength(key, name string, length int) {
	etcdWatcherChannelLength.WithLabelValues(key, name).Set(float64(length))
}

func RecordEtcdWatcherEventCount(key string, count int) {
	etcdEventsCounter.WithLabelValues(key).Add(float64(count))
}

func RecordEtcdWatcherEventLatency(key string, duration time.Duration) {
	sendedEventsLatency.WithLabelValues(key).Observe(float64(duration/time.Microsecond) / 1000)
}

func Reset() {
	etcdRequestLatenciesSummary.Reset()
}
