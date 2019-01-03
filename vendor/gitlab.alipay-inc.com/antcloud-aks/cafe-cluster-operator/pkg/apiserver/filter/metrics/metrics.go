package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
)

var (
	bucketQuotaGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_bucket_used_quota",
			Help: "Gauge of used quota.",
		},
		[]string{"type", "bucket_name", "priority_band"},
	)
	metrics = []prometheus.Collector{
		bucketQuotaGauge,
	}
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	registerMetrics.Do(func() {
		for _, metric := range metrics {
			prometheus.MustRegister(metric)
		}
	})
}

func MonitorAcquireReservedQuota(t, bktName string, priority cluster.PriorityBand) {
	bucketQuotaGauge.WithLabelValues(t, bktName, string(priority)).Inc()
}

func MonitorReleaseReservedQuota(t, bktName string, priority cluster.PriorityBand) {
	bucketQuotaGauge.WithLabelValues(t, bktName, string(priority)).Dec()
}
