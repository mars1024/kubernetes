load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["sketch_builder_mock.go"],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders/testing",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//vendor/github.com/golang/mock/gomock:go_default_library",
    ],
)
