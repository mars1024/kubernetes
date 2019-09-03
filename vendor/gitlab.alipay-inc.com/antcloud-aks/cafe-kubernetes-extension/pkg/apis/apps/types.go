package apps

// Work around issue when `apiserver-boot build generated` after using alias
type UnitType string

const (
	UnitTypeZone UnitType = "Zone"
	UnitTypeCell UnitType = "Cell"
)

func (in UnitType) DeepCopy() (out UnitType) {
	return in
}

type UpgradeType string

const (
	UpgradeBeta  UpgradeType = "Beta"
	UpgradeBatch UpgradeType = "Batch"
)

func (in UpgradeType) DeepCopy() (out UpgradeType) {
	return in
}

type CafeDeploymentConditionType string

const (
	CafeDeploymentConditionTypeCellCreateFail CafeDeploymentConditionType = "CellProvisionFailure"
	CafeDeploymentConditionTypeCellDeleteFail CafeDeploymentConditionType = "CellReclaimFailure"
	CafeDeploymentConditionTypeScaleFail      CafeDeploymentConditionType = "ScaleFailure"
	CafeDeploymentConditionTypeRescheduleFail CafeDeploymentConditionType = "RescheduleFailure"
	CafeDeploymentConditionTypeReleaseFail    CafeDeploymentConditionType = "ReleaseFailure"
	CafeDeploymentConditionTypeRollbackFail   CafeDeploymentConditionType = "RollbackFailure"
)

func (in CafeDeploymentConditionType) DeepCopy() (out CafeDeploymentConditionType) {
	return in
}

type ReleaseProgress string

const (
	CafeDeploymentReleaseProgressWaitingForConfirmation ReleaseProgress = "WaitingForConfirmation"
	CafeDeploymentReleaseProgressExecuting              ReleaseProgress = "Executing"
	CafeDeploymentReleaseProgressCompleted              ReleaseProgress = "Completed"
	CafeDeploymentReleaseProgressAborted                ReleaseProgress = "Aborted"
)

func (in ReleaseProgress) DeepCopy() (out ReleaseProgress) {
	return in
}

type AutoScheduleProgress string

const (
	CafeDeploymentAutoRescheduleStatusRescheduling      AutoScheduleProgress = "Rescheduling"
	CafeDeploymentAutoRescheduleStatusCompleted         AutoScheduleProgress = "Completed"
	CafeDeploymentAutoRescheduleStatusNoUnitSchedulable AutoScheduleProgress = "NoUnitSchedulable"
)

func (in AutoScheduleProgress) DeepCopy() (out AutoScheduleProgress) {
	return in
}

type InPlaceSetConditionType string

const (
	InPlaceSetReplicaFailure    InPlaceSetConditionType = "ReplicaFailure"
	InPlaceSetUpgradeFailure    InPlaceSetConditionType = "UpgradeFailure"
	InPlaceSetPodUpgradeFailure InPlaceSetConditionType = "PodUpgradeFailure"
)

func (in InPlaceSetConditionType) DeepCopy() (out InPlaceSetConditionType) {
	return in
}
