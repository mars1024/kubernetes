package sigma

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSNFromLabel(t *testing.T) {
	snStr := "12345678"
	for caseName, testCase := range map[string]struct {
		testPod    *v1.Pod
		expectedSN string
	}{

		"pod has sn": {
			testPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "default",
					Labels:    map[string]string{sigmak8sapi.LabelPodSn: snStr},
				},
			},
			expectedSN: snStr,
		},
		"pod doesn't have sn": {
			testPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "default",
					Labels:    map[string]string{"sigma.ali/sn-fake": snStr},
				},
			},
			expectedSN: "",
		},
		"pod doesn't have label": {
			testPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "default",
				},
			},
			expectedSN: "",
		},
	} {
		t.Logf("Start to test testcase: %s", caseName)
		sn, _ := GetSNFromLabel(testCase.testPod)
		assert.Equal(t, testCase.expectedSN, sn, caseName)
	}
}
