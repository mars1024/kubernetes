/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

// AllMetricGroups is MetricGroup set
var AllMetricGroups = []MetricGroup{
	ContainerCPUUsageLimitGroup,
	ContainerCPUUsageRequestGroup,
	ContainerCPULoadAverageGroup,
	ContainerMemoryMetricsGroup,
	NodeCPUUsageMetricsGroup,
	NodeSystemLoadMetricsGroup,
	NodeMemoryMetricsGroup,
}

// ContainerCPUUsageLimitGroup represents container cpu usage limit metrics
var ContainerCPUUsageLimitGroup = MetricGroup{
	Name:    "container_cpu_usage_limit",
	Help:    "represents container cpu usage limit metrics",
	Type:    ContainerMetricType,
	Scraper: "walle",
	Labels: []string{
		"id",
		"pod",
		"namespace",
		"image",
	},
	Metrics: []Metric{
		ContainerCPUUsageLimit,
		ContainerCPUUsageLimitMaxOver1min,
		ContainerCPUUsageLimitMinOver1min,
		ContainerCPUUsageLimitAvgOver1min,
		ContainerCPUUsageLimitP99Over1min,
		ContainerCPUUsageLimitPredict1min,
		ContainerCPUUsageLimitMaxOver5min,
		ContainerCPUUsageLimitMinOver5min,
		ContainerCPUUsageLimitAvgOver5min,
		ContainerCPUUsageLimitP99Over5min,
		ContainerCPUUsageLimitPredict5min,
		ContainerCPUUsageLimitMaxOver15min,
		ContainerCPUUsageLimitMinOver15min,
		ContainerCPUUsageLimitAvgOver15min,
		ContainerCPUUsageLimitP99Over15min,
		ContainerCPUUsageLimitPredict15min,
	},
}

// ContainerCPUUsageRequestGroup represents container cpu usage request metrics
var ContainerCPUUsageRequestGroup = MetricGroup{
	Name:    "container_cpu_usage_request",
	Help:    "represents container cpu usage request metrics",
	Type:    ContainerMetricType,
	Scraper: "walle",
	Labels: []string{
		"id",
		"pod",
		"namespace",
		"image",
	},
	Metrics: []Metric{
		ContainerCPUUsageRequest,
		ContainerCPUUsageRequestMaxOver1min,
		ContainerCPUUsageRequestMinOver1min,
		ContainerCPUUsageRequestAvgOver1min,
		ContainerCPUUsageRequestP99Over1min,
		ContainerCPUUsageRequestPredict1min,
		ContainerCPUUsageRequestMaxOver5min,
		ContainerCPUUsageRequestMinOver5min,
		ContainerCPUUsageRequestAvgOver5min,
		ContainerCPUUsageRequestP99Over5min,
		ContainerCPUUsageRequestPredict5min,
		ContainerCPUUsageRequestMaxOver15min,
		ContainerCPUUsageRequestMinOver15min,
		ContainerCPUUsageRequestAvgOver15min,
		ContainerCPUUsageRequestP99Over15min,
		ContainerCPUUsageRequestPredict15min,
	},
}

// ContainerCPULoadAverageGroup represents container cpu load average metrics
var ContainerCPULoadAverageGroup = MetricGroup{
	Name:    "container_cpu_load_average",
	Help:    "represents container cpu load average metrics",
	Type:    ContainerMetricType,
	Scraper: "walle",
	Labels: []string{
		"id",
		"pod",
		"namespace",
		"image",
	},
	Metrics: []Metric{
		ContainerCPULoadAverage10s,
		ContainerCPULoadAverage10sMaxOver1min,
		ContainerCPULoadAverage10sMinOver1min,
		ContainerCPULoadAverage10sAvgOver1min,
		ContainerCPULoadAverage10sP99Over1min,
		ContainerCPULoadAverage10sPredict1min,
		ContainerCPULoadAverage10sMaxOver5min,
		ContainerCPULoadAverage10sMinOver5min,
		ContainerCPULoadAverage10sAvgOver5min,
		ContainerCPULoadAverage10sP99Over5min,
		ContainerCPULoadAverage10sPredict5min,
		ContainerCPULoadAverage10sMaxOver15min,
		ContainerCPULoadAverage10sMinOver15min,
		ContainerCPULoadAverage10sAvgOver15min,
		ContainerCPULoadAverage10sP99Over15min,
		ContainerCPULoadAverage10sPredict15min,
	},
}

// ContainerMemoryMetricsGroup represents container memory metrics
var ContainerMemoryMetricsGroup = MetricGroup{
	Name:    "container_memory_metrics",
	Help:    "represents container memory metrics",
	Type:    ContainerMetricType,
	Scraper: "walle",
	Labels: []string{
		"id",
		"pod",
		"namespace",
		"image",
	},
	Metrics: []Metric{
		ContainerMemoryAvailableBytes,
		ContainerMemoryUsageBytes,
		ContainerMemoryWorkingSetBytes,
	},
}

// NodeCPUUsageMetricsGroup represents node cpu usage metrics
var NodeCPUUsageMetricsGroup = MetricGroup{
	Name:    "node_cpu_usage_metrics",
	Help:    "represents node cpu usage metrics",
	Type:    NodeMetricType,
	Scraper: "walle",
	Metrics: []Metric{
		NodeCPUUsage,
		NodeCPUUsageMaxOver1min,
		NodeCPUUsageMinOver1min,
		NodeCPUUsageAvgOver1min,
		NodeCPUUsageP99Over1min,
		NodeCPUUsagePredict1min,
		NodeCPUUsageMaxOver5min,
		NodeCPUUsageMinOver5min,
		NodeCPUUsageAvgOver5min,
		NodeCPUUsageP99Over5min,
		NodeCPUUsagePredict5min,
		NodeCPUUsageMaxOver15min,
		NodeCPUUsageMinOver15min,
		NodeCPUUsageAvgOver15min,
		NodeCPUUsageP99Over15min,
		NodeCPUUsagePredict15min,
	},
}

