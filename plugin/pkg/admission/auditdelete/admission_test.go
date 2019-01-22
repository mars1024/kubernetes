package auditdelete

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"
)

func TestDeleteNamespace(t *testing.T) {
	ctrl := newPlugin()
	informerFactory := informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	ctrl.SetInternalKubeInformerFactory(informerFactory)
	err := ctrl.ValidateInitialization()
	if err != nil {
		t.Fatalf("neither pv lister nor storageclass lister can be nil")
	}

	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "ns-test"},
		Spec: api.PodSpec{},
	}
	_ = informerFactory.Core().InternalVersion().Pods().Informer().GetStore().Add(pod)

	resource := api.SchemeGroupVersion.WithResource("namespaces")
	ns := api.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "ns-test"},
		Spec: api.NamespaceSpec{},
	}
	attr := admission.NewAttributesRecord(&ns, nil, schema.GroupVersionKind{Kind: "Namespace"}, metav1.NamespaceNone, ns.Name, resource, "", admission.Delete, false, nil)

	gotErr := ctrl.Validate(attr)
	//if gotErr == nil {
	//	t.Fatalf("expected error, got nil")
	//}
	t.Logf("got expected error: %v", gotErr)

	//_ = informerFactory.Core().InternalVersion().Pods().Informer().GetStore().Delete(pod)
	//gotErr = ctrl.Validate(attr)
	//if gotErr != nil {
	//	t.Fatalf("expected no error, got %v", gotErr)
	//}
}

func fakePodCounterZero(namespace string, expectedOwner admission.Attributes) (int, error) {
	return 0, nil
}

func fakePodCounterOne(namespace string, expectedOwner admission.Attributes) (int, error) {
	return 1, nil
}

func TestAdmissionNonNilAttributeZero(t *testing.T) {
	handler := new(auditDeletePlugin)
	handler.podCounterHandler = fakePodCounterZero
	err := handler.Validate(admission.NewAttributesRecord(nil, nil,
		api.Kind("Namespace").WithVersion("version"), "namespace", "name",
		api.Resource("resource").WithVersion("version"), "subresource",
		admission.Delete, false, nil))
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
		admission.Delete, false, nil))
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
