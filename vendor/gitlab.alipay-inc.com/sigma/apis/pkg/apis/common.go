/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

package apis

// AlipayGroupName is the group name used to identify k8s domain in alipay.com.
const AlipayGroupName string = "k8s.alipay.com"

const (
	// MetaAlipayPrefix is metadata used to store general informations.
	MetaAlipayPrefix string = "meta" + "." + AlipayGroupName
	// CustomAlipayPrefix is customer defined sub-domain.
	CustomAlipayPrefix string = "custom" + "." + AlipayGroupName
	// MandatoryAlipayPrefix is special sub-domain for mandatory labels.
	MandatoryAlipayPrefix string = "mandatory" + "." + AlipayGroupName
	// NodeAlipayPrefix is Node labels/taint/annotations key prefix
	NodeAlipayPrefix string = "node" + "." + AlipayGroupName
	// NetworkAlipayPrefix is network prefix for Alibaba Cloud
	NetworkAlipayPrefix string = "network" + "." + AlipayGroupName
	// FinalizerAlipayPrefix is finalizer prefix for Alipay
	FinalizerAlipayPrefix string = "finalizer" + "." + AlipayGroupName
)