// NodeSystemLoadMetricsGroup represents node system load metrics
var NodeSystemLoadMetricsGroup = MetricGroup{
	Name:    "node_system_load_metrics",
	Help:    "represents node system load metrics",
	Type:    NodeMetricType,
	Scraper: "walle",
	Metrics: []Metric{
		NodeLoad1m,
		NodeLoad1mMaxOver1min,
		NodeLoad1mMinOver1min,
		NodeLoad1mAvgOver1min,
		NodeLoad1mP99Over1min,
		NodeLoad1mPredict1min,
		NodeLoad1mMaxOver5min,
		NodeLoad1mMinOver5min,
		NodeLoad1mAvgOver5min,
		NodeLoad1mP99Over5min,
		NodeLoad1mPredict5min,
		NodeLoad1mMaxOver15min,
		NodeLoad1mMinOver15min,
		NodeLoad1mAvgOver15min,
		NodeLoad1mP99Over15min,
		NodeLoad1mPredict15min,
		NodeLoad5m,
		NodeLoad5mMaxOver1min,
		NodeLoad5mMinOver1min,
		NodeLoad5mAvgOver1min,
		NodeLoad5mP99Over1min,
		NodeLoad5mPredict1min,
		NodeLoad5mMaxOver5min,
		NodeLoad5mMinOver5min,
		NodeLoad5mAvgOver5min,
		NodeLoad5mP99Over5min,
		NodeLoad5mPredict5min,
		NodeLoad5mMaxOver15min,
		NodeLoad5mMinOver15min,
		NodeLoad5mAvgOver15min,
		NodeLoad5mP99Over15min,
		NodeLoad5mPredict15min,
		NodeLoad15m,
		NodeLoad15mMaxOver1min,
		NodeLoad15mMinOver1min,
		NodeLoad15mAvgOver1min,
		NodeLoad15mP99Over1min,
		NodeLoad15mPredict1min,
		NodeLoad15mMaxOver5min,
		NodeLoad15mMinOver5min,
		NodeLoad15mAvgOver5min,
		NodeLoad15mP99Over5min,
		NodeLoad15mPredict5min,
		NodeLoad15mMaxOver15min,
		NodeLoad15mMinOver15min,
		NodeLoad15mAvgOver15min,
		NodeLoad15mP99Over15min,
		NodeLoad15mPredict15min,
	},
}

// NodeMemoryMetricsGroup represents node memory metrics
var NodeMemoryMetricsGroup = MetricGroup{
	Name:    "node_memory_metrics",
	Help:    "represents node memory metrics",
	Type:    NodeMetricType,
	Scraper: "walle",
	Metrics: []Metric{
		NodeMemoryAvailableBytes,
		NodeMemoryUsedBytes,
		NodeMemoryWorkingsetBytes,
	},
}

// ContainerCPUUsageLimit represents container cpu utilization relative to Resource.CPU.Limit
var ContainerCPUUsageLimit = Metric{
	Name: "container_cpu_usage_limit",
	Help: "represents container cpu utilization relative to Resource.CPU.Limit",
	Expr: "container_cpu_usage_limit",
}

// ContainerCPUUsageLimitMaxOver1min aggregates the max value of container_cpu_usage_limit over the last minute
var ContainerCPUUsageLimitMaxOver1min = Metric{
	Name: "container_cpu_usage_limit_max_over_1min",
	Help: "aggregates the max value of container_cpu_usage_limit over the last minute",
	Expr: "max_over_time(container_cpu_usage_limit[1m])",
}

// ContainerCPUUsageLimitMinOver1min aggregates the min value of container_cpu_usage_limit over the last minute
var ContainerCPUUsageLimitMinOver1min = Metric{
	Name: "container_cpu_usage_limit_min_over_1min",
	Help: "aggregates the min value of container_cpu_usage_limit over the last minute",
	Expr: "min_over_time(container_cpu_usage_limit[1m])",
}

// ContainerCPUUsageLimitAvgOver1min aggregates the average value of container_cpu_usage_limit over the last minute
var ContainerCPUUsageLimitAvgOver1min = Metric{
	Name: "container_cpu_usage_limit_avg_over_1min",
	Help: "aggregates the average value of container_cpu_usage_limit over the last minute",
	Expr: "avg_over_time(container_cpu_usage_limit[1m])",
}

// ContainerCPUUsageLimitP99Over1min aggregates the P99 value of container_cpu_usage_limit over the last minute
var ContainerCPUUsageLimitP99Over1min = Metric{
	Name: "container_cpu_usage_limit_p99_over_1min",
	Help: "aggregates the P99 value of container_cpu_usage_limit over the last minute",
	Expr: "quantile_over_time(0.99, container_cpu_usage_limit[1m])",
}

// ContainerCPUUsageLimitPredict1min predicts the value of container_cpu_usage_limit over the last minute
var ContainerCPUUsageLimitPredict1min = Metric{
	Name: "container_cpu_usage_limit_predict_1min",
	Help: "predicts the value of container_cpu_usage_limit over the last minute",
	Expr: "predict_linear(container_cpu_usage_limit[1m], 60)",
}

// ContainerCPUUsageLimitMaxOver5min aggregates the max value of container_cpu_usage_limit over the last 5 minutess
var ContainerCPUUsageLimitMaxOver5min = Metric{
	Name: "container_cpu_usage_limit_max_over_5min",
	Help: "aggregates the max value of container_cpu_usage_limit over the last 5 minutess",
	Expr: "max_over_time(container_cpu_usage_limit[5m])",
}

// ContainerCPUUsageLimitMinOver5min aggregates the min value of container_cpu_usage_limit over the last 5 minutess
var ContainerCPUUsageLimitMinOver5min = Metric{
	Name: "container_cpu_usage_limit_min_over_5min",
	Help: "aggregates the min value of container_cpu_usage_limit over the last 5 minutess",
	Expr: "min_over_time(container_cpu_usage_limit[5m])",
}

// ContainerCPUUsageLimitAvgOver5min aggregates the average value of container_cpu_usage_limit over the last 5 minutess
var ContainerCPUUsageLimitAvgOver5min = Metric{
	Name: "container_cpu_usage_limit_avg_over_5min",
	Help: "aggregates the average value of container_cpu_usage_limit over the last 5 minutess",
	Expr: "avg_over_time(container_cpu_usage_limit[5m])",
}

