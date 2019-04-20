package core

import (
	"encoding/json"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/allocators"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/core/equivalence"
	"k8s.io/kubernetes/pkg/scheduler/volumebinder"
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
