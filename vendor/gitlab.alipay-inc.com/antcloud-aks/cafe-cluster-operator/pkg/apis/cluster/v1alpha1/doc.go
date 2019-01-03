


// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster
// +k8s:defaulter-gen=TypeMeta
// +groupName=cluster.aks.cafe.sofastack.io
package v1alpha1 // import "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"

