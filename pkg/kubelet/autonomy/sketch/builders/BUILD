load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "build_container_cpu_sketch.go",
        "build_memory_sketch.go",
        "build_node_cpu_sketch.go",
        "build_sketch_data.go",
        "build_system_load_sketch.go",
        "sketch_builder.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/pod:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "build_memory_sketch_test.go",
        "build_sketch_data_test.go",
        "sketch_builder_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/kubelet/autonomy/sketch/api/v1alpha1:go_default_library",
        "//pkg/kubelet/autonomy/sketch/builders/testing:go_default_library",
        "//pkg/kubelet/autonomy/sketch/core:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/container/testing:go_default_library",
        "//pkg/kubelet/pod:go_default_library",
        "//pkg/kubelet/pod/testing:go_default_library",
        "//pkg/kubelet/types:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//vendor/github.com/golang/mock/gomock:go_default_library",
        "//vendor/github.com/stretchr/testify/mock:go_default_library",
    ],
)