// ContainerCPUUsageLimitP99Over5min aggregates the P99 value of container_cpu_usage_limit over the last 5 minutess
var ContainerCPUUsageLimitP99Over5min = Metric{
	Name: "container_cpu_usage_limit_p99_over_5min",
	Help: "aggregates the P99 value of container_cpu_usage_limit over the last 5 minutess",
	Expr: "quantile_over_time(0.99, container_cpu_usage_limit[5m])",
}

// ContainerCPUUsageLimitPredict5min predicts the value of container_cpu_usage_limit over the last 5 minutess
var ContainerCPUUsageLimitPredict5min = Metric{
	Name: "container_cpu_usage_limit_predict_5min",
	Help: "predicts the value of container_cpu_usage_limit over the last 5 minutess",
	Expr: "predict_linear(container_cpu_usage_limit[5m], 60)",
}

// ContainerCPUUsageLimitMaxOver15min aggregates the max value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageLimitMaxOver15min = Metric{
	Name: "container_cpu_usage_limit_max_over_15min",
	Help: "aggregates the max value of container_cpu_usage_request over the last 15 minutes",
	Expr: "max_over_time(container_cpu_usage_limit[15m])",
}

// ContainerCPUUsageLimitMinOver15min aggregates the min value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageLimitMinOver15min = Metric{
	Name: "container_cpu_usage_limit_min_over_15min",
	Help: "aggregates the min value of container_cpu_usage_request over the last 15 minutes",
	Expr: "min_over_time(container_cpu_usage_limit[15m])",
}

// ContainerCPUUsageLimitAvgOver15min aggregates the average value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageLimitAvgOver15min = Metric{
	Name: "container_cpu_usage_limit_avg_over_15min",
	Help: "aggregates the average value of container_cpu_usage_request over the last 15 minutes",
	Expr: "avg_over_time(container_cpu_usage_limit[15m])",
}

// ContainerCPUUsageLimitP99Over15min aggregates the P99 value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageLimitP99Over15min = Metric{
	Name: "container_cpu_usage_limit_p99_over_15min",
	Help: "aggregates the P99 value of container_cpu_usage_request over the last 15 minutes",
	Expr: "quantile_over_time(0.99, container_cpu_usage_limit[15m])",
}

// ContainerCPUUsageLimitPredict15min predicts the value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageLimitPredict15min = Metric{
	Name: "container_cpu_usage_limit_predict_15min",
	Help: "predicts the value of container_cpu_usage_request over the last 15 minutes",
	Expr: "predict_linear(container_cpu_usage_limit[15m], 60)",
}

// ContainerCPUUsageRequest represents container cpu utilization relative to Resource.CPU.Request
var ContainerCPUUsageRequest = Metric{
	Name: "container_cpu_usage_request",
	Help: "represents container cpu utilization relative to Resource.CPU.Request",
	Expr: "container_cpu_usage_request",
}

// ContainerCPUUsageRequestMaxOver1min aggregates the max value of container_cpu_usage_request over the last minute
var ContainerCPUUsageRequestMaxOver1min = Metric{
	Name: "container_cpu_usage_request_max_over_1min",
	Help: "aggregates the max value of container_cpu_usage_request over the last minute",
	Expr: "max_over_time(container_cpu_usage_request[1m])",
}

// ContainerCPUUsageRequestMinOver1min aggregates the min value of container_cpu_usage_request over the last minute
var ContainerCPUUsageRequestMinOver1min = Metric{
	Name: "container_cpu_usage_request_min_over_1min",
	Help: "aggregates the min value of container_cpu_usage_request over the last minute",
	Expr: "min_over_time(container_cpu_usage_request[1m])",
}

// ContainerCPUUsageRequestAvgOver1min aggregates the average value of container_cpu_usage_request over the last minute
var ContainerCPUUsageRequestAvgOver1min = Metric{
	Name: "container_cpu_usage_request_avg_over_1min",
	Help: "aggregates the average value of container_cpu_usage_request over the last minute",
	Expr: "avg_over_time(container_cpu_usage_request[1m])",
}

// ContainerCPUUsageRequestP99Over1min aggregates the P99 value of container_cpu_usage_request over the last minute
var ContainerCPUUsageRequestP99Over1min = Metric{
	Name: "container_cpu_usage_request_p99_over_1min",
	Help: "aggregates the P99 value of container_cpu_usage_request over the last minute",
	Expr: "quantile_over_time(0.99, container_cpu_usage_request[1m])",
}

// ContainerCPUUsageRequestPredict1min predicts the value of container_cpu_usage_request over the last minute
var ContainerCPUUsageRequestPredict1min = Metric{
	Name: "container_cpu_usage_request_predict_1min",
	Help: "predicts the value of container_cpu_usage_request over the last minute",
	Expr: "predict_linear(container_cpu_usage_request[1m], 60)",
}

// ContainerCPUUsageRequestMaxOver5min aggregates the max value of container_cpu_usage_request over the last 5 minutes
var ContainerCPUUsageRequestMaxOver5min = Metric{
	Name: "container_cpu_usage_request_max_over_5min",
	Help: "aggregates the max value of container_cpu_usage_request over the last 5 minutes",
	Expr: "max_over_time(container_cpu_usage_request[5m])",
}

// ContainerCPUUsageRequestMinOver5min aggregates the min value of container_cpu_usage_request over the last 5 minutes
var ContainerCPUUsageRequestMinOver5min = Metric{
	Name: "container_cpu_usage_request_min_over_5min",
	Help: "aggregates the min value of container_cpu_usage_request over the last 5 minutes",
	Expr: "min_over_time(container_cpu_usage_request[5m])",
}

// ContainerCPUUsageRequestAvgOver5min aggregates the average max value of container_cpu_usage_request over the last 5 minutes
var ContainerCPUUsageRequestAvgOver5min = Metric{
	Name: "container_cpu_usage_request_avg_over_5min",
	Help: "aggregates the average max value of container_cpu_usage_request over the last 5 minutes",
	Expr: "avg_over_time(container_cpu_usage_request[5m])",
}

// ContainerCPUUsageRequestP99Over5min aggregates the P99 value of container_cpu_usage_request over the last 5 minutes
var ContainerCPUUsageRequestP99Over5min = Metric{
	Name: "container_cpu_usage_request_p99_over_5min",
	Help: "aggregates the P99 value of container_cpu_usage_request over the last 5 minutes",
	Expr: "quantile_over_time(0.99, container_cpu_usage_request[5m])",
}

