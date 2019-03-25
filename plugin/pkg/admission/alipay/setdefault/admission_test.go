package setdefault

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	kubeadmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	"k8s.io/kubernetes/plugin/pkg/admission/alipay/util"
	"k8s.io/utils/pointer"
)

func TestRegister(t *testing.T) {
	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()
	if len(registered) == 1 && registered[0] == PluginName {
		return
	} else {
		t.Errorf("Register failed")
	}
}

func TestHandles(t *testing.T) {
	for op, shouldHandle := range map[admission.Operation]bool{
		admission.Create:  true,
		admission.Update:  false,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipaySetDefault()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

func TestValidate(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		initfunc func() *core.Pod
		validate func(*core.Pod)
		err      error
	}{
		{
			name: "validate cgroup parent",
			initfunc: func() *core.Pod {
				pod := newPod()
				allocSpec := sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: defaultCGroupParent}},
						{HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: defaultCGroupParent}},
					},
				}
				data, _ := json.Marshal(&allocSpec)
				pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
				return pod
			},
		},
		{
			name: "set cgroup parent for one of containers",
			initfunc: func() *core.Pod {
				pod := newPod()
				allocSpec := sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{Name: "javaweb", HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: bestEffortCGroupName}},
					},
				}
				data, _ := json.Marshal(&allocSpec)
				pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
				return pod
			},
			err: nil,
		},
		{
			name: "cgroup parent invalid",
			initfunc: func() *core.Pod {
				pod := newPod()
				allocSpec := sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{Name: "javaweb", HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: "unknownapp"}},
						{Name: "sidecar", HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: defaultCGroupParent}},
					},
				}
				data, _ := json.Marshal(&allocSpec)
				pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
				return pod
			},
			err: fmt.Errorf("pods \"test-setdefault-pod\" is forbidden: %s container javaweb cgroup parent invalid, choices: [/sigma /sigma-stream /system-agent]",
				sigmak8sapi.AnnotationPodAllocSpec),
		},
	} {
		t.Logf("testcase [%s]", test.name)

		handler, f, err := newHandlerForTest(
			fake.NewSimpleClientset(
				&core.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: customCGroupParentNamespace,
						Name:      customCGroupParentName,
					},
					Data: map[string]string{
						customCGroupParentDataKey: "/sigma;/sigma-stream;/system-agent",
					},
				},
			),
		)
		if err != nil {
			t.Errorf("unexpected error initializing handler: %v", err)
		}
		f.Start(stopCh)

		pod := test.initfunc()
		a := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		err = handler.Validate(a)
		if test.err != nil {
			assert.Equal(t, test.err.Error(), err.Error())
		} else {
			assert.Equal(t, nil, err)
		}

		if test.validate != nil {
			test.validate(pod)
		}
	}
}

