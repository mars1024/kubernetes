/*
Copyright 2019 The Alipay Authors.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Indicate a StorageClass associate to some mount points on a Node
type DiskStorageClass struct {
	// StorageClassName of these MountPoints
	StorageClassName string `json:"storageClassName"`
	// MountPoints of this StorageClass
	MountPoints []string `json:"mountPoints"`
}

type NodeDiskStorageClassSpec struct {
	// Label selector for Nodes.
	Selector *metav1.LabelSelector `json:"selector"`

	// Indicate the relationship of StorageClass and disk mount points
	StorageClasses []DiskStorageClass `json:"storageClasses"`

	// Indicate the root dir of these Nodes
	Root string `json:"root"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeDiskStorageClass is used for bind the relationship of StorageClass and disk mount points of selected Nodes
// +k8s:openapi-gen=true
type NodeDiskStorageClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeDiskStorageClassSpec   `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeDiskStorageClassList contains a list of NodeDiskStorageClass
type NodeDiskStorageClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeDiskStorageClass `json:"items"`
}
