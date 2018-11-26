load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["handler.go"],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/handler",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/autonomy/sketch:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/emicklei/go-restful:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["handler_test.go"],
    embed = [":go_default_library"],
    deps = [
        "//pkg/kubelet/autonomy/sketch:go_default_library",
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/testing:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/emicklei/go-restful:go_default_library",
        "//vendor/github.com/stretchr/testify/assert:go_default_library",
    ],
)
