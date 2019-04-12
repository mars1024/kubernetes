package apis

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FinalizerAlibabaCloudCni           = FinalizerAlipayPrefix + "/alibabacloud-cni"
	FinalizerNetworkInterface          = FinalizerAlipayPrefix + "/network-interface"
	FinalizerPredictedNetworkInterface = FinalizerAlipayPrefix + "/predicted-network-interface"
)

const (
	AnnotationNetworkSpec      = NetworkAlipayPrefix + "/spec"
	AnnotationNetworkInterface = NetworkAlipayPrefix + "/interface"
)

const (
	LabelVSwitchID   = NetworkAlipayPrefix + "/vswitch"
	LabelNetworkNode = NetworkAlipayPrefix + "/node"
	LabelNetworkPod  = NetworkAlipayPrefix + "/pod"
)

type NetworkSpec struct {
	ReservedIP            string                `json:"reservedIP,omitempty"`
	VSwitchSelector       *metav1.LabelSelector `json:"vSwitchSelector"`
	SecurityGroupSelector *metav1.LabelSelector `json:"securityGroupSelector"`
	InterfaceSelector     *metav1.LabelSelector `json:"interfaceSelector"`
}
