package v1alpha1

func GetCondition(status MinionClusterStatus, condType MinionClusterConditionType) *MinionClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition adds/replaces the given condition in the replication controller status.
func SetCondition(status *MinionClusterStatus, condition MinionClusterCondition) {
	currentCond := GetCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

func RemoveCondition(status MinionClusterStatus, condType MinionClusterConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

func filterOutCondition(conditions []MinionClusterCondition, condType MinionClusterConditionType) []MinionClusterCondition {
	var newConditions []MinionClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
