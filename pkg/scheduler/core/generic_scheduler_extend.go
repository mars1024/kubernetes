package core

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/allocators"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/predicates"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/core/equivalence"
	"k8s.io/kubernetes/pkg/scheduler/util"
	"k8s.io/kubernetes/pkg/scheduler/volumebinder"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

const (
	PriorityWeightOverrideAnnotationKey = "scheduling.aks.cafe.sofastack.io/priority-weight-override"
)

func getEffectivePriorityWeight(priorityWeightOverrideMap map[string]int, priorityConfigName string, fallbackValue int) int {
	overriddenWeight, ok := priorityWeightOverrideMap[priorityConfigName]
	if ok {
		return overriddenWeight
	}

	return fallbackValue
}

func getPriorityWeightOverrideMap(pod *v1.Pod) map[string]int {
	priorityWeightOverride := make(map[string]int)
	if pod.Annotations != nil {
		priorityWeightOverrideAnnotation, ok := pod.Annotations[PriorityWeightOverrideAnnotationKey]
		if ok {
			_ = json.Unmarshal([]byte(priorityWeightOverrideAnnotation), &priorityWeightOverride)
		}
	}
	return priorityWeightOverride
}

// Expose this global variable to type assertion
type GenericSchedulerExtend struct {
	genericScheduler
	client clientset.Interface
}

// Allocate allocates the resources for the pod on given host
func (ge *GenericSchedulerExtend) Allocate(pod *v1.Pod, host string) error {
	nodeInfo := ge.cachedNodeInfoMap[host]
	allocator := allocators.NewCPUAllocator(nodeInfo)
	result := allocator.Allocate(pod)
	if len(result) <= 0 {
		glog.V(3).Infof("patch result is %#v for pod %s/%s, skipping patch", result, pod.Namespace, pod.Name)
		return nil
	}
	glog.V(3).Infof("going to patch CPUSet %v for pod %s/%s", result, pod.Namespace, pod.Name)
	patchPod := allocators.MakePodCPUPatch(pod, result)
	cpuAllocator, ok := allocator.(*allocators.CPUAllocator)
	var nodeShareCPUPool cpuset.CPUSet
	var nodeName string
	if ok {
		nodeShareCPUPool = cpuAllocator.NodeCPUSharePool()
		nodeName = host
	}
	err := allocators.DoPatchAll(ge.client, pod, patchPod, nodeShareCPUPool, nodeName)
	if err != nil {
		return allocators.ErrAllocatorFailure(allocator.Name(), err.Error())
	}
	return err
}

func (ge *GenericSchedulerExtend) inplaceUpdateReallocate(pod *v1.Pod) (string, error) {
	glog.V(4).Infof("Attempting to process inplace update pod %s/%s", pod.Namespace, pod.Name)

	// Get last spec from pod annotations.
	lastSpec := util.LastSpecFromPod(pod)
	if lastSpec == nil {
		return "", fmt.Errorf("failed to get last spec from annotation of pod %s/%s", pod.Namespace, pod.Name)
	}

	nodeInfo := ge.cachedNodeInfoMap[pod.Spec.NodeName]

	// Remove the pod with last spec as the resource check will be ok.
	// And we can add this pod back into cache if scheduling pass.
	var err error
	if isAssumed, _ := ge.cache.IsAssumedPod(pod); isAssumed {
		err = ge.cache.ForgetPod(pod)
	} else {
		err = ge.cache.RemovePod(pod)
	}
	if err != nil {
		glog.Errorf("failed to remove pod %s/%s from cache", pod.Namespace, pod.Name)
		return "", err
	}

	// Call PodFitsResources to check resources.
	if fit, predicateFails, _ := predicates.PodCPUSetResourceFit(pod, nil, nodeInfo); !fit {
		return "", fmt.Errorf("failed to fit resource in inplace update processing with predicateFails: %+v", predicateFails)
	}

	// If CPU value is not changed, return immediately.
	if !util.IsCPUResourceChanged(lastSpec, &pod.Spec) {
		glog.V(4).Infof("CPU resource not changed or this is a cpushare request, return immediately")
		ge.cache.FinishBinding(pod)
		return "", nil
	}

	// Call CPUSetAllocation to alloc CPUIDs.
	cpuAlloc := allocators.NewCPUAllocator(nodeInfo)
	cpuAssignments := cpuAlloc.Reallocate(pod)
	ge.cache.FinishBinding(pod)
	return string(allocators.GenAllocSpecAnnotation(pod, cpuAssignments)), nil
}

