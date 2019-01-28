package resourcemutationburstable

import (
	"encoding/json"
	"fmt"
	"io"

	log "github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/admission"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins.
	PluginName = "AlipayResourceMutationBurstable"
)

var (
	_ admission.MutationInterface = &AlipayResourceMutationBurstable{}
)

// Register is used to register current plugin to APIServer.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(config io.Reader) (admission.Interface, error) {
			return newAlipayResourceMutationBurstable(), nil
		})
}

// AlipayResourceMutationBurstable is the main struct to mutate pod resource setting.
type AlipayResourceMutationBurstable struct {
	*admission.Handler
}

func newAlipayResourceMutationBurstable() *AlipayResourceMutationBurstable {
	return &AlipayResourceMutationBurstable{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit adds pod `sigma.ali/qos` label for sigmaburstable pod if it's not alreay set.
func (a *AlipayResourceMutationBurstable) Admit(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod, ok := attr.GetObject().(*v1.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	if err = mutatePodResource(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func isCPUSetPod(pod *v1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}

	allocSpecStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	// if pod does not have allocSpec annotation, it is a cpushare pod.
	if !exists {
		return false
	}
	// Unmarshal allocSpecStr into struct.
	allocSpec := sigmak8sapi.AllocSpec{}
	if err := json.Unmarshal([]byte(allocSpecStr), &allocSpec); err != nil {
		// In case of malformatted alllospec, we explictly log the error, and regrad it as non-cpuset,
		// thus will not mutate its resource.
		log.Errorf("Invalid data in pod annotation from %s/%s: %v", pod.Namespace, pod.Name, err)
		return false
	}

	for _, c := range allocSpec.Containers {
		if c.Resource.CPU.CPUSet != nil {
			return true
		}
	}
	return false
}

func isCPUSharePod(pod *v1.Pod) bool {
	return !isCPUSetPod(pod)
}

func mutatePodResource(pod *v1.Pod) error {
	// pod alreay has expected label
	if sigmak8sapi.GetPodQOSClass(pod) == sigmak8sapi.SigmaQOSBurstable {
		return nil
	}

	if isCPUSharePod(pod) && !isSigmaBestEffortPod(pod) {
		pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSBurstable)
	}

	return nil
}

// isSigmaBestEffortPod
// return true if podQOSClass == SigmaQOSBestEffort
func isSigmaBestEffortPod(pod *v1.Pod) bool {
	return sigmak8sapi.GetPodQOSClass(pod) == sigmak8sapi.SigmaQOSBestEffort
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != v1.Resource("pods") {
		return true
	}

	if a.GetSubresource() != "" {
		return true
	}

	_, ok := a.GetObject().(*v1.Pod)
	if !ok {
		log.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() != admission.Create {
		// only admit created pod.
		return true
	}

	return false
}
