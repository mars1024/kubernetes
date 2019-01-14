package kubelet

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	containertest "k8s.io/kubernetes/pkg/kubelet/container/testing"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
)

func TestGetRunningPodByUID(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kubelet := testKubelet.kubelet
	fakeRuntime := testKubelet.fakeRuntime
	fakeRuntime.PodList = []*containertest.FakePod{
		{Pod: &kubecontainer.Pod{
			ID:        "fake-uid-1",
			Name:      "fake-pod-name1",
			Namespace: "fake-pod-namespace1",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
		{Pod: &kubecontainer.Pod{
			ID:        "fake-uid-2",
			Name:      "fake-pod-name2",
			Namespace: "fake-pod-namespace2",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
	}

	for caseName, testCase := range map[string]struct {
		uid                string
		isValid            bool
		expectPodName      string
		expectPodNamespace string
	}{
		"get pod of fake-uid-1": {
			uid:                "fake-uid-1",
			isValid:            true,
			expectPodName:      "fake-pod-name1",
			expectPodNamespace: "fake-pod-namespace1",
		},
		"get pod of fake-uid-2": {
			uid:                "fake-uid-2",
			isValid:            true,
			expectPodName:      "fake-pod-name2",
			expectPodNamespace: "fake-pod-namespace2",
		},
		"get pod of not-exists": {
			uid:     "fake-uid-3",
			isValid: false,
		},
	} {
		kubePod := kubelet.getRunningPodByUID(testCase.uid)
		if !testCase.isValid {
			if kubePod != nil {
				t.Errorf("test case %s, expected kubePod is nil but got: %v", caseName, kubePod)
			}
			continue
		}
		if kubePod.Name != testCase.expectPodName {
			t.Errorf("test case %s, expected pod name is %s, but got %s", caseName, testCase.expectPodName, kubePod.Name)
		}
		if kubePod.Namespace != testCase.expectPodNamespace {
			t.Errorf("test case %s, expected pod namespace is %s, but got %s", caseName, testCase.expectPodNamespace, kubePod.Namespace)
		}
	}
}

func TestGetRunningPodByName(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kubelet := testKubelet.kubelet
	fakeRuntime := testKubelet.fakeRuntime
	fakeRuntime.PodList = []*containertest.FakePod{
		{Pod: &kubecontainer.Pod{
			ID:        "fake-uid-1",
			Name:      "fake-pod-name1",
			Namespace: "fake-pod-namespace1",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
		{Pod: &kubecontainer.Pod{
			ID:        "fake-uid-2",
			Name:      "fake-pod-name2",
			Namespace: "fake-pod-namespace2",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
	}

	for caseName, testCase := range map[string]struct {
		name        string
		namespace   string
		expectedUID string
		isValid     bool
	}{
		"get pod of fake-pod-name1": {
			name:        "fake-pod-name1",
			namespace:   "fake-pod-namespace1",
			expectedUID: "fake-uid-1",
			isValid:     true,
		},
		"get pod of fake-pod-name2": {
			name:        "fake-pod-name2",
			namespace:   "fake-pod-namespace2",
			expectedUID: "fake-uid-2",
			isValid:     true,
		},
		"get pod of not-exists": {
			name:      "fake-pod-name3",
			namespace: "fake-pod-namespace2",
			isValid:   false,
		},
	} {
		kubePod := kubelet.getRunningPodByName(testCase.name, testCase.namespace)
		if !testCase.isValid {
			if kubePod != nil {
				t.Errorf("test case %s, expected kubePod is nil but got: %v", caseName, kubePod)
			}
			continue
		}
		if string(kubePod.ID) != testCase.expectedUID {
			t.Errorf("test case %s, expected pod uid is %s, but got %s", caseName, testCase.expectedUID, kubePod.ID)
		}
	}
}

func TestIsPodFromAPIServerSource(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	testCases := []struct {
		name         string
		pod          *v1.Pod
		expectResult bool
	}{
		{
			name: "pod is from apiserver",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						kubetypes.ConfigSourceAnnotationKey: kubetypes.ApiserverSource,
					},
				},
			},
			expectResult: true,
		},
		{
			name: "pod is mirror pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						kubetypes.ConfigSourceAnnotationKey: kubetypes.ApiserverSource,
						kubetypes.ConfigMirrorAnnotationKey: "123456",
					},
				},
			},
			expectResult: false,
		},
		{
			name: "pod is from file",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						kubetypes.ConfigSourceAnnotationKey: kubetypes.FileSource,
					},
				},
			},
			expectResult: false,
		},
		{
			name: "pod is from http",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						kubetypes.ConfigSourceAnnotationKey: kubetypes.HTTPSource,
					},
				},
			},
			expectResult: false,
		},
	}
	for _, ts := range testCases {
		t.Run(ts.name, func(t *testing.T) {
			result := isPodFromAPIServerSource(ts.pod)
			assert.Equal(t, result, ts.expectResult)
		})
	}
}