func (ge *GenericSchedulerExtend) HandleInplacePodUpdate(pod *v1.Pod) error {
	var err error
	var allocSpec string
	assumed := pod.DeepCopy()
	allocSpec, err = ge.inplaceUpdateReallocate(pod)
	if err != nil {
		glog.V(4).Infof("setting inplace update pod %s/%s as failed due to %v", pod.Namespace, pod.Name, err)
		// Set inplace update state to "failed".
		assumed.Annotations[sigmaapi.AnnotationPodInplaceUpdateState] = sigmaapi.InplaceUpdateStateFailed
		// Set assumed resource to last spec.
		lastSpec := util.LastSpecFromPod(assumed)
		if lastSpec == nil || len(lastSpec.Containers) != len(assumed.Spec.Containers) {
			return fmt.Errorf("get unexpected lastSpec from pod %s/%s: %v", pod.Namespace, pod.Name, lastSpec.Containers)
		}
		// Reset resource value.
		for idx, _ := range assumed.Spec.Containers {
			assumed.Spec.Containers[idx].Resources = lastSpec.Containers[idx].Resources
		}
		if err := ge.cache.AssumePod(assumed); err != nil {
			glog.Error(fmt.Errorf("failed to assume inplace-update pod %s/%s: %v", pod.Namespace, pod.Name, err))
		}
	} else {
		// Set inplace update state to "accepted".
		glog.V(4).Infof("setting inplace update pod %s/%s as accepted", pod.Namespace, pod.Name)
		assumed.Annotations[sigmaapi.AnnotationPodInplaceUpdateState] = sigmaapi.InplaceUpdateStateAccepted
		if len(allocSpec) > 0 {
			assumed.Annotations[sigmaapi.AnnotationPodAllocSpec] = allocSpec
		}
	}
	if err = util.PatchPod(ge.client, pod, assumed); err != nil {
		glog.Error(err)
	}
	return err
}

// NewGenericScheduler creates a genericScheduler object.
func NewGenericSchedulerExtend(
	cache schedulercache.Cache,
	eCache *equivalence.Cache,
	podQueue SchedulingQueue,
	predicates map[string]algorithm.FitPredicate,
	predicateMetaProducer algorithm.PredicateMetadataProducer,
	prioritizers []algorithm.PriorityConfig,
	priorityMetaProducer algorithm.PriorityMetadataProducer,
	extenders []algorithm.SchedulerExtender,
	volumeBinder *volumebinder.VolumeBinder,
	pvcLister corelisters.PersistentVolumeClaimLister,
	alwaysCheckAllPredicates bool,
	disablePreemption bool,
	percentageOfNodesToScore int32,
	client clientset.Interface,
) algorithm.ScheduleAlgorithm {

	gs := genericScheduler{
		cache:                    cache,
		equivalenceCache:         eCache,
		schedulingQueue:          podQueue,
		predicates:               predicates,
		predicateMetaProducer:    predicateMetaProducer,
		prioritizers:             prioritizers,
		priorityMetaProducer:     priorityMetaProducer,
		extenders:                extenders,
		cachedNodeInfoMap:        make(map[string]*schedulercache.NodeInfo),
		volumeBinder:             volumeBinder,
		pvcLister:                pvcLister,
		alwaysCheckAllPredicates: alwaysCheckAllPredicates,
		disablePreemption:        disablePreemption,
		percentageOfNodesToScore: percentageOfNodesToScore,
	}
	return &GenericSchedulerExtend{
		genericScheduler: gs,
		client:           client,
	}
}