// ContainerCPUUsageRequestPredict5min predicts the value of container_cpu_usage_request over the last 5 minutes
var ContainerCPUUsageRequestPredict5min = Metric{
	Name: "container_cpu_usage_request_predict_5min",
	Help: "predicts the value of container_cpu_usage_request over the last 5 minutes",
	Expr: "predict_linear(container_cpu_usage_request[5m], 60)",
}

// ContainerCPUUsageRequestMaxOver15min aggregates the max value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageRequestMaxOver15min = Metric{
	Name: "container_cpu_usage_request_max_over_15min",
	Help: "aggregates the max value of container_cpu_usage_request over the last 15 minutes",
	Expr: "max_over_time(container_cpu_usage_request[15m])",
}

// ContainerCPUUsageRequestMinOver15min aggregates the min value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageRequestMinOver15min = Metric{
	Name: "container_cpu_usage_request_min_over_15min",
	Help: "aggregates the min value of container_cpu_usage_request over the last 15 minutes",
	Expr: "min_over_time(container_cpu_usage_request[15m])",
}

// ContainerCPUUsageRequestAvgOver15min aggregates the average value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageRequestAvgOver15min = Metric{
	Name: "container_cpu_usage_request_avg_over_15min",
	Help: "aggregates the average value of container_cpu_usage_request over the last 15 minutes",
	Expr: "avg_over_time(container_cpu_usage_request[15m])",
}

// ContainerCPUUsageRequestP99Over15min aggregates P99 value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageRequestP99Over15min = Metric{
	Name: "container_cpu_usage_request_p99_over_15min",
	Help: "aggregates P99 value of container_cpu_usage_request over the last 15 minutes",
	Expr: "quantile_over_time(0.99, container_cpu_usage_request[15m])",
}

// ContainerCPUUsageRequestPredict15min predicts the value of container_cpu_usage_request over the last 15 minutes
var ContainerCPUUsageRequestPredict15min = Metric{
	Name: "container_cpu_usage_request_predict_15min",
	Help: "predicts the value of container_cpu_usage_request over the last 15 minutes",
	Expr: "predict_linear(container_cpu_usage_request[15m], 60)",
}

// ContainerCPULoadAverage10s represents container CPU Load over the last 10 seconds
var ContainerCPULoadAverage10s = Metric{
	Name: "container_cpu_load_average_10s",
	Help: "represents container CPU Load over the last 10 seconds",
	Expr: "container_cpu_load_average_10s",
}

// ContainerCPULoadAverage10sMaxOver1min aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last minute
var ContainerCPULoadAverage10sMaxOver1min = Metric{
	Name: "container_cpu_load_average_10s_max_over_1min",
	Help: "aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last minute",
	Expr: "max_over_time(container_cpu_load_average_10s[1m])",
}

// ContainerCPULoadAverage10sMinOver1min aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last minute
var ContainerCPULoadAverage10sMinOver1min = Metric{
	Name: "container_cpu_load_average_10s_min_over_1min",
	Help: "aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last minute",
	Expr: "min_over_time(container_cpu_load_average_10s[1m])",
}

// ContainerCPULoadAverage10sAvgOver1min aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last minute
var ContainerCPULoadAverage10sAvgOver1min = Metric{
	Name: "container_cpu_load_average_10s_avg_over_1min",
	Help: "aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last minute",
	Expr: "avg_over_time(container_cpu_load_average_10s[1m])",
}

// ContainerCPULoadAverage10sP99Over1min aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last minute
var ContainerCPULoadAverage10sP99Over1min = Metric{
	Name: "container_cpu_load_average_10s_p99_over_1min",
	Help: "aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last minute",
	Expr: "quantile_over_time(0.99, container_cpu_load_average_10s[1m])",
}

// ContainerCPULoadAverage10sPredict1min predicts the value of container_cpu_load_average_10s_max_over_1min over the last minute
var ContainerCPULoadAverage10sPredict1min = Metric{
	Name: "container_cpu_load_average_10s_predict_1min",
	Help: "predicts the value of container_cpu_load_average_10s_max_over_1min over the last minute",
	Expr: "predict_linear(container_cpu_load_average_10s[1m], 60)",
}

// ContainerCPULoadAverage10sMaxOver5min aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes
var ContainerCPULoadAverage10sMaxOver5min = Metric{
	Name: "container_cpu_load_average_10s_max_over_5min",
	Help: "aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes",
	Expr: "max_over_time(container_cpu_load_average_10s[5m])",
}

// ContainerCPULoadAverage10sMinOver5min aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes
var ContainerCPULoadAverage10sMinOver5min = Metric{
	Name: "container_cpu_load_average_10s_min_over_5min",
	Help: "aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes",
	Expr: "min_over_time(container_cpu_load_average_10s[5m])",
}

// ContainerCPULoadAverage10sAvgOver5min aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes
var ContainerCPULoadAverage10sAvgOver5min = Metric{
	Name: "container_cpu_load_average_10s_avg_over_5min",
	Help: "aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes",
	Expr: "avg_over_time(container_cpu_load_average_10s[5m])",
}

// ContainerCPULoadAverage10sP99Over5min aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes
var ContainerCPULoadAverage10sP99Over5min = Metric{
	Name: "container_cpu_load_average_10s_p99_over_5min",
	Help: "aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes",
	Expr: "quantile_over_time(0.99, container_cpu_load_average_10s[5m])",
}

// ContainerCPULoadAverage10sPredict5min predicts the value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes
var ContainerCPULoadAverage10sPredict5min = Metric{
	Name: "container_cpu_load_average_10s_predict_5min",
	Help: "predicts the value of container_cpu_load_average_10s_max_over_1min over the last 5 minutes",
	Expr: "predict_linear(container_cpu_load_average_10s[5m], 60)",
}

// ContainerCPULoadAverage10sMaxOver15min aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes
var ContainerCPULoadAverage10sMaxOver15min = Metric{
	Name: "container_cpu_load_average_10s_max_over_15min",
	Help: "aggregates the max value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes",
	Expr: "max_over_time(container_cpu_load_average_10s[15m])",
}

// ContainerCPULoadAverage10sMinOver15min aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes
var ContainerCPULoadAverage10sMinOver15min = Metric{
	Name: "container_cpu_load_average_10s_min_over_15min",
	Help: "aggregates the min value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes",
	Expr: "min_over_time(container_cpu_load_average_10s[15m])",
}

