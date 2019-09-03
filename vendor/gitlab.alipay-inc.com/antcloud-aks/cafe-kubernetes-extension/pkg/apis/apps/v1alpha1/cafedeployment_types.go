/*
Copyright 2018 The Alipay.com Inc Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AlphaCafeDeploymentAnnotationReleaseConfirmedTrue  = "true"
	AlphaCafeDeploymentAnnotationReleaseConfirmedFalse = "false"
	AlphaCafeDeploymentAnnotationReleaseConfirmedAbort = "abort"

	// The number of times we retry updating
	UpdateRetries = 5

	CafeDeploymentEventTypeGetInPlaceSetFail = "FailedGetInPlaceSet"

	CafeDeploymentEventTypeConstructRevisionFail = "FailedCreateRevision"

	CafeDeploymentEventTypeDelDupInPlaceSetFail = "FailedDelDupInPlaceSet"

	CafeDeploymentEventTypeUnitCreateSucc = "SuccessfulUnitProvision"
	CafeDeploymentEventTypeUnitCreateFail = "FailedUnitProvision"

	CafeDeploymentEventTypeUnitDeleteSucc = "SuccessfulUnitReclaim"
	CafeDeploymentEventTypeUnitDeleteFail = "FailedUnitReclaim"

	CafeDeploymentEventTypeScaleSucc = "SuccessfulScale"
	CafeDeploymentEventTypeScaleFail = "FailedScale"

	CafeDeploymentEventTypeRescheduleSucc = "SuccessfulReschedule"
	CafeDeploymentEventTypeRescheduleFail = "FailedReschedule"

	CafeDeploymentEventTypeRelease     = "Release"
	CafeDeploymentEventTypeReleaseFail = "FailedRelease"

	CafeDeploymentEventTypeRollbackSucc = "SuccessfulRollback"
)

type UnitType string

const (
	UnitTypeZone UnitType = "Zone"
	UnitTypeCell UnitType = "Cell"
)

type UpgradeType string

const (
	UpgradeBeta  UpgradeType = "Beta"
	UpgradeBatch UpgradeType = "Batch"
)

type CafeDeploymentConditionType string

const (
	CafeDeploymentConditionTypeCellCreateFail CafeDeploymentConditionType = "CellProvisionFailure"
	CafeDeploymentConditionTypeCellDeleteFail CafeDeploymentConditionType = "CellReclaimFailure"
	CafeDeploymentConditionTypeScaleFail      CafeDeploymentConditionType = "ScaleFailure"
	CafeDeploymentConditionTypeRescheduleFail CafeDeploymentConditionType = "RescheduleFailure"
	CafeDeploymentConditionTypeReleaseFail    CafeDeploymentConditionType = "ReleaseFailure"
	CafeDeploymentConditionTypeRollbackFail   CafeDeploymentConditionType = "RollbackFailure"
)

type ReleaseProgress string

const (
	CafeDeploymentReleaseProgressWaitingForConfirmation ReleaseProgress = "WaitingForConfirmation"
	CafeDeploymentReleaseProgressExecuting              ReleaseProgress = "Executing"
	CafeDeploymentReleaseProgressCompleted              ReleaseProgress = "Completed"
	CafeDeploymentReleaseProgressAborted                ReleaseProgress = "Aborted"
)

type AutoScheduleProgress string

const (
	CafeDeploymentAutoRescheduleStatusRescheduling      AutoScheduleProgress = "Rescheduling"
	CafeDeploymentAutoRescheduleStatusCompleted         AutoScheduleProgress = "Completed"
	CafeDeploymentAutoRescheduleStatusNoUnitSchedulable AutoScheduleProgress = "NoUnitSchedulable"
)

var (
	AutoRescheduleInitialDelaySecondsDefaultSeconds int32 = 10
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CafeDeployment
// +k8s:openapi-gen=true
// +resource:path=cafedeployments,strategy=CafeDeploymentStrategy
type CafeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CafeDeploymentSpec   `json:"spec,omitempty"`
	Status CafeDeploymentStatus `json:"status,omitempty"`
}

// CafeDeploymentSpec defines the desired state of CafeDeployment
type CafeDeploymentSpec struct {
	// replicas is the totally desired number of replicas of all the owning InPlaceSet.
	// If unspecified, defaults to 0.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// selector is a label query over pods that should match the replica count.
	// It must match the pod template's labels.
	Selector metav1.LabelSelector `json:"selector,omitempty"`

	// template is the object that describes the pod that will be created if
	// insufficient replicas are detected. Each pod stamped out by the InPlaceSet
	// will fulfill this Template, but have a unique identity from the rest
	// of the InPlaceSet.
	Template corev1.PodTemplateSpec `json:"template,omitempty"`

	// Contains the information of cells topology
	Topology Topology `json:"topology,omitempty"`

	Strategy CafeDeploymentUpgradeStrategy `json:"strategy,omitempty"`

	// indicate the number of histories to be conserved
	// If unspecified, defaults to 20
	// +optional
	HistoryLimit int32 `json:"historyLimit,omitempty"`
}

type CafeDeploymentUpgradeStrategy struct {
	// Indicate the type of the upgrade process
	UpgradeType UpgradeType `json:"upgradeType,omitempty"`

	// Indicates that the deployment is paused and will not be processed by the
	// deployment controller.
	// +optional
	Pause bool `json:"pause,omitempty"`

	// Indicate if it should wait for a confirmation before continue to next batch
	// Defaults false
	NeedWaitingForConfirm bool `json:"needWaitingForConfirm,omitempty"`

	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`

	// Indicate how many pods under the CafeDeployment to be upgraded at one time
	// Defaults nil (fully release)
	// +optional
	BatchSize *int32 `json:"batchSize,omitempty"`

	// The maximum number of pods that can be scheduled above the original
	// number of pods in one group during the update
	// +optional
	MaxSurgeSizeInGroup int32 `json:"maxSurgeSizeInGroup,omitempty"`
}

type Topology struct {
	// Type of unit
	UnitType UnitType `json:"unitType,omitempty"`

	// Contains the names of each cells
	Values []string `json:"values,omitempty"`

	// Indicate the pod spread detail to each unit
	// +optional
	UnitReplicas map[string]intstr.IntOrString `json:"unitReplicas,omitempty"`

	// Configuration of auto-reschedule feature
	// +optional
	AutoReschedule *AutoScheduleConfig `json:"autoReschedule,omitempty"`
}

type AutoScheduleConfig struct {
	// A switch to enable auto-reschedule feature, which will rebalance all pods
	// to each unit when some pods is not able to be scheduled because of resources insufficiency
	// Defaults to false
	// +optional
	Enable bool `json:"enable,omitempty"`

	// Number of seconds after the CafeDeployment has been provisioned before auto-reschedule feature works.
	// Defaults to 10 seconds
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`
}

// CafeDeploymentStatus defines the observed state of CafeDeployment
type CafeDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// observedGeneration is the most recent generation observed for this InPlaceSet. It corresponds to the
	// InPlaceSet's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// the number of scheduled replicas for the cafeDeployment
	// +optional
	ScheduledReplicas int32 `json:"scheduledReplicas,omitempty"`

	// The number of available replicas (ready for at least minReadySeconds) for this replica set.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Replicas is the most recently observed number of replicas.
	Replicas int32 `json:"replicas,omitempty"`

	// The number of pods in current version
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// The number of ready current revision replicas for this InPlaceSet.
	// A pod is updated ready means all of its container has bean updated by sigma.
	// +optional
	UpdatedReadyReplicas int32 `json:"updatedReadyReplicas,omitempty"`

	// The number of available current revision replicas for this InPlaceSet.
	// A pod is updated available means the pod is ready for current revision and accessible
	// +optional
	UpdatedAvailableReplicas int32 `json:"updatedAvailableReplicas,omitempty"`

	// The number of pods that have labels matching the labels of the pod template of the InPlaceSet.
	// +optional
	FullyLableledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`

	// Count of hash collisions for the DaemonSet. The DaemonSet controller
	// uses this field as a collision avoidance mechanism when it needs to
	// create the name for the newest ControllerRevision.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`

	// CurrentRevision, if not empty, indicates the version of the CafeDeployment used to generate InPlaceSet in the .
	CurrentRevision string `json:"currentRevision,omitempty"`

	// Records the topology detail information of the replicas of each unit.
	UnitReplicas map[string]int32 `json:"unitReplicas,omitempty"`

	// Represents the latest available observations of a InPlaceSet's current state.
	// +optional
	Conditions []CafeDeploymentCondition `json:"conditions,omitempty"`

	// Records the information of release progress.
	// +optional
	ReleaseStatus *ReleaseStatus `json:"releaseStatus,omitempty"`

	// Records the information of auto reschedule.
	AutoRescheduleStatus *AutoRescheduleStatus `json:"autoRescheduleStatus,omitempty"`
}

type CafeDeploymentCondition struct {
	// Type of in place set condition.
	Type CafeDeploymentConditionType `json:"type,omitempty"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty"`

	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"last_transition_time,omitempty"`

	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type ReleaseStatus struct {
	// Records the latest revision.
	// +optional
	UpdateRevision string `json:"updateRevision,omitempty"`

	// Records the current batch serial number.
	// +optional
	CurrentBatchIndex int32 `json:"currentBatchIndex,omitempty"`

	// Records the current partition.
	// +optional
	CurrentPartitions map[string]int32 `json:"currentPartitions,omitempty"`

	// The phase current release reach
	// +optional
	Progress ReleaseProgress `json:"progress,omitempty"`

	// Last time the release transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type AutoRescheduleStatus struct {
	// Count number of auto reschedule
	Count int64 `json:"count,omitempty"`

	// The progress of auto reschedule
	Progress AutoScheduleProgress `json:"progress,omitempty"`

	// Last time the reschedule transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// DefaultingFunction sets default CafeDeployment field values
func (CafeDeploymentSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*CafeDeployment)
	// set default field values here
	DefaultingCafeDeployment(obj)
	log.Printf("Defaulting fields for CafeDeployment %s\n", obj.Name)
}

func DefaultingCafeDeployment(o *CafeDeployment) {
	if o.Spec.HistoryLimit == 0 {
		o.Spec.HistoryLimit = 20
	}

	if o.Spec.Strategy.UpgradeType == "" {
		o.Spec.Strategy.UpgradeType = UpgradeBeta
	}

	if o.Spec.Topology.UnitType == "" {
		o.Spec.Topology.UnitType = UnitTypeCell
	}

	if o.Spec.Topology.AutoReschedule != nil {
		if o.Spec.Topology.AutoReschedule.InitialDelaySeconds == nil {
			o.Spec.Topology.AutoReschedule.InitialDelaySeconds = &AutoRescheduleInitialDelaySecondsDefaultSeconds
		}
	}

	SetDefaults_PodSpec(&o.Spec.Template.Spec)
}

func ParseUnitReplicas(replicas int32, unitReplicas intstr.IntOrString) (int32, error) {
	if unitReplicas.Type == intstr.Int {
		if unitReplicas.IntVal < 0 {
			return 0, fmt.Errorf("unitReplicas (%d) should not be less than 0", unitReplicas.IntVal)
		}
		return unitReplicas.IntVal, nil
	} else {
		strVal := unitReplicas.StrVal
		if !strings.HasSuffix(strVal, "%") {
			return 0, fmt.Errorf("unitReplicas (%s) only support int value or percentage value with a suffix '%%'", strVal)
		}

		intPart := strVal[:len(strVal)-1]
		percent64, err := strconv.ParseInt(intPart, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("unitReplicas (%s) should be correct percentage value", strVal)
		}

		if percent64 > int64(100) || percent64 < int64(0) {
			return 0, fmt.Errorf("unitReplicas (%s) should be in range (0, 100]", strVal)
		}

		return int32(replicas * int32(percent64) / 100), nil
	}
}
