load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "metrics_scraper.go",
        "options.go",
        "prometheus_api.go",
        "provider.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/github.com/prometheus/client_golang/api:go_default_library",
        "//vendor/github.com/prometheus/client_golang/api/prometheus/v1:go_default_library",
        "//vendor/github.com/prometheus/client_golang/prometheus:go_default_library",
        "//vendor/github.com/prometheus/common/model:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "metrics_scraper_test.go",
        "options_test.go",
        "prometheus_api_test.go",
        "provider_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//pkg/kubelet/autonomy/sketch/scrapers/walle/testing:go_default_library",
        "//vendor/github.com/golang/mock/gomock:go_default_library",
        "//vendor/github.com/prometheus/client_golang/api:go_default_library",
        "//vendor/github.com/prometheus/client_golang/api/prometheus/v1:go_default_library",
        "//vendor/github.com/prometheus/common/model:go_default_library",
        "//vendor/github.com/stretchr/testify/assert:go_default_library",
    ],
)
