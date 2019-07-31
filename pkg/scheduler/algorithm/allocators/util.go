package allocators

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/scheduler/util"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
)

// IsExclusiveContainer determine whether the container is
// a exclusive container
func IsExclusiveContainer(pod *v1.Pod, container *v1.Container) bool {
	if pod == nil {
		return false
	}

	labels := pod.Labels
	if v, ok := labels[ExclusiveCPU]; ok {
		if v == "1" || v == "yes" || v == "true" {

			if container == nil {
				// Treat all containers in a pod as exclusive containers?
				return true
			}
			req := container.Resources.Requests[v1.ResourceCPU]
			limit := container.Resources.Limits[v1.ResourceCPU]
			if !ok {
				return false
			}
			if req.Cmp(limit) != 0 {
				return false
			}
			return req.Value()*1000 == req.MilliValue()
		}
	}
	return false
}

func getAllocContainer(containers []sigmak8sapi.Container, name string) *sigmak8sapi.Container {
	for _, c := range containers {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

// getPodCPUSet get the cpuset allocated for cpu-affinity pod
// this doesn't contain the cpushare pod
func getPodCPUSet(pod *v1.Pod) (cpuset.CPUSet, bool) {
	allocSpec := util.AllocSpecFromPod(pod)
	podCPUSet := cpuset.NewBuilder()
	if allocSpec != nil && len(allocSpec.Containers) > 0 {
		for _, c := range allocSpec.Containers {
			if c.Resource.CPU.CPUSet != nil {
				ids := c.Resource.CPU.CPUSet.CPUIDs
				glog.V(5).Infof("container(%s/%s/%s) cpuset %v", pod.Namespace, pod.Name, c.Name, ids)
				podCPUSet.Add(ids...)
			}
		}
	}
	return podCPUSet.Result(), false
}

func ContainerCPUCount(container *v1.Container) int {
	if limit, ok := container.Resources.Limits[v1.ResourceCPU]; ok {
		return int((limit.MilliValue() + int64(999)) / 1000)
	}
	return 0 // this should be a CPUShare container?
}

func ContainerName(pod *v1.Pod, container *v1.Container) string {
	return fmt.Sprintf("%s/%s/%s", pod.Namespace, pod.Name, container.Name)
}

// IsSharedCPUSetPod determines whether pod
// is a SharedCPUSet pod
func IsSharedCPUSetPod(pod *v1.Pod) bool {
	alloc := util.AllocSpecFromPod(pod)
	if alloc == nil {
		// Native pod goes native way
		return false
	}
	return !IsExclusiveContainer(pod, nil)
}

// IsPodCpuSet define if the pod with cpuset request
func IsPodCpuSet(pod *v1.Pod) bool {
	alloc := util.AllocSpecFromPod(pod)
	if alloc == nil {
		// Native pod goes native way
		return false
	}
	return true
}

// GenAllocSpecAnnotation create the annotation for each container in pod
// NOTE: simge-cerebellum set same CPUIDs for all containers
func GenAllocSpecAnnotation(pod *v1.Pod, containerCPUs ContainerCPUAssignments) []byte {
	// Set CPUIDs to pod annotations.
	allocSpec := util.AllocSpecFromPod(pod)

	//Now we apply these CPUIDs to each containers
	// TODO(yuzhi.wx): Later we may apply same CPUIDs for all container as
	// 	we consider that they are sharing the same cores in one pod.
	// 	e.g. mosn sidecar container with app container in the same pod.
	for i, c := range allocSpec.Containers {
		if c.Resource.CPU.CPUSet == nil {
			allocSpec.Containers[i].Resource.CPU.CPUSet = &sigmak8sapi.CPUSetSpec{}
		}
		cName := c.Name
		if set, ok := containerCPUs[cName]; ok {
			allocSpec.Containers[i].Resource.CPU.CPUSet.CPUIDs = set.ToSlice()
		} else {
			glog.Warningf("container %s/%s/%s is not setting any CPUIDs, but existing in alloc-spec.", pod.Namespace, pod.Name, c.Name)
		}
	}

	b, err := json.Marshal(allocSpec)
	if err != nil {
		glog.Errorf("[GenAllocSpecAnnotation] json.Marshal failed: %v", err)
		return nil
	}
	return b
}

func MakePodCPUPatch(pod *v1.Pod, containerCPUs ContainerCPUAssignments) []byte {
	podCopy := pod.DeepCopy()

	//// Patch pod on inplace updating.
	//if kubeutils.IsInplaceUpdateRequest(pod) {
	//	if v, ok := pod.Annotations[kubeutils.NeedToPatchLastSpecInAnnotations]; ok {
	//		if v == kubeutils.True {
	//			// Need to patch last spec in annotations.
	//			delete(pod.Annotations, sigmak8s.AnnotationPodLastSpec)
	//		}
	//	}
	//
	//	// Update alloc spec if needed for inplace update processing.
	//	if alloc.ResourceAllocated.CPU.IsCPUSet {
	//		val := string(GenAllocSpecAnnotation(pod, alloc.ResourceAllocated.CPU.Set))
	//		pod.Annotations[sigmak8s.AnnotationPodAllocSpec] = val
	//	}
	//
	//	pod.Annotations[sigmak8s.AnnotationPodInplaceUpdateState] = sigmak8s.InplaceUpdateStateAccepted
	//	hasPodChanged = true
	//}
	//
	//if !hasPodChanged {
	//	return []byte{}
	//}
	val := string(GenAllocSpecAnnotation(pod, containerCPUs))
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = val
	patch, err := util.CreatePodPatch(podCopy, pod)
	if err != nil {
		glog.Error(err)
		return []byte{}
	}
	if len(patch) == 0 || string(patch) == "{}" {
		glog.Warningf("no patch for setting pod %v annotations", pod.Name)
		return []byte{}
	}
	glog.V(8).Infof("patch result for pod %q: %s", pod.Name, string(patch))
	return patch
}

func DoPatchAll(kubeClient clientset.Interface, pod *v1.Pod, podPatch []byte, nodeCPUSharePool cpuset.CPUSet, nodeName string) error {
	//nodeName := pod.Spec.NodeName
	if len(nodeName) > 0 {
		node, err := kubeClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("fail to get node %s from apiserver: %v", nodeName, err)
		}
		ids := nodeCPUSharePool.ToSlice()
		nodeCPUIDs := make([]int32, 0)
		for _, id := range ids {
			nodeCPUIDs = append(nodeCPUIDs, int32(id))
		}
		nodePatch, err := CreateNodeCPUSharePoolPatch(node, nodeCPUIDs)
		if err != nil {
			return fmt.Errorf("fail create node patch: %v", err)
		}
		err = retryOnEOF(func() error {
			if len(nodePatch) == 0 || string(nodePatch) == "{}" {
				glog.Infof("nodePatch is empty: %s", string(nodePatch))
				return nil
			}
			_, err := kubeClient.CoreV1().Nodes().Patch(nodeName, apimachinerytypes.StrategicMergePatchType, nodePatch)
			return err
		})
		if err != nil {
			return fmt.Errorf("fail to patch node %s with %s: %v", nodeName, string(nodePatch), err)
		}
	}

	// assume that pod patch and binding is mutually exclusive.
	if len(podPatch) != 0 {
		glog.Infof("Attempting to patch pod %s/%s with %s", pod.Namespace, pod.Name, string(podPatch))
		if _, err := kubeClient.CoreV1().Pods(pod.Namespace).Patch(pod.Name, apimachinerytypes.StrategicMergePatchType, podPatch); err != nil {
			glog.Errorf("Fail to patch pod %s/%s with %s: %v", pod.Namespace, pod.Name, string(podPatch), err)
			return err
		}
	}
	return nil
}

// CreateNodeCPUSharePoolPatch creates kubernetes node patch for
// updating CPU share pool
func CreateNodeCPUSharePoolPatch(node *v1.Node, cpus []int32) ([]byte, error) {
	if node == nil {
		return nil, fmt.Errorf("failed to create CPUSharePool patch, node is nil")
	}
	b, err := json.Marshal(&sigmak8sapi.CPUSharePool{CPUIDs: cpus})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CPUSharePool: %v", err)
	}
	s := string(b)
	previousCPUSharePool, ok := node.Annotations[sigmak8sapi.AnnotationNodeCPUSharePool]
	if ok && s == previousCPUSharePool {
		return nil, nil
	}
	nodeCopy := node.DeepCopy()
	nodeCopy.Annotations[sigmak8sapi.AnnotationNodeCPUSharePool] = s
	cur, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal v1.Node: %v", err)
	}
	mod, err := json.Marshal(nodeCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal v1.Node.DeepCopy: %v", err)
	}
	return strategicpatch.CreateTwoWayMergePatch(cur, mod, v1.Node{})
}

func retryOnEOF(f func() error) error {
	times := 3
	factor := 1
	for i := 0; i < times; i++ {
		err := f()
		switch {
		case err == nil:
			if i > 0 {
				glog.Infof("succeed after %d retry", i)
			}
			return nil
		case err.Error() == io.EOF.Error(): // retry
			time.Sleep(100 * time.Duration(factor) * time.Millisecond)
			factor <<= 1
		default:
			return err
		}
	}
	return fmt.Errorf("retry %d times and still get %v", times, io.EOF)
}
