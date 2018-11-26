package kubelet

import (
	"k8s.io/api/core/v1"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPodHostNameTemplate(t *testing.T) {
	testCase := []struct {
		name                   string
		annotationValue        string
		expectError            bool
		withWrongAnnotationKey bool
	}{
		{
			name:        "annotation is empty, should error",
			expectError: true,
		},
		{
			name:            "every thing is ok",
			expectError:     false,
			annotationValue: "123",
		},
		{
			name:                   "with wrong annotation key, so exist error",
			expectError:            true,
			annotationValue:        "123",
			withWrongAnnotationKey: true,
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}
			if cs.annotationValue != "" {
				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationPodHostNameTemplate: cs.annotationValue,
				}
				if cs.withWrongAnnotationKey {
					pod.Annotations["testKey"] = "testValue"
					delete(pod.Annotations, sigmak8sapi.AnnotationPodHostNameTemplate)
				}
			}
			hostTemplate, err := GetPodHostNameTemplate(pod)
			if cs.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, hostTemplate, cs.annotationValue)
		})
	}
}

func TestGeneratePodHostNameAndDomainByHostNameTemplate(t *testing.T) {
	testCase := []struct {
		name             string
		hostTemplate     string
		podIP            string
		expectHostName   string
		expectHostDomain string
		expectSuccess    bool
		expectErr        bool
	}{
		{
			name:             "annotation is empty, so host name and domain is empty",
			expectHostDomain: "",
			expectHostName:   "",
			expectSuccess:    false,
			expectErr:        true,
		},
		{
			name:             "podIp is invalid, so host name and domain is empty",
			hostTemplate:     "app-test",
			expectHostDomain: "",
			expectHostName:   "",
			podIP:            "1.1.1",
			expectSuccess:    false,
			expectErr:        true,
		},
		{
			name:             "template need not replace, domain is empty",
			hostTemplate:     "app-test",
			podIP:            "1.1.1.1",
			expectHostDomain: "",
			expectHostName:   "app-test",
			expectSuccess:    true,
			expectErr:        false,
		},
		{
			name:             "template need not replace, every thing is ok",
			hostTemplate:     "app-test.zbyun.et2",
			podIP:            "1.1.1.1",
			expectHostDomain: "zbyun.et2",
			expectHostName:   "app-test",
			expectSuccess:    true,
			expectErr:        false,
		},
		{
			name:             "template need  replace, domain is empty",
			hostTemplate:     "app-test{{.IpAddress}}",
			podIP:            "1.1.1.1",
			expectHostDomain: "",
			expectHostName:   "app-test001001001001",
			expectSuccess:    true,
			expectErr:        false,
		},
		{
			name:             "template need  replace, every thing is ok",
			hostTemplate:     "app-test{{.IpAddress}}.zbyun.et2",
			podIP:            "1.1.1.1",
			expectHostDomain: "zbyun.et2",
			expectHostName:   "app-test001001001001",
			expectSuccess:    true,
			expectErr:        false,
		},
		{
			name:             "template need  replace, but hostname is not valid",
			hostTemplate:     "APP-test{{.IpAddress}}.zbyun.et2",
			podIP:            "1.1.1.1",
			expectHostDomain: "",
			expectHostName:   "",
			expectSuccess:    false,
			expectErr:        true,
		},
		{
			name: "template need replace, but hostname too long",
			hostTemplate: "app-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp" +
				"-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-testapp-test{{.IpAddress}}.zbyun.et2",
			podIP:            "1.1.1.1",
			expectHostDomain: "",
			expectHostName:   "",
			expectSuccess:    false,
			expectErr:        true,
		},
		{
			name: "template need replace, but hostdomain too long",
			hostTemplate: "app-test{{.IpAddress}}.zbyun.et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2e" +
				"t2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2et2",
			podIP:            "1.1.1.1",
			expectHostDomain: "",
			expectHostName:   "",
			expectSuccess:    false,
			expectErr:        true,
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}

			pod.Annotations = map[string]string{
				sigmak8sapi.AnnotationPodHostNameTemplate: cs.hostTemplate,
			}
			hostname, hostDomain, success, err := GeneratePodHostNameAndDomainByHostNameTemplate(pod, cs.podIP)
			if cs.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, cs.expectHostName, hostname)
			assert.Equal(t, cs.expectHostDomain, hostDomain)
			assert.Equal(t, cs.expectSuccess, success)
		})
	}
}

func TestParseIPToString(t *testing.T) {
	testCase := []struct {
		name      string
		podIP     string
		expectIP  string
		expectErr bool
	}{
		{
			name:      "everything is ok",
			podIP:     "1.1.1.1",
			expectIP:  "001001001001",
			expectErr: false,
		},
		{
			name:      "invalid ip",
			podIP:     "1.1.1",
			expectErr: true,
		},
		{
			name:      "everything is ok",
			podIP:     "127.0.0.0",
			expectIP:  "127000000000",
			expectErr: false,
		},
	}
	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			ipAddress, err := ParseIPToString(cs.podIP)
			if cs.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, cs.expectIP, ipAddress)
		})
	}
}

func TestPodHaveCNIAllocatedFinalizer(t *testing.T) {
	testCase := []struct {
		name          string
		finalizer     []string
		haveFinalizer bool
	}{
		{
			name:          "everything is ok",
			finalizer:     []string{"test"},
			haveFinalizer: false,
		},
		{
			name:          "invalid ip",
			finalizer:     []string{},
			haveFinalizer: false,
		},
		{
			name:          "everything is ok",
			finalizer:     []string{sigmak8sapi.FinalizerPodCNIAllocated},
			haveFinalizer: true,
		},
	}
	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: cs.finalizer,
				},
			}
			have := PodHaveCNIAllocatedFinalizer(pod)
			assert.Equal(t, have, cs.haveFinalizer)
		})
	}
}

func TestFilterCgroupShouldNotCleanPods(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kubelet := testKubelet.kubelet
	pods := newTestPods(5)
	now := metav1.NewTime(time.Now())
	pods[0].Status.Phase = v1.PodFailed
	pods[1].Status.Phase = v1.PodSucceeded
	pods[1].ObjectMeta.Finalizers = []string{sigmak8sapi.FinalizerPodCNIAllocated}
	// The pod is terminating, should not filter out.
	pods[2].Status.Phase = v1.PodRunning
	pods[2].DeletionTimestamp = &now
	pods[2].Status.ContainerStatuses = []v1.ContainerStatus{
		{State: v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: now,
			},
		}},
	}
	pods[3].Status.Phase = v1.PodPending
	pods[4].Status.Phase = v1.PodRunning

	expected := []*v1.Pod{pods[1], pods[2], pods[3], pods[4]}
	kubelet.podManager.SetPods(pods)

	actual := kubelet.filterCgroupShouldPreservePods(pods)
	assert.Equal(t, expected, actual)
}
