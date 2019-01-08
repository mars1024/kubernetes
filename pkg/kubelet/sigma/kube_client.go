package sigma

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
)

// GetDanglingPods can get dangling pods from apiserver.
func GetDanglingPods(kubeClient clientset.Interface, nodeName string) ([]sigmak8sapi.DanglingPod, error) {
	node, err := kubeClient.CoreV1().Nodes().Get(string(nodeName), metav1.GetOptions{})
	if err != nil {
		glog.Warningf("Failed to get node: %s", nodeName)
		return []sigmak8sapi.DanglingPod{}, err
	}
	danglingPods, err := GetDanglingPodsFromNodeAnnotation(node)
	if err != nil {
		return []sigmak8sapi.DanglingPod{}, err
	}
	return danglingPods, nil
}

// UpdateDanglingPods can update dangling pods information to node's annotation.
func UpdateDanglingPods(kubeClient clientset.Interface, nodeName string, danglingPods []sigmak8sapi.DanglingPod) error {
	danglingPodsBytes, err := json.Marshal(danglingPods)
	if err != nil {
		glog.Warningf("Failed to marshal danglingPods: %v, update danglingPods in next loop", danglingPods)
		return err
	}
	glog.V(0).Infof("Dangling pod(s) %v will be updated to node's annotation", string(danglingPodsBytes))
	patchData := fmt.Sprintf(
		`{"metadata":{"annotations":{"%s":%q}}}`, sigmak8sapi.AnnotationDanglingPods, string(danglingPodsBytes))
	if _, err := kubeClient.CoreV1().Nodes().Patch(string(nodeName), types.StrategicMergePatchType, []byte(patchData)); err != nil {
		glog.Warningf("Failed to update dangling pods: %v, do it in next loop", err)
		return err
	}
	return nil
}
