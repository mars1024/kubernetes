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
	"sync"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
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
	rwLock *sync.RWMutex
}

// Allocate allocates the resources for the pod on given host
func (ge *GenericSchedulerExtend) Allocate(pod *v1.Pod, host string) error {
	alloc := util.AllocSpecFromPod(pod)
	if alloc == nil {
		// Native pod goes native way
		return nil
	}
	nodeInfo := ge.cachedNodeInfoMap[host]
	if util.LocalInfoFromNode(nodeInfo.Node()) == nil {
		glog.V(4).Infof("node %s is nil or not eligible for cpuset Allocation", host)
		return nil
	}
	ge.rwLock.Lock()
	allocator := allocators.NewCPUAllocator(nodeInfo)
	result, err := allocator.Allocate(pod)
	if err != nil {
		ge.rwLock.Unlock()
		return err
	}
	if len(result) <= 0 {
		glog.V(3).Infof("patch result is %#v for pod %s/%s, skipping patch", result, pod.Namespace, pod.Name)
		ge.rwLock.Unlock()
		return nil
	}
	glog.V(3).Infof("going to patch CPUSet %v for pod %s/%s", result, pod.Namespace, pod.Name)
	originAlloc, allocExists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	patchPod := allocators.MakePodCPUPatch(pod, result)
	cpuAllocator, ok := allocator.(*allocators.CPUAllocator)
	var nodeShareCPUPool cpuset.CPUSet
	var nodeName string
	if ok {
		nodeShareCPUPool = cpuAllocator.NodeCPUSharePool()
		nodeName = host
	}
	ge.rwLock.Unlock() // Unlock now, do not wait for patch result
	err = allocators.DoPatchAll(ge.client, pod, patchPod, nodeShareCPUPool, nodeName)
	if err != nil {
		// revert the alloc-spec to previous status
		if allocExists {
			pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = originAlloc
		}
		return allocators.ErrAllocatorFailure(allocator.Name(), err.Error())
	}
	return err
}

func (ge *GenericSchedulerExtend) inplaceUpdateReallocate(pod *v1.Pod) (string, error) {
	glog.V(4).Infof("[inplaceUpdateReallocate]Attempting to process inplace update pod %s/%s", pod.Namespace, pod.Name)

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
		errRemove := ge.cache.RemovePod(pod)
		if errRemove != nil {
			glog.Warningf("[inplaceUpdateReallocate]failed to remove pod from scheduler cache:%s", errRemove.Error())
		}
	}
	defer func() {
		ge.cache.FinishBinding(pod)
	}()
	if err != nil {
		glog.Errorf("[inplaceUpdateReallocate]failed to forget/remove pod %s/%s from cache: %s", pod.Namespace, pod.Name, err.Error())
		return "", err
	}
	// Call PodFitsResources to check resources.
	if fit, predicateFails, _ := predicates.PodCPUSetResourceFit(pod, nil, nodeInfo); !fit {
		return "", fmt.Errorf("failed to fit resource[PodCPUSetResourceFit] in inplace update processing with predicateFails: %+v", predicateFails)
	}

	if fit, predicateFails, _ := predicates.PodFitsResources(pod, nil, nodeInfo); !fit {
		return "", fmt.Errorf("failed to fit resource[PodFitsResources] in inplace update processing with predicateFails: %+v", predicateFails)
	}
	// If CPU value is not changed, return immediately.
	if !util.IsResourceChanged(lastSpec, &pod.Spec, v1.ResourceCPU) {
		glog.V(4).Infof("[inplaceUpdateReallocate]CPU resource not changed or this is a cpushare request, return immediately")
		return "", nil
	}

	glog.V(4).Infof("[inplaceUpdateReallocate]reallocating CPUSet for pod %q", pod.Name)
	// Call CPUSetAllocation to alloc CPUIDs.
	cpuAlloc := allocators.NewCPUAllocator(nodeInfo)
	cpuAssignments, err := cpuAlloc.Reallocate(pod)
	return string(allocators.GenAllocSpecAnnotation(pod, cpuAssignments)), err
}

func (ge *GenericSchedulerExtend) HandleInplacePodUpdate(pod *v1.Pod) (string, error) {
	allocSpec, err := ge.inplaceUpdateReallocate(pod)
	if err != nil {
		assumed := pod.DeepCopy()
		// Set assumed resource to last spec.
		lastSpec := util.LastSpecFromPod(assumed)
		if lastSpec == nil || len(lastSpec.Containers) != len(assumed.Spec.Containers) {
			return "", fmt.Errorf("get unexpected lastSpec from pod %s/%s: %v", pod.Namespace, pod.Name, lastSpec.Containers)
		}
		// Reset resource value, so that assume the correct resources into cache
		for idx := range assumed.Spec.Containers {
			assumed.Spec.Containers[idx].Resources = lastSpec.Containers[idx].Resources
		}
		// TODO(yuzhi.wx) remove the assume for now, not figure out its purpose
		//errAssumed := ge.cache.AssumePod(assumed)
		//if errAssumed != nil {
		//	glog.Error(fmt.Errorf("failed to assume inplace-update pod %s/%s: %v", pod.Namespace, pod.Name, err))
		//}
		// return the original error
		return "", err
	}
	return allocSpec, nil
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
		rwLock:           new(sync.RWMutex),
	}
}
