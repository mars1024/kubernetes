package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/dnscache"
)

// Register registers metrics for dnscache stats.
func Register(stats dnscache.Stats, labels prometheus.Labels) error {
	if err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "dnscache_connections_total",
		Help:        "Counter of connections created by dns cache dialer",
		ConstLabels: labels,
	}, func() float64 { return float64(stats.Stats().TotalConn) })); err != nil {
		return err
	}
	if err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "dnscache_cache_hit_total",
		Help:        "Counter of cache hit in dns cache",
		ConstLabels: labels,
	}, func() float64 { return float64(stats.Stats().CacheHit) })); err != nil {
		return err
	}
	if err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "dnscache_cache_miss_total",
		Help:        "Counter of cache miss in dns cache",
		ConstLabels: labels,
	}, func() float64 { return float64(stats.Stats().CacheMiss) })); err != nil {
		return err
	}
	if err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "dnscache_dns_query_total",
		Help:        "Counter of real dns queries sended by dns cache dialer",
		ConstLabels: labels,
	}, func() float64 { return float64(stats.Stats().DNSQuery) })); err != nil {
		return err
	}
	if err := prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "dnscache_successful_dns_query_total",
		Help:        "Counter of successful dns queries received by dns cache dialer",
		ConstLabels: labels,
	}, func() float64 { return float64(stats.Stats().SuccessfulDNSQuery) })); err != nil {
		return err
	}
	return nil
}

// MustRegister registers metrics for dnscache stats or panic if any error occurs.
func MustRegister(stats dnscache.Stats, labels prometheus.Labels) {
	if err := Register(stats, labels); err != nil {
		panic(err)
	}
}
