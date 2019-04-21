package util

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

const (
	needToPatchLastSpecInAnnotations = "need-to-patch-last-spec-in-annotations-magic-code"
	inplaceTrue                      = "true"
)

func IsInplaceUpdatePod(pod *v1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}

	if state, ok := pod.Annotations[sigmaapi.AnnotationPodInplaceUpdateState]; ok {
		if state == sigmaapi.InplaceUpdateStateCreated {
			return true
		}
	}

	return false
}

func LastSpecFromPod(pod *v1.Pod) *v1.PodSpec {
	spec := &v1.PodSpec{}
	if pod == nil {
		return nil
	}

	specData, ok := pod.Annotations[sigmaapi.AnnotationPodLastSpec]
	if !ok {
		return nil
	}
	if len(specData) == 0 {
		return nil
	}

	if err := json.Unmarshal([]byte(specData), spec); err != nil {
		glog.Errorf("unmarshal last spec from pod[%s] failed: %v", pod.Name, err)
		return nil
	}

	return spec
}

func StoreLastSpecIfNeeded(oldPod, newPod *v1.Pod) {
	state, ok := newPod.Annotations[sigmaapi.AnnotationPodInplaceUpdateState]
	if !ok {
		return
	}

	if state != sigmaapi.InplaceUpdateStateCreated {
		return
	}

	// Compares the old spec received by api-server and last spec get from annotations.
	// If they are equal (we only care about resources), this last spec update will not be patched.
	f := func(spec *v1.PodSpec) *v1.PodSpec {
		s := &v1.PodSpec{}
		s.Containers = make([]v1.Container, len(spec.Containers))
		for idx, container := range spec.Containers {
			s.Containers[idx].Name = container.Name
			s.Containers[idx].Resources = container.Resources
		}
		return s
	}

	needToPatchLastSpec := false
	lastSpec := LastSpecFromPod(newPod)
	if lastSpec == nil {
		needToPatchLastSpec = true
		lastSpec = f(&oldPod.Spec)
	}

	lastSpecCopy, oldPodSpecCopy := f(lastSpec), f(&oldPod.Spec)
	if !reflect.DeepEqual(lastSpecCopy, oldPodSpecCopy) {
		needToPatchLastSpec = true
		lastSpec = oldPodSpecCopy
	}

	lastSpecStr, err := json.Marshal(lastSpec)
	if err != nil {
		glog.Errorf("failed to do json marshal last spec: %v", err)
		return
	}

	if newPod.Annotations == nil {
		newPod.Annotations = make(map[string]string)
	}

	newPod.Annotations[sigmaapi.AnnotationPodLastSpec] = string(lastSpecStr)
	if needToPatchLastSpec {
		newPod.Annotations[needToPatchLastSpecInAnnotations] = inplaceTrue
	}

	glog.V(4).Infof("lastSpecStr: %s", string(lastSpecStr))
}

func IsCPUResourceChanged(oldSpec, newSpec *v1.PodSpec) bool {
	if len(newSpec.Containers) != len(oldSpec.Containers) {
		return true
	}

	for idx, container := range newSpec.Containers {
		for rName, rQuantity := range container.Resources.Requests {
			switch rName {
			case v1.ResourceCPU:
				newMilliCPU := rQuantity.MilliValue()
				if idx < len(oldSpec.Containers) {
					oldContainer := oldSpec.Containers[idx]
					oldRequests := oldContainer.Resources.Requests
					if oldQuantity, ok := oldRequests[v1.ResourceCPU]; ok {
						oldMilliCPU := oldQuantity.MilliValue()
						if newMilliCPU != oldMilliCPU {
							// CPU resource changed, return true.
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func PatchPod(client clientset.Interface, oldPod, newPod *v1.Pod) error {
	// CreatePodPatch
	patch, err := CreatePodPatch(oldPod, newPod)
	if err != nil {
		return err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return fmt.Errorf("no patch for setting pod %v annotations", newPod.Name)
	}
	if _, err := client.CoreV1().Pods(newPod.Namespace).Patch(newPod.Name, apimachinerytypes.StrategicMergePatchType, patch); err != nil {
		return fmt.Errorf("Fail to patch pod %s/%s with %s: %v", newPod.Namespace, newPod.Name, string(patch), err)
	}
	return nil
}
