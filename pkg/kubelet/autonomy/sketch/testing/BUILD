load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "provider.go",
        "snapshoter.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/testing",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/autonomy/sketch:go_default_library",
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/stretchr/testify/mock:go_default_library",
    ],
)
