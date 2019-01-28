package resourcemutationqos

import (
	"encoding/json"
	"fmt"
	"io"

	log "github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins.
	PluginName = "AlipayResourceMutationQOS"
)

var (
	_ admission.MutationInterface = &AlipayResourceMutationQOS{}
)

// Register is used to register current plugin to APIServer.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(config io.Reader) (admission.Interface, error) {
			return newAlipayResourceMutationQOS(), nil
		})
}

// AlipayResourceMutationQOS is the main struct to mutate pod resource setting.
type AlipayResourceMutationQOS struct {
	*admission.Handler
}

func newAlipayResourceMutationQOS() *AlipayResourceMutationQOS {
	return &AlipayResourceMutationQOS{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit adds pod sigma QoS label, thus making sure all pods should have correct
// sigmaQoS label.
// In short, we propose three sigma qos level(which is independent from kubernetes qos):
// 1. SigmaGuaranteed: default mode for pod which has cpuset containers
// 2. SigmaBurstable: for pods with cpushare containers
// 3. SigmaBestEffort: for Job-type pods, this label must be passed in when creating pod
//
// Design doc: https://yuque.antfin-inc.com/sys/sigma3.x/ptavcw#s3ahqe.
func (a *AlipayResourceMutationQOS) Admit(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	if err = mutatePodResource(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func isCPUSetPod(pod *core.Pod) bool {
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

func isCPUSharePod(pod *core.Pod) bool {
	return !isCPUSetPod(pod)
}

func mutatePodResource(pod *core.Pod) error {
	// if pod has SigmaBestEffort qos label, do nothing
	if pod.Labels[sigmak8sapi.LabelPodQOSClass] == string(sigmak8sapi.SigmaQOSBestEffort) {
		return nil
	}

	if isCPUSetPod(pod) {
		// if it is a cpuset pod, make sure it has `sigma.ali/qos: SigmaGuaranteed` label
		if pod.Labels[sigmak8sapi.LabelPodQOSClass] != string(sigmak8sapi.SigmaQOSGuaranteed) {
			pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSGuaranteed)
		}

	} else {
		// this is a cpushare pod, make sure it has `sigma.ali/qos: SigmaBurstable` label
		if pod.Labels[sigmak8sapi.LabelPodQOSClass] != string(sigmak8sapi.SigmaQOSBurstable) {
			pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSBurstable)
		}
	}

	return nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != core.Resource("pods") {
		return true
	}

	if a.GetSubresource() != "" {
		return true
	}

	_, ok := a.GetObject().(*core.Pod)
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