// ContainerCPULoadAverage10sAvgOver15min aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes
var ContainerCPULoadAverage10sAvgOver15min = Metric{
	Name: "container_cpu_load_average_10s_avg_over_15min",
	Help: "aggregates the average value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes",
	Expr: "avg_over_time(container_cpu_load_average_10s[15m])",
}

// ContainerCPULoadAverage10sP99Over15min aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes
var ContainerCPULoadAverage10sP99Over15min = Metric{
	Name: "container_cpu_load_average_10s_p99_over_15min",
	Help: "aggregates the P99 value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes",
	Expr: "quantile_over_time(0.99, container_cpu_load_average_10s[15m])",
}

// ContainerCPULoadAverage10sPredict15min predicts the value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes
var ContainerCPULoadAverage10sPredict15min = Metric{
	Name: "container_cpu_load_average_10s_predict_15min",
	Help: "predicts the value of container_cpu_load_average_10s_max_over_1min over the last 15 minutes",
	Expr: "predict_linear(container_cpu_load_average_10s[15m], 60)",
}

// ContainerMemoryAvailableBytes represents the available memory bytes of container
var ContainerMemoryAvailableBytes = Metric{
	Name: "container_memory_available_bytes",
	Help: "represents the available memory bytes of container",
	Expr: "container_memory_avail_bytes",
}

// ContainerMemoryUsageBytes represents the used memory bytes of container
var ContainerMemoryUsageBytes = Metric{
	Name: "container_memory_usage_bytes",
	Help: "represents the used memory bytes of container",
	Expr: "container_memory_usage_bytes",
}

// ContainerMemoryWorkingSetBytes represents the workingset bytes of container
var ContainerMemoryWorkingSetBytes = Metric{
	Name: "container_memory_working_set_bytes",
	Help: "represents the workingset bytes of container",
	Expr: "container_memory_working_set_bytes",
}

// NodeCPUUsage represents the system CPU utilization
var NodeCPUUsage = Metric{
	Name: "node_cpu_usage",
	Help: "represents the system CPU utilization",
	Expr: "node_cpu_usage",
}

// NodeCPUUsageMaxOver1min aggregates the max value of node_cpu_usage over the last minute
var NodeCPUUsageMaxOver1min = Metric{
	Name: "node_cpu_usage_max_over_1min",
	Help: "aggregates the max value of node_cpu_usage over the last minute",
	Expr: "max_over_time(node_cpu_usage[1m])",
}

// NodeCPUUsageMinOver1min aggregates the min value of node_cpu_usage over the last minute
var NodeCPUUsageMinOver1min = Metric{
	Name: "node_cpu_usage_min_over_1min",
	Help: "aggregates the min value of node_cpu_usage over the last minute",
	Expr: "min_over_time(node_cpu_usage[1m])",
}

// NodeCPUUsageAvgOver1min aggregates the average value of node_cpu_usage over the last minute
var NodeCPUUsageAvgOver1min = Metric{
	Name: "node_cpu_usage_avg_over_1min",
	Help: "aggregates the average value of node_cpu_usage over the last minute",
	Expr: "avg_over_time(node_cpu_usage[1m])",
}

// NodeCPUUsageP99Over1min aggregates the P99 value of node_cpu_usage over the last minute
var NodeCPUUsageP99Over1min = Metric{
	Name: "node_cpu_usage_p99_over_1min",
	Help: "aggregates the P99 value of node_cpu_usage over the last minute",
	Expr: "quantile_over_time(0.99, node_cpu_usage[1m])",
}

// NodeCPUUsagePredict1min predicts the value of node_cpu_usage over the last minute
var NodeCPUUsagePredict1min = Metric{
	Name: "node_cpu_usage_predict_1min",
	Help: "predicts the value of node_cpu_usage over the last minute",
	Expr: "predict_linear(node_cpu_usage[1m], 60)",
}

// NodeCPUUsageMaxOver5min aggregates the max value of node_cpu_usage over the last 5 minutess
var NodeCPUUsageMaxOver5min = Metric{
	Name: "node_cpu_usage_max_over_5min",
	Help: "aggregates the max value of node_cpu_usage over the last 5 minutess",
	Expr: "max_over_time(node_cpu_usage[5m])",
}

// NodeCPUUsageMinOver5min aggregates the min value of node_cpu_usage over the last 5 minutess
var NodeCPUUsageMinOver5min = Metric{
	Name: "node_cpu_usage_min_over_5min",
	Help: "aggregates the min value of node_cpu_usage over the last 5 minutess",
	Expr: "min_over_time(node_cpu_usage[5m])",
}

// NodeCPUUsageAvgOver5min aggregates the average value of node_cpu_usage over the last 5 minutess
var NodeCPUUsageAvgOver5min = Metric{
	Name: "node_cpu_usage_avg_over_5min",
	Help: "aggregates the average value of node_cpu_usage over the last 5 minutess",
	Expr: "avg_over_time(node_cpu_usage[5m])",
}

// NodeCPUUsageP99Over5min aggregates the P99 value of node_cpu_usage over the last 5 minutess
var NodeCPUUsageP99Over5min = Metric{
	Name: "node_cpu_usage_p99_over_5min",
	Help: "aggregates the P99 value of node_cpu_usage over the last 5 minutess",
	Expr: "quantile_over_time(0.99, node_cpu_usage[5m])",
}

// NodeCPUUsagePredict5min predicts the value of node_cpu_usage over the last 5 minutess
var NodeCPUUsagePredict5min = Metric{
	Name: "node_cpu_usage_predict_5min",
	Help: "predicts the value of node_cpu_usage over the last 5 minutess",
	Expr: "predict_linear(node_cpu_usage[5m], 60)",
}

// NodeCPUUsageMaxOver15min aggregates the max value of node_cpu_usage over the last 15 minutess
var NodeCPUUsageMaxOver15min = Metric{
	Name: "node_cpu_usage_max_over_15min",
	Help: "aggregates the max value of node_cpu_usage over the last 15 minutess",
	Expr: "max_over_time(node_cpu_usage[15m])",
}

