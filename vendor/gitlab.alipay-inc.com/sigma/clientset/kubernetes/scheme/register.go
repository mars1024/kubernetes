/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package scheme

import (
	apiextensionsv1beta1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/apiextensions/v1beta1"
	apiregistrationv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/apiregistration/v1"
	clusterv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/cluster/v1"
	kokv1alpha1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/kok/v1alpha1"
	machinev1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/machine/v1"
	monitoringv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/monitoring/v1"
	networkv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/network/v1"
	opsv1alpha1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/ops/v1alpha1"
	profilev1alpha1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/profile/v1alpha1"
	promotionv1beta1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/promotion/v1beta1"
	quotav1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/quota/v1"
	schedulingextensionsv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/schedulingextensions/v1"
	storageextensionsv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/storageextensions/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

var Scheme = scheme.Scheme
var Codecs = scheme.Codecs
var ParameterCodec = scheme.ParameterCodec

var localSchemeBuilder = runtime.SchemeBuilder{
	apiextensionsv1beta1.AddToScheme,
	apiregistrationv1.AddToScheme,
	clusterv1.AddToScheme,
	kokv1alpha1.AddToScheme,
	machinev1.AddToScheme,
	monitoringv1.AddToScheme,
	networkv1.AddToScheme,
	opsv1alpha1.AddToScheme,
	profilev1alpha1.AddToScheme,
	promotionv1beta1.AddToScheme,
	quotav1.AddToScheme,
	schedulingextensionsv1.AddToScheme,
	storageextensionsv1.AddToScheme,
}

// AddToScheme adds all types of this clientset into the given scheme. This allows composition
// of clientsets, like in:
//
//   import (
//     "k8s.io/client-go/kubernetes"
//     clientsetscheme "k8s.io/client-go/kubernetes/scheme"
//     aggregatorclientsetscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"
//   )
//
//   kclientset, _ := kubernetes.NewForConfig(c)
//   _ = aggregatorclientsetscheme.AddToScheme(clientsetscheme.Scheme)
//
// After this, RawExtensions in Kubernetes types will serialize kube-aggregator types
// correctly.
var AddToScheme = localSchemeBuilder.AddToScheme

func init() {
	utilruntime.Must(AddToScheme(Scheme))
}
