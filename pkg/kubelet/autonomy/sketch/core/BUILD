load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "labels.go",
        "metrics.go",
        "types.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["types_test.go"],
    embed = [":go_default_library"],
    deps = ["//vendor/github.com/stretchr/testify/assert:go_default_library"],
)