// NodeCPUUsageMinOver15min aggregates the min value of node_cpu_usage over the last 15 minutess
var NodeCPUUsageMinOver15min = Metric{
	Name: "node_cpu_usage_min_over_15min",
	Help: "aggregates the min value of node_cpu_usage over the last 15 minutess",
	Expr: "min_over_time(node_cpu_usage[15m])",
}

// NodeCPUUsageAvgOver15min aggregates the average value of node_cpu_usage over the last 15 minutess
var NodeCPUUsageAvgOver15min = Metric{
	Name: "node_cpu_usage_avg_over_15min",
	Help: "aggregates the average value of node_cpu_usage over the last 15 minutess",
	Expr: "avg_over_time(node_cpu_usage[15m])",
}

// NodeCPUUsageP99Over15min aggregates the P99 value of node_cpu_usage over the last 15 minutess
var NodeCPUUsageP99Over15min = Metric{
	Name: "node_cpu_usage_p99_over_15min",
	Help: "aggregates the P99 value of node_cpu_usage over the last 15 minutess",
	Expr: "quantile_over_time(0.99, node_cpu_usage[15m])",
}

// NodeCPUUsagePredict15min predicts the value of node_cpu_usage over the last 15 minutess
var NodeCPUUsagePredict15min = Metric{
	Name: "node_cpu_usage_predict_15min",
	Help: "predicts the value of node_cpu_usage over the last 15 minutess",
	Expr: "predict_linear(node_cpu_usage[15m], 60)",
}

// NodeLoad1m represents system load average over the last minute
var NodeLoad1m = Metric{
	Name: "node_load_1m",
	Help: "represents system load average over the last minute",
	Expr: "node_load_1m",
}

// NodeLoad1mMaxOver1min aggregates the max value of node_cpunode_load_1m_usage over the last minute
var NodeLoad1mMaxOver1min = Metric{
	Name: "node_load_1m_max_over_1min",
	Help: "aggregates the max value of node_cpunode_load_1m_usage over the last minute",
	Expr: "max_over_time(node_load_1m[1m])",
}

// NodeLoad1mMinOver1min aggregates the min value of node_cpunode_load_1m_usage over the last minute
var NodeLoad1mMinOver1min = Metric{
	Name: "node_load_1m_min_over_1min",
	Help: "aggregates the min value of node_cpunode_load_1m_usage over the last minute",
	Expr: "min_over_time(node_load_1m[1m])",
}

// NodeLoad1mAvgOver1min aggregates the average value of node_cpunode_load_1m_usage over the last minute
var NodeLoad1mAvgOver1min = Metric{
	Name: "node_load_1m_avg_over_1min",
	Help: "aggregates the average value of node_cpunode_load_1m_usage over the last minute",
	Expr: "avg_over_time(node_load_1m[1m])",
}

// NodeLoad1mP99Over1min aggregates the P99 value of node_cpunode_load_1m_usage over the last minute
var NodeLoad1mP99Over1min = Metric{
	Name: "node_load_1m_p99_over_1min",
	Help: "aggregates the P99 value of node_cpunode_load_1m_usage over the last minute",
	Expr: "quantile_over_time(0.99, node_load_1m[1m])",
}

// NodeLoad1mPredict1min predicts the value of node_cpunode_load_1m_usage over the last minute
var NodeLoad1mPredict1min = Metric{
	Name: "node_load_1m_predict_1min",
	Help: "predicts the value of node_cpunode_load_1m_usage over the last minute",
	Expr: "predict_linear(node_load_1m[1m], 60)",
}

// NodeLoad1mMaxOver5min aggregates the max value of node_cpunode_load_1m_usage over the last 5 minutess
var NodeLoad1mMaxOver5min = Metric{
	Name: "node_load_1m_max_over_5min",
	Help: "aggregates the max value of node_cpunode_load_1m_usage over the last 5 minutess",
	Expr: "max_over_time(node_load_1m[5m])",
}

// NodeLoad1mMinOver5min aggregates the min value of node_cpunode_load_1m_usage over the last 5 minutess
var NodeLoad1mMinOver5min = Metric{
	Name: "node_load_1m_min_over_5min",
	Help: "aggregates the min value of node_cpunode_load_1m_usage over the last 5 minutess",
	Expr: "min_over_time(node_load_1m[5m])",
}

// NodeLoad1mAvgOver5min aggregates the average value of node_cpunode_load_1m_usage over the last 5 minutess
var NodeLoad1mAvgOver5min = Metric{
	Name: "node_load_1m_avg_over_5min",
	Help: "aggregates the average value of node_cpunode_load_1m_usage over the last 5 minutess",
	Expr: "avg_over_time(node_load_1m[5m])",
}

// NodeLoad1mP99Over5min aggregates the P99 value of node_cpunode_load_1m_usage over the last 5 minutess
var NodeLoad1mP99Over5min = Metric{
	Name: "node_load_1m_p99_over_5min",
	Help: "aggregates the P99 value of node_cpunode_load_1m_usage over the last 5 minutess",
	Expr: "quantile_over_time(0.99, node_load_1m[5m])",
}

// NodeLoad1mPredict5min predicts the value of node_cpunode_load_1m_usage over the last 5 minutess
var NodeLoad1mPredict5min = Metric{
	Name: "node_load_1m_predict_5min",
	Help: "predicts the value of node_cpunode_load_1m_usage over the last 5 minutess",
	Expr: "predict_linear(node_load_1m[5m], 60)",
}

// NodeLoad1mMaxOver15min aggregates the max value of node_cpunode_load_1m_usage over the last 15 minutess
var NodeLoad1mMaxOver15min = Metric{
	Name: "node_load_1m_max_over_15min",
	Help: "aggregates the max value of node_cpunode_load_1m_usage over the last 15 minutess",
	Expr: "max_over_time(node_load_1m[15m])",
}

// NodeLoad1mMinOver15min aggregates the min value of node_cpunode_load_1m_usage over the last 15 minutess
var NodeLoad1mMinOver15min = Metric{
	Name: "node_load_1m_min_over_15min",
	Help: "aggregates the min value of node_cpunode_load_1m_usage over the last 15 minutess",
	Expr: "min_over_time(node_load_1m[15m])",
}

