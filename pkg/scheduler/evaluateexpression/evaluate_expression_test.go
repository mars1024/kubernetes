package evaluateexpression

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/core/v1"
	"testing"
)

var defaultNode = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Annotations: map[string]string{
			"ase.cloud.alipay.com/my-annotation": "123",
			"aaa":                                "true",
			"pi":                                 "3.14",
			"f":                                  "3.18",
			"text":                               "text",
		},
		Labels: map[string]string{
			"ase.cloud.alipay.com/my-label": "789",
		},
	},
}

var defaultPod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Annotations: map[string]string{
			"ase.cloud.alipay.com/my-annotation": "321",
		},
		Labels: map[string]string{
			"ase.cloud.alipay.com/my-label": "456",
		},
	},
}

func TestCustomExpression(t *testing.T) {
	tests := []struct {
		Expression    string
		ExpectedValue interface{}
		ExpectError   bool
		Node          *v1.Node
		Pod           *v1.Pod
	}{
		{
			Expression:  "pod.labels['abc'] / 2",
			ExpectError: true,
		},
		{
			Expression:  "abc",
			ExpectError: true,
		},
		{
			Expression:  "unknownFunc()",
			ExpectError: true,
		},
		{
			Expression:    "true",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Expression:    "0",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:    "pod.labels['abc']",
			ExpectedValue: "",
			ExpectError:   false,
		},
		{
			Expression:    "node.annotations['ase.cloud.alipay.com/my-annotation']",
			ExpectedValue: 123,
			ExpectError:   false,
		},
		{
			Expression:    "node.labels['ase.cloud.alipay.com/my-label']",
			ExpectedValue: 789,
			ExpectError:   false,
		},
		{
			Expression:    "pod.annotations['ase.cloud.alipay.com/my-annotation']",
			ExpectedValue: 321,
			ExpectError:   false,
		},
		{
			Expression:    "pod.labels['ase.cloud.alipay.com/my-label']",
			ExpectedValue: 456,
			ExpectError:   false,
		},
		{
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"a": "true",
					},
				},
			},
			Expression:    "pod.labels['a']",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"a": "false",
					},
				},
			},
			Expression:    "pod.labels['a']",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"a": "abc",
					},
				},
			},
			Expression:    "pod.labels['a']",
			ExpectedValue: "abc",
			ExpectError:   false,
		},

		// --- strlen

		{
			Expression:    "strlen(\"abc\")",
			ExpectedValue: 3,
			ExpectError:   false,
		},
		{
			Expression:    "strlen(5)",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:  "strlen()",
			ExpectError: true,
		},

		// --- round

		{
			Expression:    "round(3.2)",
			ExpectedValue: 3,
			ExpectError:   false,
		},
		{
			Expression:    "round(3.6)",
			ExpectedValue: 4,
			ExpectError:   false,
		},
		{
			Expression:    "round(false)",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:  "round()",
			ExpectError: true,
		},

		// --- floor

		{
			Expression:    "floor(3.2)",
			ExpectedValue: 3,
			ExpectError:   false,
		},
		{
			Expression:    "floor(false)",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:  "floor()",
			ExpectError: true,
		},

		// --- ceil

		{
			Expression:    "ceil(3.2)",
			ExpectedValue: 4,
			ExpectError:   false,
		},
		{
			Expression:    "ceil(false)",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:  "ceil()",
			ExpectError: true,
		},

		// --- toFixed

		{
			Expression:    "toFixed(3.2, 2)",
			ExpectedValue: "3.20",
			ExpectError:   false,
		},
		{
			Expression:  "toFixed(2,\"2\")",
			ExpectError: true,
		},
		{
			Expression:  "toFixed(\"2\",2)",
			ExpectError: true,
		},
		{
			Expression:  "toFixed(2)",
			ExpectError: true,
		},
		{
			Expression:  "toFixed()",
			ExpectError: true,
		},

		// --- number

		{
			Expression:    "number(3.2)",
			ExpectedValue: 3.2,
			ExpectError:   false,
		},
		{
			Expression:    "number(\"3.2\")",
			ExpectedValue: 3.2,
			ExpectError:   false,
		},
		{
			Expression:    "number(\"\")",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:    "number(true)",
			ExpectedValue: 1,
			ExpectError:   false,
		},
		{
			Expression:    "number(false)",
			ExpectedValue: 0,
			ExpectError:   false,
		},
		{
			Expression:  "number(\"a\")",
			ExpectError: true,
		},
		{
			Expression:  "number()",
			ExpectError: true,
		},


		// --- string

		{
			Expression:    "string(3.2)",
			ExpectedValue: "3.2",
			ExpectError:   false,
		},
		{
			Expression:    "string(\"3.2\")",
			ExpectedValue: "3.2",
			ExpectError:   false,
		},
		{
			Expression:    "string(\"\")",
			ExpectedValue: "",
			ExpectError:   false,
		},
		{
			Expression:    "string(true)",
			ExpectedValue: "true",
			ExpectError:   false,
		},
		{
			Expression:    "string(false)",
			ExpectedValue: "false",
			ExpectError:   false,
		},
		{
			Expression:  "string()",
			ExpectError: true,
		},

		// --- bool

		{
			Expression:    "bool(3.2)",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Expression:    "bool(0)",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Expression:    "bool(\"true\")",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Expression:    "bool(\"false\")",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Expression:    "bool(\"\")",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Expression:    "bool(\"1\")",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Expression:    "bool(\"0\")",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Expression:    "bool(true)",
			ExpectedValue: true,
			ExpectError:   false,
		},
		{
			Expression:    "bool(false)",
			ExpectedValue: false,
			ExpectError:   false,
		},
		{
			Expression:  "bool()",
			ExpectError: true,
		},
	}

	for i, tc := range tests {
		node := defaultNode
		if tc.Node != nil {
			node = tc.Node
		}

		pod := defaultPod
		if tc.Pod != nil {
			pod = tc.Pod
		}

		result, err := EvaluateExpression(tc.Expression, node, pod)

		if tc.ExpectError && err == nil {
			t.Errorf("tc %d: expected error but got no error", i)
			continue
		}

		if !tc.ExpectError && err != nil {
			t.Errorf("tc %d: unexpected error %v", i, err)
			continue
		}

		if tc.ExpectError {
			continue
		}

		match := false
		switch tc.ExpectedValue.(type) {
		case string:
			match = result.(string) == tc.ExpectedValue.(string)
			break
		case int:
			match = result.(float64) == float64(tc.ExpectedValue.(int))
			break
		case bool:
			match = result.(bool) == tc.ExpectedValue.(bool)
			break
		case float64:
			match = result.(float64) == tc.ExpectedValue.(float64)
			break
		}

		if !match {
			t.Errorf("tc %d: expected %v but got %v", i, tc.ExpectedValue, result)
			continue
		}
	}

}
