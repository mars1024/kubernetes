package namespacedelete

import (
	"testing"

	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func fakePodCounterZero(namespace string) (int, error) {
	return 0, nil
}

func fakePodCounterOne(namespace string) (int, error) {
	return 1, nil
}

func TestAdmissionNonNilAttributeZero(t *testing.T) {
	handler := new(auditDeletePlugin)
	handler.podCounterHandler = fakePodCounterZero
	err := handler.Validate(admission.NewAttributesRecord(nil, nil,
		api.Kind("Namespace").WithVersion("version"), "namespace", "name",
		api.Resource("resource").WithVersion("version"), "subresource",
		admission.Delete, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from AuditDelete admission controller: %v", err)
	}
}

func TestAdmissionNonNilAttributeOne(t *testing.T) {
	handler := new(auditDeletePlugin)
	handler.podCounterHandler = fakePodCounterOne
	err := handler.Validate(admission.NewAttributesRecord(nil, nil,
		api.Kind("Namespace").WithVersion("version"), "namespace", "name",
		api.Resource("resource").WithVersion("version"), "subresource",
		admission.Delete, nil))
	if err == nil {
		t.Errorf("Unexpected admit with one pod in the requested namespace")
	}
}

func TestAdmissionNilAttribute(t *testing.T) {
	handler := new(auditDeletePlugin)
	err := handler.Validate(nil)
	if err != nil {
		t.Errorf("Unexpected error returned from AuditDelete admission controller: %v", err)
	}
}
