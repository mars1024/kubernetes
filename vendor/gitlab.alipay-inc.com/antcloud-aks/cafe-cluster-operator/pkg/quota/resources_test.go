package quota

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/api/core/v1"
)

func TestEquals(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		b        v1.ResourceList
		expected bool
	}{
		"isEqual": {
			a:        v1.ResourceList{},
			b:        v1.ResourceList{},
			expected: true,
		},
		"isEqualWithKeys": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			b: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expected: true,
		},
		"isNotEqualSameKeys": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("200m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			b: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expected: false,
		},
		"isNotEqualDiffKeys": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			b: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
				v1.ResourcePods:   resource.MustParse("1"),
			},
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		if result := Equals(testCase.a, testCase.b); result != testCase.expected {
			t.Errorf("%s expected: %v, actual: %v, a=%v, b=%v", testName, testCase.expected, result, testCase.a, testCase.b)
		}
	}
}

func TestAdd(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		b        v1.ResourceList
		expected v1.ResourceList
	}{
		"noKeys": {
			a:        v1.ResourceList{},
			b:        v1.ResourceList{},
			expected: v1.ResourceList{},
		},
		"toEmpty": {
			a:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			b:        v1.ResourceList{},
			expected: v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
		},
		"matching": {
			a:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			b:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			expected: v1.ResourceList{v1.ResourceCPU: resource.MustParse("200m")},
		},
	}
	for testName, testCase := range testCases {
		sum := Add(testCase.a, testCase.b)
		if result := Equals(testCase.expected, sum); !result {
			t.Errorf("%s expected: %v, actual: %v", testName, testCase.expected, sum)
		}
	}
}

func TestSubtract(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		b        v1.ResourceList
		expected v1.ResourceList
	}{
		"noKeys": {
			a:        v1.ResourceList{},
			b:        v1.ResourceList{},
			expected: v1.ResourceList{},
		},
		"value-empty": {
			a:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			b:        v1.ResourceList{},
			expected: v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
		},
		"empty-value": {
			a:        v1.ResourceList{},
			b:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			expected: v1.ResourceList{v1.ResourceCPU: resource.MustParse("-100m")},
		},
		"value-value": {
			a:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("200m")},
			b:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
			expected: v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
		},
	}
	for testName, testCase := range testCases {
		sub := Subtract(testCase.a, testCase.b)
		if result := Equals(testCase.expected, sub); !result {
			t.Errorf("%s expected: %v, actual: %v", testName, testCase.expected, sub)
		}
	}
}

func TestResourceNames(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		expected []v1.ResourceName
	}{
		"empty": {
			a:        v1.ResourceList{},
			expected: []v1.ResourceName{},
		},
		"values": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expected: []v1.ResourceName{v1.ResourceMemory, v1.ResourceCPU},
		},
	}
	for testName, testCase := range testCases {
		actualSet := ToSet(ResourceNames(testCase.a))
		expectedSet := ToSet(testCase.expected)
		if !actualSet.Equal(expectedSet) {
			t.Errorf("%s expected: %v, actual: %v", testName, expectedSet, actualSet)
		}
	}
}

func TestIsZero(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		expected bool
	}{
		"empty": {
			a:        v1.ResourceList{},
			expected: true,
		},
		"zero": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("0"),
				v1.ResourceMemory: resource.MustParse("0"),
			},
			expected: true,
		},
		"non-zero": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("200m"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expected: false,
		},
	}
	for testName, testCase := range testCases {
		if result := IsZero(testCase.a); result != testCase.expected {
			t.Errorf("%s expected: %v, actual: %v", testName, testCase.expected, result)
		}
	}
}

func TestIsNegative(t *testing.T) {
	testCases := map[string]struct {
		a        v1.ResourceList
		expected []v1.ResourceName
	}{
		"empty": {
			a:        v1.ResourceList{},
			expected: []v1.ResourceName{},
		},
		"some-negative": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("-10"),
				v1.ResourceMemory: resource.MustParse("0"),
			},
			expected: []v1.ResourceName{v1.ResourceCPU},
		},
		"all-negative": {
			a: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("-200m"),
				v1.ResourceMemory: resource.MustParse("-1Gi"),
			},
			expected: []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory},
		},
	}
	for testName, testCase := range testCases {
		actual := IsNegative(testCase.a)
		actualSet := ToSet(actual)
		expectedSet := ToSet(testCase.expected)
		if !actualSet.Equal(expectedSet) {
			t.Errorf("%s expected: %v, actual: %v", testName, expectedSet, actualSet)
		}
	}
}
