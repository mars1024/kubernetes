package opentracing

import (
	"fmt"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

// ExtentionFormat is the external type which corresponds to opentracing.BuiltinFormat.
type ExtentionFormat byte

const (
	// MetaObject it the external format for kubernetes meta.Object.
	MetaObject ExtentionFormat = iota
)

func keyForService(service string) string {
	return fmt.Sprintf("%s-%s", sigmak8sapi.AnnotationKeyTrace, service)
}

func compressedKeyForService(service string) string {
	return fmt.Sprintf("%s-%s", sigmak8sapi.AnnotationKeyCompressedTrace, service)
}

func keyForTraceID() string {
	return sigmak8sapi.AnnotationKeyTraceID
}