func TestAdmit(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		initfunc func() *core.Pod
		validate func(*core.Pod)
		err      error
	}{
		{
			name:     "default cgroup parent",
			initfunc: newPod,
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				for i := 0; i < 2; i++ {
					assert.Equal(t, defaultCGroupParent, allocSpec.Containers[i].HostConfig.CgroupParent)
					assert.Equal(t, containerSNEnvName, pod.Spec.Containers[i].Env[0].Name)
					assert.Equal(t, "sn1", pod.Spec.Containers[i].Env[0].Value)
					assert.Equal(t, cpuBvtWarpNsLatencySensitve, allocSpec.Containers[i].HostConfig.CPUBvtWarpNs)
				}
				assert.Equal(t, int64(netPriorityLatencySensitive), podNetPriority(pod))
			},
		},
		{
			name: "set cgroup parent for one of containers",
			initfunc: func() *core.Pod {
				pod := newPod()
				allocSpec := sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: bestEffortCGroupName}},
					},
				}
				data, _ := json.Marshal(&allocSpec)
				pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
				return pod
			},
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				for i := 0; i < 2; i++ {
					assert.Equal(t, pod.Spec.Containers[i].Env[0].Name, containerSNEnvName)
					assert.Equal(t, pod.Spec.Containers[i].Env[0].Value, "sn1")
				}
				assert.Equal(t, allocSpec.Containers[0].HostConfig.CgroupParent, bestEffortCGroupName)
				assert.Equal(t, allocSpec.Containers[0].HostConfig.CPUBvtWarpNs, cpuBvtWarpNsBatchJobs)
				assert.Equal(t, allocSpec.Containers[1].HostConfig.CgroupParent, defaultCGroupParent)
				assert.Equal(t, allocSpec.Containers[1].HostConfig.CPUBvtWarpNs, cpuBvtWarpNsLatencySensitve)
				assert.Equal(t, int64(netPriorityBatchJobs), podNetPriority(pod))
			},
		},
		{
			name: "env SN exists",
			initfunc: func() *core.Pod {
				pod := newPod()
				pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, core.EnvVar{Name: containerSNEnvName, Value: "sn2"})
				return pod
			},
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				for i := 0; i < 2; i++ {
					assert.Equal(t, allocSpec.Containers[i].HostConfig.CgroupParent, defaultCGroupParent)
					assert.Equal(t, allocSpec.Containers[i].HostConfig.CPUBvtWarpNs, cpuBvtWarpNsLatencySensitve)
					assert.Len(t, pod.Spec.Containers[i].Env, 1)
					assert.Equal(t, pod.Spec.Containers[i].Env[0].Name, containerSNEnvName)
				}
				assert.Equal(t, int64(netPriorityLatencySensitive), podNetPriority(pod))
				assert.Equal(t, pod.Spec.Containers[0].Env[0].Value, "sn2")
				assert.Equal(t, pod.Spec.Containers[1].Env[0].Value, "sn1")
			},
		},
		{
			name: "set guaranteed pod oom_score_adj",
			initfunc: func() *core.Pod {
				guaranteedPodOOMScoreAdj = pointer.Int64Ptr(-1)
				return newPod()
			},
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				assert.Equal(t, allocSpec.Containers[0].HostConfig.OomScoreAdj, int64(-1))
				assert.Equal(t, allocSpec.Containers[1].HostConfig.OomScoreAdj, int64(-1))
			},
		},
		{
			name: "don't set non-guaranteed pod oom_score_adj",
			initfunc: func() *core.Pod {
				guaranteedPodOOMScoreAdj = pointer.Int64Ptr(-1)
				pod := newPod()
				pod.Spec.Containers[0].Resources.Limits[core.ResourceMemory] = resource.MustParse("3Gi")
				return pod
			},
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				assert.Equal(t, allocSpec.Containers[0].HostConfig.OomScoreAdj, int64(0))
				assert.Equal(t, allocSpec.Containers[1].HostConfig.OomScoreAdj, int64(0))
			},
		},
		{
			name: "system-agent cgroup parent",
			initfunc: func() *core.Pod {
				pod := newPod()
				allocSpec := sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{Name: "javaweb", HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: systemAgentCGroupParent}},
						{Name: "sidecar", HostConfig: sigmak8sapi.HostConfigInfo{CgroupParent: systemAgentCGroupParent}},
					},
				}
				data, _ := json.Marshal(&allocSpec)
				pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
				return pod
			},
			validate: func(pod *core.Pod) {
				allocSpec, err := util.PodAllocSpec(pod)
				assert.Nil(t, err)
				for i := 0; i < 2; i++ {
					assert.Equal(t, cpuBvtWarpUnknown, allocSpec.Containers[i].HostConfig.CPUBvtWarpNs)
				}
				assert.Equal(t, int64(netPriorityBatchJobs), podNetPriority(pod))
			},
		},
	} {
		t.Logf("testcase [%s]", test.name)

		handler, f, err := newHandlerForTest(fake.NewSimpleClientset())
		if err != nil {
			t.Errorf("unexpected error initializing handler: %v", err)
		}
		f.Start(stopCh)

		guaranteedPodOOMScoreAdj = new(int64)
		pod := test.initfunc()
		a := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		err = handler.Admit(a)
		assert.Equal(t, test.err, err)

		if test.validate != nil {
			test.validate(pod)
		}
	}
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Labels:      map[string]string{sigmak8sapi.LabelPodSn: "sn1"},
			Annotations: map[string]string{},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "javaweb",
					Image: "pause:2.0",
					Resources: core.ResourceRequirements{
						Limits: core.ResourceList{
							core.ResourceCPU:    resource.MustParse("1"),
							core.ResourceMemory: resource.MustParse("1Gi"),
						},
						Requests: core.ResourceList{
							core.ResourceCPU:    resource.MustParse("1"),
							core.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
				{
					Name:  "sidecar",
					Image: "pause:2.0",
					Resources: core.ResourceRequirements{
						Limits: core.ResourceList{
							core.ResourceCPU:    resource.MustParse("1"),
							core.ResourceMemory: resource.MustParse("1Gi"),
						},
						Requests: core.ResourceList{
							core.ResourceCPU:    resource.MustParse("1"),
							core.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
}

func newHandlerForTest(c internalclientset.Interface) (*AlipaySetDefault, internalversion.SharedInformerFactory, error) {
	f := internalversion.NewSharedInformerFactory(c, 5*time.Minute)
	handler := NewAlipaySetDefault()
	pluginInitializer := kubeadmission.NewPluginInitializer(c, f, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err := admission.ValidateInitialization(handler)
	return handler, f, err
}

func TestAdmitOtherResources(t *testing.T) {
	pod := newPod()

	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-pod resource",
			kind:     "Foo",
			resource: "foos",
			object:   pod,
		},
		{
			name:        "pod subresource",
			kind:        "Pod",
			resource:    "pods",
			subresource: "eviction",
			object:      pod,
		},
		{
			name:        "non-pod object",
			kind:        "Pod",
			resource:    "pods",
			object:      &core.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := NewAlipaySetDefault()

		err := handler.Admit(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), pod.Namespace, pod.Name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
	}
}

func TestValidateOtherResources(t *testing.T) {
	pod := newPod()

	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-pod resource",
			kind:     "Foo",
			resource: "foos",
			object:   pod,
		},
		{
			name:        "pod subresource",
			kind:        "Pod",
			resource:    "pods",
			subresource: "eviction",
			object:      pod,
		},
		{
			name:     "non-pod object",
			kind:     "Service",
			resource: "services",
			object:   &core.Service{},
		},
	}

	for _, tc := range tests {
		handler := NewAlipaySetDefault()

		err := handler.Validate(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), pod.Namespace, pod.Name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
	}
}
