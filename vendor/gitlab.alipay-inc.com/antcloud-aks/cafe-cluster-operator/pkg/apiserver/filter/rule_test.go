package filter

import (
	"testing"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestMatchBinding(t *testing.T) {
	username := "martin"
	usergroups := []string{"student", "assistant", "coder"}
	namespace := "default"
	name := "foo"
	verb := "GET"
	apigroup := "example.com"
	apiversion := "v1"
	resource := "test"
	subresource := "sub_test"

	sub := &subject{
		username:    username,
		usergroups:  usergroups,
		namespace:   namespace,
		name:        name,
		verb:        verb,
		apigroup:    apigroup,
		apiversion:  apiversion,
		resource:    resource,
		subresource: subresource,
	}

	testCases := []struct {
		name          string
		binding       *v1alpha1.BucketBinding
		expectedMatch bool
	}{
		{
			name: "empty values will match nothing",
			binding: &v1alpha1.BucketBinding{
				Spec: v1alpha1.BucketBindingSpec{
					Rules: []*v1alpha1.BucketBindingRule{
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserName,
							Values: []string{},
						},
					},
				},
			},
			expectedMatch: false,
		},
		{
			name: "normal username match",
			binding: &v1alpha1.BucketBinding{
				Spec: v1alpha1.BucketBindingSpec{
					Rules: []*v1alpha1.BucketBindingRule{
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserName,
							Values: []string{username, "duck"},
						},
					},
				},
			},
			expectedMatch: true,
		},
		{
			name: "normal user groups match",
			binding: &v1alpha1.BucketBinding{
				Spec: v1alpha1.BucketBindingSpec{
					Rules: []*v1alpha1.BucketBindingRule{
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserGroup,
							Values: usergroups,
						},
					},
				},
			},
			expectedMatch: true,
		},
		{
			name: "username match but group doesn't match",
			binding: &v1alpha1.BucketBinding{
				Spec: v1alpha1.BucketBindingSpec{
					Rules: []*v1alpha1.BucketBindingRule{
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserName,
							Values: []string{username},
						},
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserGroup,
							Values: []string{"na"},
						},
					},
				},
			},
			expectedMatch: false,
		},
		{
			name: "username match but verb doesn't match",
			binding: &v1alpha1.BucketBinding{
				Spec: v1alpha1.BucketBindingSpec{
					Rules: []*v1alpha1.BucketBindingRule{
						{
							Field:  v1alpha1.BucketBindingSubjectFieldUserName,
							Values: []string{username},
						},
						{
							Field:  v1alpha1.BucketBindingSubjectFieldRequestVerb,
							Values: []string{"na"},
						},
					},
				},
			},
			expectedMatch: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedMatch, match(sub, testCase.binding))
		})
	}

}
