load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "EvaluableExpression.go",
        "EvaluableExpression_sql.go",
        "ExpressionToken.go",
        "OperatorSymbol.go",
        "TokenKind.go",
        "evaluationStage.go",
        "expressionFunctions.go",
        "expressionOutputStream.go",
        "lexerState.go",
        "lexerStream.go",
        "parameters.go",
        "parsing.go",
        "sanitizedParameters.go",
        "stagePlanner.go",
        "tokenStream.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/github.com/Knetic/govaluate",
    importpath = "github.com/Knetic/govaluate",
    visibility = ["//visibility:public"],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
