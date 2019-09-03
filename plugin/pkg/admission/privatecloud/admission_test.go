package privatecloud

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kadmission "k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	namespaceslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/controller"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

// NewTestAdmission provides an admission plugin with test implementations of internal structs.  It uses
// an authorizer that always returns true.
func NewTestAdmission(lister namespaceslisters.NamespaceLister) kadmission.MutationInterface {
	return &privateCloudPlugin{
		Handler: kadmission.NewHandler(kadmission.Create),
		lister:  lister,
	}
}

func TestAdmit(t *testing.T) {
	fakeTenantID := "fake-tenant"
	fakeWorkspaceID := "fake-workspace"
	fakeClusterID := "fake-cluster"
	fakeTenantInfo := multitenancy.NewTenantInfo(fakeTenantID, fakeWorkspaceID, fakeClusterID)

	nsWithEmptyTenantInfo := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns1",
			Annotations: map[string]string{
				"emptyAnnotations": "true",
			},
			Labels: map[string]string{
				"emptyLabels": "true",
			},
		},
	}

	nsWithTenantAnnotations := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns2",
			Annotations: map[string]string{
				"alpha.cloud.alipay.com/tenant-id":    fakeTenantID,
				"alpha.cloud.alipay.com/workspace-id": fakeWorkspaceID,
				"alpha.cloud.alipay.com/cluster-id":   fakeClusterID,
			},
		},
	}

	nsWithTenantLabels := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns3",
			Labels: map[string]string{
				tenantLabel:    fakeTenantID,
				workspaceLabel: fakeWorkspaceID,
				clusterLabel:   fakeClusterID,
			},
		},
	}

	nsWithTenantInfo := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns4",
			Annotations: map[string]string{
				"alpha.cloud.alipay.com/tenant-id":    fakeTenantID,
				"alpha.cloud.alipay.com/workspace-id": fakeWorkspaceID,
				"alpha.cloud.alipay.com/cluster-id":   fakeClusterID,
			},
			Labels: map[string]string{
				tenantLabel:    fakeTenantID,
				workspaceLabel: fakeWorkspaceID,
				clusterLabel:   fakeClusterID,
			},
		},
	}

	nsWithIncompleteTenantInfo := &api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns5",
			Annotations: map[string]string{
				"alpha.cloud.alipay.com/tenant-id":    fakeTenantID,
				"alpha.cloud.alipay.com/workspace-id": fakeWorkspaceID,
			},
		},
	}

	podTemplate := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypod",
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:  "container",
					Image: "fake-image",
				},
			},
		},
	}

	testCases := map[string]struct {
		namespaces      []*api.Namespace
		targetNamespace string
		shouldAccept    bool
		emptyTenant     bool
	}{
		"should admit when missing tenant info from namespace": {
			namespaces:      []*api.Namespace{nsWithEmptyTenantInfo.DeepCopy()},
			targetNamespace: nsWithEmptyTenantInfo.Name,
			shouldAccept:    true,
			emptyTenant:     true,
		},
		"should admit when the namespace has tenant info": {
			namespaces:      []*api.Namespace{nsWithEmptyTenantInfo.DeepCopy(), nsWithTenantInfo.DeepCopy()},
			targetNamespace: nsWithTenantInfo.Name,
			shouldAccept:    true,
		},
		"should admit when the namespace has tenant info in annotations": {
			namespaces:      []*api.Namespace{nsWithEmptyTenantInfo.DeepCopy(), nsWithTenantAnnotations.DeepCopy()},
			targetNamespace: nsWithTenantAnnotations.Name,
			shouldAccept:    true,
		},
		"should admit when the namespace has tenant info in labels": {
			namespaces:      []*api.Namespace{nsWithEmptyTenantInfo.DeepCopy(), nsWithTenantLabels.DeepCopy()},
			targetNamespace: nsWithTenantLabels.Name,
			shouldAccept:    true,
		},
		"should admit when the namespace has incomplete tenant info": {
			namespaces:      []*api.Namespace{nsWithEmptyTenantInfo.DeepCopy(), nsWithIncompleteTenantInfo.DeepCopy()},
			targetNamespace: nsWithIncompleteTenantInfo.Name,
			shouldAccept:    true,
			emptyTenant:     true,
		},
	}

	for name, tc := range testCases {
		t.Log(name)
		pod := podTemplate.DeepCopy()
		pod.Namespace = tc.targetNamespace

		err := admitPod(pod, tc.namespaces)
		if err != nil {
			if tc.shouldAccept {
				t.Fatalf("unexpected error: %v", err)
			}
		} else {
			if !tc.shouldAccept {
				t.Fatalf("should not be admitted")
			} else if !tc.emptyTenant {
				currentTenantInfo, err := multitenancyutil.TransformTenantInfoFromAnnotations(pod.Annotations)
				if err != nil {
					t.Fatalf("unexpected err when getting tenantInfo: %v", err)
				}
				if !reflect.DeepEqual(currentTenantInfo, fakeTenantInfo) {
					t.Errorf("wanted tenantInfo: %+v, got:%+v", currentTenantInfo, fakeTenantInfo)
				}
			}
		}
	}
}

func admitPod(pod *api.Pod, namespaces []*api.Namespace) error {
	informerFactory := informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	store := informerFactory.Core().InternalVersion().Namespaces().Informer().GetStore()
	for _, ns := range namespaces {
		store.Add(ns)
	}

	plugin := NewTestAdmission(informerFactory.Core().InternalVersion().Namespaces().Lister())
	attrs := kadmission.NewAttributesRecord(
		pod,
		nil,
		api.Kind("Pod").WithVersion("version"),
		pod.Namespace,
		pod.Name,
		api.Resource("pods").WithVersion("version"),
		"",
		kadmission.Create,
		false,
		&user.DefaultInfo{},
	)

	err := plugin.Admit(attrs)
	if err != nil {
		return err
	}
	return nil
}

func TestInspectChangedField(t *testing.T) {
	testCases := []struct {
		oldLabels map[string]string
		newLabels map[string]string
		fieldName string
		changed   bool
		deleted   bool
	}{
		{
			oldLabels: map[string]string{"a": "1"},
			newLabels: map[string]string{"a": "2", "b": "2"},
			fieldName: "a",
			changed:   true,
		},
		{
			oldLabels: map[string]string{"a": "1"},
			newLabels: map[string]string{"a": "1", "b": "2"},
			fieldName: "b",
			changed:   true,
		},
		{
			oldLabels: map[string]string{"a": "1"},
			newLabels: map[string]string{"b": "2"},
			fieldName: "a",
			deleted:   true,
		},
	}

	for idx, tc := range testCases {
		changed, deleted := inspectChangedField(tc.oldLabels, tc.newLabels, tc.fieldName)
		if changed != tc.changed {
			t.Errorf("[%d] change expected %v, but got %v", idx, changed, tc.changed)
		}
		if deleted != tc.deleted {
			t.Errorf("[%d] delete expected %v, but got %v", idx, deleted, tc.deleted)
		}
	}
}
