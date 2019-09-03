package quota

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

)

// Equals returns true if the two lists are equivalent
func Equals(a v1.ResourceList, b v1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}

	for key, value1 := range a {
		value2, found := b[key]
		if !found {
			return false
		}
		if value1.Cmp(value2) != 0 {
			return false
		}
	}

	return true
}

// Add returns the result of a + b for each named resource
func Add(a v1.ResourceList, b v1.ResourceList) v1.ResourceList {
	result := v1.ResourceList{}
	for key, value := range a {
		quantity := *value.Copy()
		if other, found := b[key]; found {
			quantity.Add(other)
		}
		result[key] = quantity
	}
	for key, value := range b {
		if _, found := result[key]; !found {
			quantity := *value.Copy()
			result[key] = quantity
		}
	}
	return result
}

// ToSet takes a list of resource names and converts to a string set
func ToSet(resourceNames []v1.ResourceName) sets.String {
	result := sets.NewString()
	for _, resourceName := range resourceNames {
		result.Insert(string(resourceName))
	}
	return result
}

// Intersection returns the intersection of both list of resources
func Intersection(a []v1.ResourceName, b []v1.ResourceName) []v1.ResourceName {
	setA := ToSet(a)
	setB := ToSet(b)
	setC := setA.Intersection(setB)
	var result []v1.ResourceName
	for _, resourceName := range setC.List() {
		result = append(result, v1.ResourceName(resourceName))
	}
	return result
}

// LessThanOrEqual returns true if a < b for each key in b
// If false, it returns the keys in a that exceeded b
func LessThanOrEqual(a v1.ResourceList, b v1.ResourceList) (bool, []v1.ResourceName) {
	result := true
	var resourceNames []v1.ResourceName
	for key, value := range b {
		if other, found := a[key]; found {
			if other.Cmp(value) > 0 {
				result = false
				resourceNames = append(resourceNames, key)
			}
		}
	}
	return result, resourceNames
}

// Subtract returns the result of a - b for each named resource
func Subtract(a v1.ResourceList, b v1.ResourceList) v1.ResourceList {
	result := v1.ResourceList{}
	for key, value := range a {
		quantity := *value.Copy()
		if other, found := b[key]; found {
			quantity.Sub(other)
		}
		result[key] = quantity
	}
	for key, value := range b {
		if _, found := result[key]; !found {
			quantity := *value.Copy()
			quantity.Neg()
			result[key] = quantity
		}
	}
	return result
}

// Mask returns a new resource list that only has the values with the specified names
func Mask(resources v1.ResourceList, names []v1.ResourceName) v1.ResourceList {
	nameSet := ToSet(names)
	result := v1.ResourceList{}
	for key, value := range resources {
		if nameSet.Has(string(key)) {
			result[key] = *value.Copy()
		}
	}
	return result
}

// ResourceNames returns a list of all resource names in the ResourceList
func ResourceNames(resources v1.ResourceList) []v1.ResourceName {
	var result []v1.ResourceName
	for resourceName := range resources {
		result = append(result, resourceName)
	}
	return result
}

// IsZero returns true if each key maps to the quantity value 0
func IsZero(a v1.ResourceList) bool {
	zero := resource.MustParse("0")
	for _, v := range a {
		if v.Cmp(zero) != 0 {
			return false
		}
	}
	return true
}

// IsNegative returns the set of resource names that have a negative value.
func IsNegative(a v1.ResourceList) []v1.ResourceName {
	var results []v1.ResourceName
	zero := resource.MustParse("0")
	for k, v := range a {
		if v.Cmp(zero) < 0 {
			results = append(results, k)
		}
	}
	return results
}

// SubtractWithNonNegativeResult - subtracts and returns result of a - b but
// makes sure we don't return negative values to prevent negative resource usage.
func SubtractWithNonNegativeResult(a v1.ResourceList, b v1.ResourceList) v1.ResourceList {
	zero := resource.MustParse("0")

	result := v1.ResourceList{}
	for key, value := range a {
		quantity := *value.Copy()
		if other, found := b[key]; found {
			quantity.Sub(other)
		}
		if quantity.Cmp(zero) > 0 {
			result[key] = quantity
		} else {
			result[key] = zero
		}
	}

	for key := range b {
		if _, found := result[key]; !found {
			result[key] = zero
		}
	}
	return result
}

// CalculateUsage calculates and returns the requested ResourceList usage
func CalculateUsage(namespaceName string, scopes []v1.ResourceQuotaScope, hardLimits v1.ResourceList, registry Registry, scopeSelector *v1.ScopeSelector) (v1.ResourceList, error) {
	// find the intersection between the hard resources on the quota
	// and the resources this controller can track to know what we can
	// look to measure updated usage stats for
	hardResources := ResourceNames(hardLimits)
	var potentialResources []v1.ResourceName
	evaluators := registry.List()
	for _, evaluator := range evaluators {
		matchingResources := evaluator.MatchingResources(hardResources)
		potentialResources = append(potentialResources, matchingResources...)
	}
	// NOTE: the intersection just removes duplicates since the evaluator match intersects with hard
	matchedResources := Intersection(hardResources, potentialResources)

	// sum the observed usage from each evaluator
	newUsage := v1.ResourceList{}
	for _, evaluator := range evaluators {
		// only trigger the evaluator if it matches a resource in the quota, otherwise, skip calculating anything
		intersection := evaluator.MatchingResources(matchedResources)
		if len(intersection) == 0 {
			continue
		}

		usageStatsOptions := UsageStatsOptions{Namespace: namespaceName, Scopes: scopes, Resources: intersection, ScopeSelector: scopeSelector}
		stats, err := evaluator.UsageStats(usageStatsOptions)
		if err != nil {
			return nil, err
		}
		newUsage = Add(newUsage, stats.Used)
	}

	// mask the observed usage to only the set of resources tracked by this quota
	// merge our observed usage with the quota usage status
	// if the new usage is different than the last usage, we will need to do an update
	newUsage = Mask(newUsage, matchedResources)
	return newUsage, nil
}
