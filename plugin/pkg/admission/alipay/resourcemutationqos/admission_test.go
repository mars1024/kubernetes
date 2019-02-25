package resourcemutationqos

import (
	"encoding/json"
	"fmt"
	"testing"

	log "github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

func TestResourceMutationQOS(t *testing.T) {
	testcases := []struct {
		pod         *core.Pod
		qosType     sigmak8sapi.SigmaQOSClass
		description string
	}{
		{
			pod:         newQosPod(sigmak8sapi.SigmaQOSGuaranteed),
			qosType:     sigmak8sapi.SigmaQOSGuaranteed,
			description: "cpushare pod without label should be added burstable label",
		},
		{
			pod:         newBurstablePodWithLabel(),
			qosType:     sigmak8sapi.SigmaQOSBurstable,
			description: "cpushare pod with label should be kept as the same",
		},
		{
			pod:         newBurstablePodWithoutLabel(),
			qosType:     sigmak8sapi.SigmaQOSBurstable,
			description: "cpushare pod without label should be added burstable label",
		},
		{
			pod:         newQosPod(sigmak8sapi.SigmaQOSBestEffort),
			qosType:     sigmak8sapi.SigmaQOSBestEffort,
			description: "best effort pod should have sigma best-effort label",
		},
	}

	for i, tcase := range testcases {
		pod := tcase.pod
		attr := admission.NewAttributesRecord(
			pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
			core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)

		handler := newAlipayResourceMutationQOS()
		err := handler.Admit(attr)
		assert.Nil(t, err)

		log.Infof("running case: %d", i)

		assert.Equal(t, pod.Labels[sigmak8sapi.LabelPodQOSClass], string(tcase.qosType),
			fmt.Sprintf("sigma %s pod should have the correct label after mutation", string(tcase.qosType)))
	}
}

// newBurstablePodWithLabel create a burstable pod, already with `sigmaBurstable` label set.
func newBurstablePodWithLabel() *core.Pod {
	pod := newQosPod(sigmak8sapi.SigmaQOSBurstable)
	pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSBurstable)

	return pod
}

func newBurstablePodWithoutLabel() *core.Pod {
	return newQosPod(sigmak8sapi.SigmaQOSBurstable)
}

func newQosPod(qos sigmak8sapi.SigmaQOSClass) *core.Pod {
	pod := newPodWithResource(1000, 2000)

	switch qos {
	case sigmak8sapi.SigmaQOSBestEffort:
		pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSBestEffort)
	case sigmak8sapi.SigmaQOSGuaranteed:
		updateGuaranteedPodAllocSpec(pod)
	}

	return pod
}

func updateGuaranteedPodAllocSpec(pod *core.Pod) error {
	allocSpec := sigmak8sapi.AllocSpec{}
	err := json.Unmarshal([]byte(pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]), &allocSpec)
	if err != nil {
		return fmt.Errorf("can not unmarshal pod allocSpec annotation: %v", err)
	}

	for i := range allocSpec.Containers {
		allocSpec.Containers[i].Resource.CPU.CPUSet = &sigmak8sapi.CPUSetSpec{
			SpreadStrategy: sigmak8sapi.SpreadStrategySameCoreFirst,
			CPUIDs:         []int{},
		}
	}

	data, _ := json.Marshal(allocSpec)
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	return nil
}

func newPodWithResource(cpuRequest, cpuLimit int64) *core.Pod {
	pod := newPod()
	allocSpec := sigmak8sapi.AllocSpec{}
	for i, c := range pod.Spec.Containers {
		pod.Spec.Containers[i].Resources.Limits = map[core.ResourceName]resource.Quantity{}
		pod.Spec.Containers[i].Resources.Requests = map[core.ResourceName]resource.Quantity{}

		pod.Spec.Containers[i].Resources.Limits[core.ResourceCPU] = *resource.NewMilliQuantity(cpuLimit, resource.DecimalSI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceCPU] = *resource.NewMilliQuantity(cpuRequest, resource.DecimalSI)
		allocSpec.Containers = append(allocSpec.Containers, newAllocSpecContainer(c.Name))
	}

	data, _ := json.Marshal(&allocSpec)
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	return pod
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-resource-mutation-burstable-pod",
			Namespace:   metav1.NamespaceDefault,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "container-1",
					Image: "image:1.0",
				},
				{
					Name:  "container-2",
					Image: "image:2.0",
				},
			},
		},
	}
}

func newAllocSpecContainer(name string) sigmak8sapi.Container {
	return sigmak8sapi.Container{
		Name: name,
		Resource: sigmak8sapi.ResourceRequirements{
			CPU: sigmak8sapi.CPUSpec{},
			// GPU.ShareMode is validated in admission controller 'sigmascheduling'
			GPU: sigmak8sapi.GPUSpec{ShareMode: sigmak8sapi.GPUShareModeExclusive},
		},
	}
}
