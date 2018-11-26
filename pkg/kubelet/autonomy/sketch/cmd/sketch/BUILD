load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/cmd/sketch",
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/kubelet/autonomy/sketch:go_default_library",
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/handler:go_default_library",
        "//vendor/github.com/emicklei/go-restful:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/github.com/prometheus/client_golang/prometheus/promhttp:go_default_library",
        "//vendor/github.com/spf13/pflag:go_default_library",
    ],
)

go_binary(
    name = "sketch",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
