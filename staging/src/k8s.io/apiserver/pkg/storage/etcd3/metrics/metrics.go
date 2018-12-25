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
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		prometheus.MustRegister(etcdRequestLatenciesSummary)
	})
}

func RecordEtcdRequestLatency(verb, resource string, startTime time.Time) {
	etcdRequestLatenciesSummary.WithLabelValues(verb, resource).Observe(float64(time.Since(startTime) / time.Microsecond))
}

func Reset() {
	etcdRequestLatenciesSummary.Reset()
}
