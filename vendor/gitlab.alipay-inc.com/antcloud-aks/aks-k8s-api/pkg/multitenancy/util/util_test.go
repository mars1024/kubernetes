package util

import (
	"testing"
	"reflect"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

func TestTransformTenantInfoToAnnotationsIncremental(t *testing.T) {
	tenant := multitenancy.NewTenantInfo("t1", "w1", "c1")
	testCases := []struct {
		annotation map[string]string
	}{
		{
			nil,
		},
		{
			make(map[string]string),
		},
		{
			map[string]string{
				multitenancy.MultiTenancyAnnotationKeyClusterID: "foo",
			},
		},
	}
	for _, testCase := range testCases {
		TransformTenantInfoToAnnotationsIncremental(tenant, &testCase.annotation)
		expectedAnnotation := map[string]string{
			multitenancy.MultiTenancyAnnotationKeyTenantID:    "t1",
			multitenancy.MultiTenancyAnnotationKeyWorkspaceID: "w1",
			multitenancy.MultiTenancyAnnotationKeyClusterID:   "c1",
		}
		if !reflect.DeepEqual(testCase.annotation, expectedAnnotation) {
			t.Errorf("transform annotation mismatched")
		}
	}

}
