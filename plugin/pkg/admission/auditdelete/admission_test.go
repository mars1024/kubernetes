package auditdelete

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/apps"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"
)

func newPod(namespace string, ownerRef *metav1.OwnerReference) *api.Pod {
	name := fmt.Sprintf("foo-%d", time.Now().UnixNano())
	p := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: api.PodSpec{},
	}
	if ownerRef != nil {
		ownerRef.Controller = new(bool)
		*ownerRef.Controller = true
		p.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	}
	return p
}

type testObject interface {
	metav1.Object
	runtime.Object
}

func TestDeleteValidation(t *testing.T) {
	testCases := []struct{
		name string
		pod *api.Pod
		object testObject
		kind string
		expectedErr bool
	}{
		{
			name: "delete namespace succeed 1",
			pod: nil,
			object: &api.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "ns-test", Namespace: "ns-test"},
				Spec: api.NamespaceSpec{},
			},
			kind: "Namespace",
			expectedErr: false,
		},
		{
			name: "delete namespace succeed 2",
			pod: newPod("ns-test1", nil),
			object: &api.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "ns-test", Namespace: "ns-test"},
				Spec: api.NamespaceSpec{},
			},
			kind: "Namespace",
			expectedErr: false,
		},
		{
			name: "delete namespace failed",
			pod: newPod("ns-test", nil),
			object: &api.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "ns-test", Namespace: "ns-test"},
				Spec: api.NamespaceSpec{},
			},
			kind: "Namespace",
			expectedErr: true,
		},
		{
			name: "delete statefulset succeed 1",
			pod: nil,
			object: &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "ss-test", Namespace: "ss-ns", Labels: map[string]string{modeForStatefulSet: "sigma"}},
				Spec: apps.StatefulSetSpec{},
			},
			kind: "StatefulSet",
			expectedErr: false,
		},
		{
			name: "delete statefulset succeed 2",
			pod: newPod("ss-ns", nil),
			object: &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "ss-test", Namespace: "ss-ns", Labels: map[string]string{modeForStatefulSet: "sigma"}},
				Spec: apps.StatefulSetSpec{},
			},
			kind: "StatefulSet",
			expectedErr: false,
		},
		{
			name: "delete statefulset succeed 2",
			pod: newPod("ss-ns", &metav1.OwnerReference{Name: "ss-test", Kind: "StatefulSet", UID: "uid01"}),
			object: &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "ss-test", Namespace: "ss-ns", UID: "uid01"},
				Spec: apps.StatefulSetSpec{},
			},
			kind: "StatefulSet",
			expectedErr: false,
		},
		{
			name: "delete statefulset failed 1",
			pod: newPod("ss-ns", &metav1.OwnerReference{Name: "ss-test", Kind: "StatefulSet", UID: "uid01"}),
			object: &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "ss-test", Namespace: "ss-ns", Labels: map[string]string{modeForStatefulSet: "sigma"}, UID: "uid01"},
				Spec: apps.StatefulSetSpec{},
			},
			kind: "StatefulSet",
			expectedErr: true,
		},
	}

	ctrl := newPlugin()
	informerFactory := informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	ctrl.SetInternalKubeInformerFactory(informerFactory)
	err := ctrl.ValidateInitialization()
	if err != nil {
		t.Fatalf("neither pv lister nor storageclass lister can be nil")
	}

	for _, test := range testCases {
		_ = informerFactory.Core().InternalVersion().Pods().Informer().GetIndexer().Replace(make([]interface{}, 0), "0")
		if test.pod != nil {
			_ = informerFactory.Core().InternalVersion().Pods().Informer().GetIndexer().Add(test.pod)
		}
		attr := admission.NewAttributesRecord(
			test.object, nil, schema.GroupVersionKind{Kind: test.kind},
			test.object.GetNamespace(), test.object.GetName(), api.SchemeGroupVersion.WithResource(""), "", admission.Delete, false, nil)
		err := ctrl.Validate(attr)
		if test.expectedErr != (err != nil) {
			t.Errorf("%s expected err %v, got %v", test.name, test.expectedErr, err)
		}
	}
}
