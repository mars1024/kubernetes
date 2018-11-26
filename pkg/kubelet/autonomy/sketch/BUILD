package(default_visibility = ["//visibility:public"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
    "go_test",
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "node_sketch_types.go",
        "options.go",
        "sketch.go",
        "snapshot.go",
        "types.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch",
    deps = [
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/builders:go_default_library",
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//pkg/kubelet/autonomy/sketch/scrapers:go_default_library",
        "//pkg/kubelet/autonomy/sketch/scrapers/walle:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/pod:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/spf13/pflag:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "options_test.go",
        "sketch_test.go",
        "snapshot_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/builders/testing:go_default_library",
        "//pkg/kubelet/autonomy/sketch/scrapers/walle/testing:go_default_library",
        "//pkg/kubelet/container/testing:go_default_library",
        "//pkg/kubelet/pod/testing:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/golang/mock/gomock:go_default_library",
        "//vendor/github.com/spf13/pflag:go_default_library",
        "//vendor/github.com/stretchr/testify/assert:go_default_library",
    ],
)
