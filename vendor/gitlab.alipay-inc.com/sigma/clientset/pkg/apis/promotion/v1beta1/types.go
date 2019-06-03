package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PromotionType is the promotion status current cluster is at
type PromotionType string

const (
	// PromotionTypeTaobao means cluster is supporting taobao double 11 promotion
	PromotionTypeTaobao PromotionType = "taobao"
	// PromotionTypeAntMember means cluster is supporting ant member
	PromotionTypeAntMember PromotionType = "antmember"
)

// PromotionStateUpdateStrategy indicates how to update promotion state
type PromotionStateUpdateStrategy string

// GrayStrategy includes a number of nodes and their promotion type
type GrayStrategy struct {
	// GrayType is the promotion type the gray nodes should be at
	GrayType PromotionType `json:"grayType"`

	// Nodes indicates which nodes should be in the gray list
	Nodes []v1.NodeSelector `json:"nodes"`
}

// PromotionClaimSpec defines the desired state of PromotionClaim
type PromotionClaimSpec struct {
	// ClusterType indicates which promotionType we expect the cluster should be,
	// either PromotionTypeTaobao or PromotionTypeAntMember.
	ClusterType PromotionType `json:"clusterType"`

	// GrayList includes nodes whose promotion type is different from cluster promotion type,
	// it is used for gray release.
	GrayList GrayStrategy `json:"grayList,omitempty"`

	// Strategy indicates how to update promotion state, this is undefinied right now.
	UpdateStrategy PromotionStateUpdateStrategy `json:"updateStrategy,omitempty"`
}

// PromotionClaimStatus defines the observed state of PromotionClaim
type PromotionClaimStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PromotionClaim is the Schema for the promotionclaims API
type PromotionClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PromotionClaimSpec   `json:"spec,omitempty"`
	Status PromotionClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PromotionClaimList contains a list of PromotionClaim
type PromotionClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionClaim `json:"items"`
}

// NodeSwapClaimPhase indicates which phase the node is at
type NodeSwapClaimPhase string

const (
	// NodeSwapClaimPhaseTaobao means the node is at Taobao promotion phase.
	// Which indicates all pods on the node are ready for taobao promotion traffic.
	NodeSwapClaimPhaseTaobao NodeSwapClaimPhase = "taobao"

	// NodeSwapClaimPhaseAntMember means the node is at AntMember promotion phase.
	// Which indicates all pods on the node are ready for ant member promotion traffic.
	NodeSwapClaimPhaseAntMember NodeSwapClaimPhase = "antmember"

	// NodeSwapClaimPhaseTaobaoTransition means the node is in transition to taobao type
	NodeSwapClaimPhaseTaobaoTransition NodeSwapClaimPhase = "taobao-transition"

	// NodeSwapClaimPhaseAntMemberTransition means the node is in transition to antmember type
	NodeSwapClaimPhaseAntMemberTransition NodeSwapClaimPhase = "antmember-transition"
)

type NodeSwapClaimConditionType string

const (
	NodeSwapConditionUnknown NodeSwapClaimConditionType = "unknown"
)

type PodPhase string

const (
	// PodPhasePending means the pod is watched, but not in any promotion status yet
	PodPhasePending PodPhase = "Pending"

	// PodPhaseSwapIn means pod has succeefully swapped in memory
	PodPhaseSwapIn PodPhase = "SwapIn"
	// PodPhaseSwapInOnGoing means pod is in swapping in process
	PodPhaseSwapInOnGoing PodPhase = "SwapInOnGoing"
	PodPhaseSwapInFailed  PodPhase = "SwapInFailed"

	PodPhaseSwapOut        PodPhase = "SwapOut"
	PodPhaseSwapOutOnGoing PodPhase = "SwapOutOnGoing"
	PodPhaseSwapOutFailed  PodPhase = "SwapOutFailed"

	// PodPhaseNormal means pod is not in any promotion phase, just a normal pod
	PodPhaseNormal PodPhase = "PodNormal"
)

// PodConditionType
type PodConditionType string

const (
	PodConditionInitialzed   PodConditionType = "Initialized"
	PodConditionSwapIn       PodConditionType = "SwapIn"
	PodConditionSwapOut      PodConditionType = "SwapOut"
	PodConditionTrafficOff   PodConditionType = "TrafficOff"
	PodConditionTrafficOn    PodConditionType = "TrafficOn"
	PodConditionHeapSizeUp   PodConditionType = "HeapSizeUp"
	PodConditionHeapSizeDown PodConditionType = "HeapSizeDown"
)

// NodeSwapClaimCondition records node condition change history
type NodeSwapClaimCondition struct {
	// NodeSwapClaimConditionType is a specific node swap condition
	Type NodeSwapClaimConditionType `json:"type"`
	// Status indicates the condition status, possible values: true, false, unknown
	Status v1.ConditionStatus `json:"status"`
	// LastProbeTime records condtion probe timestamp
	LastProbeTime metav1.Time `json:"lastProbeTime"`
	// LastTransitinoTime records condition traisition timestamp
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
}

// PodCondition records pod swap update recod
type PodCondition struct {
	Type               PodConditionType   `json:"type"`
	Status             v1.ConditionStatus `json:"status"`
	LastProbeTime      metav1.Time        `json:"lastProbeTime"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime"`
	Reason             string             `json:"reason"`
	Message            string             `json:"message"`
}

// PodStatus records status of all pods on the node
type PodStatus struct {
	PodName            string         `json:"podName"`
	PodPhase           PodPhase       `json:"podPhase"`
	PodCondition       []PodCondition `json:"podCondition"`
	LastTransitionTime metav1.Time    `json:"lastTransitionTime"`
}

// NodeSwapClaimSpec defines the desired state of NodeSwapClaim
type NodeSwapClaimSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NodeName indicates which node the spec is targeted
	NodeName string `json:"nodeName"`

	// TargetStatus indicates what promotion status this node should be at
	TargetStatus PromotionType `json:"targetStatus"`
}

// NodeSwapClaimStatus defines the observed state of NodeSwapClaim
type NodeSwapClaimStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase indicates which phase current node is at
	Phase NodeSwapClaimPhase `json:"phase"`

	// Conditions records node swap condition change history
	Conditions []NodeSwapClaimCondition `json:"conditions"`

	// PodStatus records status of pod on the node
	PodsStatus []PodStatus `json:"podStatus"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeSwapClaim is the Schema for the nodeswapclaims API
type NodeSwapClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSwapClaimSpec   `json:"spec,omitempty"`
	Status NodeSwapClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeSwapClaimList contains a list of NodeSwapClaim
type NodeSwapClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeSwapClaim `json:"items"`
}