// NodeLoad1mAvgOver15min aggregates the average value of node_cpunode_load_1m_usage over the last 15 minutess
var NodeLoad1mAvgOver15min = Metric{
	Name: "node_load_1m_avg_over_15min",
	Help: "aggregates the average value of node_cpunode_load_1m_usage over the last 15 minutess",
	Expr: "avg_over_time(node_load_1m[15m])",
}

// NodeLoad1mP99Over15min aggregates the P99 value of node_cpunode_load_1m_usage over the last 15 minutess
var NodeLoad1mP99Over15min = Metric{
	Name: "node_load_1m_p99_over_15min",
	Help: "aggregates the P99 value of node_cpunode_load_1m_usage over the last 15 minutess",
	Expr: "quantile_over_time(0.99, node_load_1m[15m])",
}

// NodeLoad1mPredict15min predicts the value of node_cpunode_load_1m_usage over the last 15 minutess
var NodeLoad1mPredict15min = Metric{
	Name: "node_load_1m_predict_15min",
	Help: "predicts the value of node_cpunode_load_1m_usage over the last 15 minutess",
	Expr: "predict_linear(node_load_1m[15m], 60)",
}

// NodeLoad5m represents system load average over the last 5 minutess
var NodeLoad5m = Metric{
	Name: "node_load_5m",
	Help: "represents system load average over the last 5 minutess",
	Expr: "node_load_5m",
}

// NodeLoad5mMaxOver1min aggregates the max value of node_load_5m over the last minute
var NodeLoad5mMaxOver1min = Metric{
	Name: "node_load_5m_max_over_1min",
	Help: "aggregates the max value of node_load_5m over the last minute",
	Expr: "max_over_time(node_load_5m[1m])",
}

// NodeLoad5mMinOver1min aggregates the min value of node_load_5m over the last minute
var NodeLoad5mMinOver1min = Metric{
	Name: "node_load_5m_min_over_1min",
	Help: "aggregates the min value of node_load_5m over the last minute",
	Expr: "min_over_time(node_load_5m[1m])",
}

// NodeLoad5mAvgOver1min aggregates the average value of node_load_5m over the last minute
var NodeLoad5mAvgOver1min = Metric{
	Name: "node_load_5m_avg_over_1min",
	Help: "aggregates the average value of node_load_5m over the last minute",
	Expr: "avg_over_time(node_load_5m[1m])",
}

// NodeLoad5mP99Over1min aggregates the P99 value of node_load_5m over the last minute
var NodeLoad5mP99Over1min = Metric{
	Name: "node_load_5m_p99_over_1min",
	Help: "aggregates the P99 value of node_load_5m over the last minute",
	Expr: "quantile_over_time(0.99, node_load_5m[1m])",
}

// NodeLoad5mPredict1min predicts the value of node_load_5m over the last minute
var NodeLoad5mPredict1min = Metric{
	Name: "node_load_5m_predict_1min",
	Help: "predicts the value of node_load_5m over the last minute",
	Expr: "predict_linear(node_load_5m[1m], 60)",
}

// NodeLoad5mMaxOver5min aggregates the max value of node_load_5m over the last 5 minutess
var NodeLoad5mMaxOver5min = Metric{
	Name: "node_load_5m_max_over_5min",
	Help: "aggregates the max value of node_load_5m over the last 5 minutess",
	Expr: "max_over_time(node_load_5m[5m])",
}

// NodeLoad5mMinOver5min aggregates the min value of node_load_5m over the last 5 minutess
var NodeLoad5mMinOver5min = Metric{
	Name: "node_load_5m_min_over_5min",
	Help: "aggregates the min value of node_load_5m over the last 5 minutess",
	Expr: "min_over_time(node_load_5m[5m])",
}

// NodeLoad5mAvgOver5min aggregates the average value of node_load_5m over the last 5 minutess
var NodeLoad5mAvgOver5min = Metric{
	Name: "node_load_5m_avg_over_5min",
	Help: "aggregates the average value of node_load_5m over the last 5 minutess",
	Expr: "avg_over_time(node_load_5m[5m])",
}

// NodeLoad5mP99Over5min aggregates the P99 value of node_load_5m over the last 5 minutess
var NodeLoad5mP99Over5min = Metric{
	Name: "node_load_5m_p99_over_5min",
	Help: "aggregates the P99 value of node_load_5m over the last 5 minutess",
	Expr: "quantile_over_time(0.99, node_load_5m[5m])",
}

// NodeLoad5mPredict5min predicts the value of node_load_5m over the last 5 minutess
var NodeLoad5mPredict5min = Metric{
	Name: "node_load_5m_predict_5min",
	Help: "predicts the value of node_load_5m over the last 5 minutess",
	Expr: "predict_linear(node_load_5m[5m], 60)",
}

// NodeLoad5mMaxOver15min aggregates the max value of node_load_5m over the last 15 minutess
var NodeLoad5mMaxOver15min = Metric{
	Name: "node_load_5m_max_over_15min",
	Help: "aggregates the max value of node_load_5m over the last 15 minutess",
	Expr: "max_over_time(node_load_5m[15m])",
}

// NodeLoad5mMinOver15min aggregates the min value of node_load_5m over the last 15 minutess
var NodeLoad5mMinOver15min = Metric{
	Name: "node_load_5m_min_over_15min",
	Help: "aggregates the min value of node_load_5m over the last 15 minutess",
	Expr: "min_over_time(node_load_5m[15m])",
}

// NodeLoad5mAvgOver15min aggregates the average value of node_load_5m over the last 15 minutess
var NodeLoad5mAvgOver15min = Metric{
	Name: "node_load_5m_avg_over_15min",
	Help: "aggregates the average value of node_load_5m over the last 15 minutess",
	Expr: "avg_over_time(node_load_5m[15m])",
}

// NodeLoad5mP99Over15min aggregates the P99 value of node_load_5m over the last 15 minutess
var NodeLoad5mP99Over15min = Metric{
	Name: "node_load_5m_p99_over_15min",
	Help: "aggregates the P99 value of node_load_5m over the last 15 minutess",
	Expr: "quantile_over_time(0.99, node_load_5m[15m])",
}

// NodeLoad5mPredict15min predicts the value of node_load_5m over the last 15 minutess
var NodeLoad5mPredict15min = Metric{
	Name: "node_load_5m_predict_15min",
	Help: "predicts the value of node_load_5m over the last 15 minutess",
	Expr: "predict_linear(node_load_5m[15m], 60)",
}

