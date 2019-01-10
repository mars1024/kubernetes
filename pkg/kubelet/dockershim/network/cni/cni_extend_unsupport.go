// +build !linux

package cni

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/network"
)

func UpdateCNIServiceAddress(plugins []network.NetworkPlugin, networkPluginName string, kubeClient clientset.Interface) {
}
