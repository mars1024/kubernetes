package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
)

var (
	usedBucketQuotaGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_bucket_used_quota",
			Help: "Gauge of used quota.",
		},
		[]string{"type", "bucket_name", "priority_band"},
	)
	remainingBucketQuotaGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_bucket_remaining_quota",
			Help: "Gauge of remaining quota.",
		},
		[]string{"type", "bucket_name", "priority_band"},
	)
	metrics = []prometheus.Collector{
		usedBucketQuotaGauge,
		remainingBucketQuotaGauge,
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
	usedBucketQuotaGauge.WithLabelValues(t, bktName, string(priority)).Inc()
}

func MonitorReleaseReservedQuota(t, bktName string, priority cluster.PriorityBand) {
	usedBucketQuotaGauge.WithLabelValues(t, bktName, string(priority)).Dec()
}

func MonitorRemainingReservedQuota(t, bktName string, priority cluster.PriorityBand, remaining int) {
	remainingBucketQuotaGauge.WithLabelValues(t, bktName, string(priority)).Set(float64(remaining))
}