// NodeLoad15m represents system load average over the last 15 minutess
var NodeLoad15m = Metric{
	Name: "node_load_15m",
	Help: "represents system load average over the last 15 minutess",
	Expr: "node_load_15m",
}

// NodeLoad15mMaxOver1min aggregates the max value of node_load_15m over the last minute
var NodeLoad15mMaxOver1min = Metric{
	Name: "node_load_15m_max_over_1min",
	Help: "aggregates the max value of node_load_15m over the last minute",
	Expr: "max_over_time(node_load_15m[1m])",
}

// NodeLoad15mMinOver1min aggregates the min value of node_load_15m over the last minute
var NodeLoad15mMinOver1min = Metric{
	Name: "node_load_15m_min_over_1min",
	Help: "aggregates the min value of node_load_15m over the last minute",
	Expr: "min_over_time(node_load_15m[1m])",
}

// NodeLoad15mAvgOver1min aggregates the average value of node_load_15m over the last minute
var NodeLoad15mAvgOver1min = Metric{
	Name: "node_load_15m_avg_over_1min",
	Help: "aggregates the average value of node_load_15m over the last minute",
	Expr: "avg_over_time(node_load_15m[1m])",
}

// NodeLoad15mP99Over1min aggregates the P99 value of node_load_15m over the last minute
var NodeLoad15mP99Over1min = Metric{
	Name: "node_load_15m_p99_over_1min",
	Help: "aggregates the P99 value of node_load_15m over the last minute",
	Expr: "quantile_over_time(0.99, node_load_15m[1m])",
}

// NodeLoad15mPredict1min predicts the value of node_load_15m over the last minute
var NodeLoad15mPredict1min = Metric{
	Name: "node_load_15m_predict_1min",
	Help: "predicts the value of node_load_15m over the last minute",
	Expr: "predict_linear(node_load_15m[1m], 60)",
}

// NodeLoad15mMaxOver5min aggregates the max value of node_load_15m over the last 5 minutess
var NodeLoad15mMaxOver5min = Metric{
	Name: "node_load_15m_max_over_5min",
	Help: "aggregates the max value of node_load_15m over the last 5 minutess",
	Expr: "max_over_time(node_load_15m[5m])",
}

// NodeLoad15mMinOver5min aggregates the min value of node_load_15m over the last 5 minutess
var NodeLoad15mMinOver5min = Metric{
	Name: "node_load_15m_min_over_5min",
	Help: "aggregates the min value of node_load_15m over the last 5 minutess",
	Expr: "min_over_time(node_load_15m[5m])",
}

// NodeLoad15mAvgOver5min aggregates the average value of node_load_15m over the last 5 minutess
var NodeLoad15mAvgOver5min = Metric{
	Name: "node_load_15m_avg_over_5min",
	Help: "aggregates the average value of node_load_15m over the last 5 minutess",
	Expr: "avg_over_time(node_load_15m[5m])",
}

// NodeLoad15mP99Over5min aggregates the P99 value of node_load_15m over the last 5 minutess
var NodeLoad15mP99Over5min = Metric{
	Name: "node_load_15m_p99_over_5min",
	Help: "aggregates the P99 value of node_load_15m over the last 5 minutess",
	Expr: "quantile_over_time(0.99, node_load_15m[5m])",
}

// NodeLoad15mPredict5min predicts the value of node_load_15m over the last 5 minutess
var NodeLoad15mPredict5min = Metric{
	Name: "node_load_15m_predict_5min",
	Help: "predicts the value of node_load_15m over the last 5 minutess",
	Expr: "predict_linear(node_load_15m[5m], 60)",
}

// NodeLoad15mMaxOver15min aggregates the max value of node_load_15m over the last 15 minutess
var NodeLoad15mMaxOver15min = Metric{
	Name: "node_load_15m_max_over_15min",
	Help: "aggregates the max value of node_load_15m over the last 15 minutess",
	Expr: "max_over_time(node_load_15m[15m])",
}

// NodeLoad15mMinOver15min aggregates the min value of node_load_15m over the last 15 minutess
var NodeLoad15mMinOver15min = Metric{
	Name: "node_load_15m_min_over_15min",
	Help: "aggregates the min value of node_load_15m over the last 15 minutess",
	Expr: "min_over_time(node_load_15m[15m])",
}

// NodeLoad15mAvgOver15min aggregates the average value of node_load_15m over the last 15 minutess
var NodeLoad15mAvgOver15min = Metric{
	Name: "node_load_15m_avg_over_15min",
	Help: "aggregates the average value of node_load_15m over the last 15 minutess",
	Expr: "avg_over_time(node_load_15m[15m])",
}

// NodeLoad15mP99Over15min aggregates the P99 value of node_load_15m over the last 15 minutess
var NodeLoad15mP99Over15min = Metric{
	Name: "node_load_15m_p99_over_15min",
	Help: "aggregates the P99 value of node_load_15m over the last 15 minutess",
	Expr: "quantile_over_time(0.99, node_load_15m[15m])",
}

// NodeLoad15mPredict15min predicts the value of node_load_15m over the last 15 minutess
var NodeLoad15mPredict15min = Metric{
	Name: "node_load_15m_predict_15min",
	Help: "predicts the value of node_load_15m over the last 15 minutess",
	Expr: "predict_linear(node_load_15m[15m], 60)",
}

// NodeMemoryAvailableBytes represents the available memory bytes in OS-level
var NodeMemoryAvailableBytes = Metric{
	Name: "node_memory_available_bytes",
	Help: "represents the available memory bytes in OS-level",
	Expr: "node_memory_total_bytes-node_memory_workingset_bytes",
}

// NodeMemoryUsedBytes represents the used memory bytes in OS-level
var NodeMemoryUsedBytes = Metric{
	Name: "node_memory_used_bytes",
	Help: "represents the used memory bytes in OS-level",
	Expr: "node_memory_used_bytes",
}

// NodeMemoryWorkingsetBytes represents bytes of active pages that represents working set for all processes
var NodeMemoryWorkingsetBytes = Metric{
	Name: "node_memory_workingset_bytes",
	Help: "represents bytes of active pages that represents working set for all processes",
	Expr: "node_memory_workingset_bytes",
}