func TestSyncDanglingPods(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kubelet := testKubelet.kubelet
	kubelet.nodeName = "host1"
	kubelet.kubeClient = fake.NewSimpleClientset()

	// Prepare node.
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "host1",
		},
	}
	danglingPods := []sigmak8sapi.DanglingPod{
		sigmak8sapi.DanglingPod{
			Name:      "pod1",
			Namespace: "namespace1",
			UID:       "uid1",
		},
		sigmak8sapi.DanglingPod{
			Name:      "pod2",
			Namespace: "namespace2",
			UID:       "uid2",
		},
		sigmak8sapi.DanglingPod{
			Name:         "pod3",
			Namespace:    "namespace3",
			UID:          "uid3",
			SafeToRemove: true,
		},
		sigmak8sapi.DanglingPod{
			Name:         "pod4",
			Namespace:    "namespace4",
			UID:          "uid4",
			SafeToRemove: true,
		},
		sigmak8sapi.DanglingPod{
			Name:         "pod5",
			Namespace:    "namespace5",
			UID:          "uid5",
			SafeToRemove: true,
		},
	}

	danglingPodsBytes, err := json.Marshal(danglingPods)
	assert.Equal(t, err, nil)
	node.Annotations = map[string]string{
		sigmak8sapi.AnnotationDanglingPods: string(danglingPodsBytes),
	}

	_, err = kubelet.kubeClient.CoreV1().Nodes().Create(node)
	assert.Equal(t, err, nil)

	pod5 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod5",
			Namespace: "namespace5",
			UID:       "uid5",
		},
	}

	_, err = kubelet.kubeClient.CoreV1().Pods("namespace5").Create(pod5)
	assert.Equal(t, err, nil)

	// Prepare runtime.
	fakeRuntime := testKubelet.fakeRuntime
	// pod1 and pod3 are not in runtime.
	fakeRuntime.PodList = []*containertest.FakePod{
		{Pod: &kubecontainer.Pod{
			ID:        "uid2",
			Name:      "pod2",
			Namespace: "namespace2",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
		{Pod: &kubecontainer.Pod{
			ID:        "uid4",
			Name:      "pod4",
			Namespace: "namespace4",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
		{Pod: &kubecontainer.Pod{
			ID:        "uid5",
			Name:      "pod5",
			Namespace: "namespace5",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
		{Pod: &kubecontainer.Pod{
			ID:        "uid6",
			Name:      "pod6",
			Namespace: "namespace6",
			Containers: []*kubecontainer.Container{
				{Name: "foo"},
			},
		}},
	}

	// pod2 is in podManager
	pod2 := podWithUIDNameNs("uid2", "name2", "namespace2")
	pod2.Labels = map[string]string{
		sigmak8sapi.LabelPodSn: "sn2",
	}
	pods := []*v1.Pod{pod2}
	kubelet.podManager.SetPods(pods)
	expectDanglingPods := map[string]sigmak8sapi.DanglingPod{
		// pod1: dangplingPod not in runtime,
		// It is not a danglingPod anymore.

		// pod2: dangplingPod in runtime.
		// pod2 can get sn from podManager.
		"pod2": sigmak8sapi.DanglingPod{
			Name:      "pod2",
			Namespace: "namespace2",
			UID:       "uid2",
			Phase:     v1.PodRunning,
			SN:        "sn2",
		},
		// pod3: danglingPod is removed because it is not in runtime already.

		// pod4: danglingPod needs to be removed, but still in runtime.
		// SafeToRemove feild is kept.
		"pod4": sigmak8sapi.DanglingPod{
			Name:         "pod4",
			Namespace:    "namespace4",
			UID:          "uid4",
			Phase:        v1.PodRunning,
			SafeToRemove: true,
		},
		// pod5: danglingPod is in apiserver, is not a danglingPod anymore.

		// pod6: a new danglingPod.
		"pod6": sigmak8sapi.DanglingPod{
			Name:      "pod6",
			Namespace: "namespace6",
			UID:       "uid6",
			Phase:     v1.PodRunning,
		},
	}

	kubelet.SyncDanglingPods()

	actualDanglingPods, err := sigmautil.GetDanglingPods(kubelet.kubeClient, string(kubelet.nodeName))
	assert.Equal(t, err, nil)

	actualDanlingPodsMap := map[string]sigmak8sapi.DanglingPod{}
	for _, actualDanglingPod := range actualDanglingPods {
		actualDanlingPodsMap[actualDanglingPod.Name] = actualDanglingPod
	}

	if !reflect.DeepEqual(expectDanglingPods, actualDanlingPodsMap) {
		t.Errorf("Get wrong danglingPods in case %q: expect: %v, but get: %v", "test for SyncDanglingPods()",
			expectDanglingPods, actualDanlingPodsMap)
	}
}
