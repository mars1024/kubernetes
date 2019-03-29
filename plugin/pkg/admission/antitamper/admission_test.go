package antitamper

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apiserver/pkg/admission"
	authenticationuser "k8s.io/apiserver/pkg/authentication/user"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/plugin/pkg/admission/antitamper/test"
	"sort"
	"strconv"
	"testing"
)

var fakeUser = &authenticationuser.DefaultInfo{
	Name:   "user",
	UID:    "user123",
	Groups: nil,
	Extra: map[string][]string{
		multitenancy.UserExtraInfoTenantID:    {"A"},
		multitenancy.UserExtraInfoWorkspaceID: {"A"},
		multitenancy.UserExtraInfoClusterID:   {"A"},
	},
}

func TestRegister(t *testing.T) {
	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()
	if len(registered) == 1 && registered[0] == PluginName {
		return
	} else {
		t.Errorf("Register failed")
	}
}

func TestFactory(t *testing.T) {
	i, err := Factory(nil)
	if i == nil || err != nil {
		t.Errorf("Factory failed")
	}
}

func TestNewAntiTamperAdmissionController(t *testing.T) {
	NewAntiTamperAdmissionController()
}


func TestValidateInitialization(t *testing.T) {
	handler := &AntiTamper{Handler: &admission.Handler{}}
	err := handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &AntiTamper{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	err = handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &AntiTamper{Handler: &admission.Handler{}}
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
	})
	err = handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &AntiTamper{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
	})
	err = handler.ValidateInitialization()
	if err != nil {
		t.Errorf("ValidateInitialization shouldn't fail")
	}
}

func makeAdmissionController(t *testing.T) *AntiTamper {
	cafeSystemReservedNamespaceNames = []string{"system-default", "anti-tamper-test-reserved"}

	handler := &AntiTamper{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
		FakeOptions: test.FakeOptions{UseNamespaceLister: test.FakeNamespaceLister{}},
	})
	err := handler.ValidateInitialization()
	if err != nil {
		t.Errorf("ValidateInitialization failed %v", err)
	}
	return handler
}

var AntiTamperTestPrefix = "anti-tamper-test-"

func TestAdminCanDoAnything(t *testing.T) {

	handler := makeAdmissionController(t)
	attributes := admission.NewAttributesRecord(nil, nil, schema.GroupVersionKind{}, "test", AntiTamperTestPrefix+"name", schema.GroupVersionResource{}, "", admission.Create, false,
		&authenticationuser.DefaultInfo{
			Name:   "system:admin",
			UID:    "",
			Groups: nil,
		})

	err := handler.Validate(attributes)
	if err != nil {
		t.Fatalf("wrong result")
	}

}

func TestTryingToUpdateImmutableLabelsOrAnnotations(t *testing.T) {
	tests := []struct {
		before      *v1.Pod
		after       *v1.Pod
		expectError bool
	}{
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			before:      makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A", "a": "a"}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A", "a": "b"}, map[string]string{}),
			expectError: false,
		},
		{
			before:      makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A"}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A"}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{"abc.cafe.sofastack.io/abc": "A"}, map[string]string{}),
			expectError: false,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{"system.sas.cafe.sofastack.io/abc": "A"}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{"abc.cafe.sofastack.io/abc": "A"}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			before:      makePod("a", "default", map[string]string{"system.sas.cafe.sofastack.io/abc": "A"}, map[string]string{}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "false"}),
			expectError: true,
		},
		{
			before:      makePod("a", "default", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true", "a": "a"}),
			after:       makePod("a", "default", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true", "b": "b"}),
			expectError: false,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildUpdatePodAttributes(test.before, test.after))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}
}

func TestTryingToUpdateImmutableLabelsOrAnnotations_Create(t *testing.T) {
	tests := []struct {
		resource    *v1.Pod
		expectError bool
	}{
		{
			resource:    makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			resource:    makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A"}, map[string]string{}),
			expectError: false,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildCreatePodAttributes(test.resource))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}
}

func TestTryingToUpdateImmutableLabelsOrAnnotations_Delete(t *testing.T) {
	tests := []struct {
		resource    *v1.Pod
		expectError bool
	}{
		{
			resource:    makePod("a", "default", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			resource:    makePod("a", "default", map[string]string{"cafe.sofastack.io/sub-cluster": "A"}, map[string]string{}),
			expectError: false,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildDeletePodAttributes(test.resource))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	tests := []struct {
		resource    *v1.Namespace
		expectError bool
	}{
		{
			resource:    makeNamespace("abc", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"}),
			expectError: true,
		},
		{
			resource:    makeNamespace("reserved", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			resource:    makeNamespace("innocent", map[string]string{}, map[string]string{}),
			expectError: false,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildCreateNamespaceAttributes(test.resource))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}
}

func TestUpdateNamespace(t *testing.T) {
	test.FakeNamespaceListerData[AntiTamperTestPrefix+"abc"] = makeNamespace("abc", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"})

	tests := []struct {
		before      *v1.Namespace
		after       *v1.Namespace
		expectError bool
	}{
		{
			before:      makeNamespace("abcd", map[string]string{}, map[string]string{"a": "a"}),
			after:       makeNamespace("abcd", map[string]string{}, map[string]string{"a": "b"}),
			expectError: false,
		},
		{
			before:      makeNamespace("abc", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"}),
			after:       makeNamespace("abc", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true", "a": "b"}),
			expectError: true,
		},
		{
			before:      makeNamespace("abcd", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "true"}),
			after:       makeNamespace("abcd", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			before:      makeNamespace("abcd", map[string]string{}, map[string]string{"cafe.sofastack.io/system-reserved-namespace": "false"}),
			after:       makeNamespace("abcd", map[string]string{}, map[string]string{}),
			expectError: true,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildUpdateNamespaceAttributes(test.before, test.after))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}

	test.FakeNamespaceListerData[AntiTamperTestPrefix+"abc"] = nil
}

func TestDeleteNamespace(t *testing.T) {

	tests := []struct {
		namespaceName string
		expectError   bool
	}{
		{
			namespaceName: "a",
			expectError:   false,
		},
		{
			namespaceName: "existing-reserved",
			expectError:   true,
		},
		{
			namespaceName: "reserved",
			expectError:   true,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildDeleteNamespaceAttributes(test.namespaceName))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}

	test.FakeNamespaceListerData[AntiTamperTestPrefix+"abc"] = nil
}

func TestCreateUpdateDeleteProtectedResource(t *testing.T) {
	protectedResources = []ResourceIdentifier{
		makeResourceIdentifier(AntiTamperTestPrefix+"abc", any, any, any, any),
		makeResourceIdentifier(AntiTamperTestPrefix+"abcd", "ns1", any, any, any),
		makeResourceIdentifier(AntiTamperTestPrefix+"abcde", "ns1", any, "v1", "Pod"),
		makeResourceIdentifier(AntiTamperTestPrefix+"abcdef", "ns1", any, "v1", "ConfigMap"),
		makeResourceIdentifier(AntiTamperTestPrefix+"abcdef-*", "ns*", any, "v1", "Pod"),
		makeResourceIdentifier(AntiTamperTestPrefix+"abcdefg", "*", "*", "*", "*"),
	}

	tests := []struct {
		pod         *v1.Pod
		expectError bool
	}{
		{
			pod:         makePod("a", "abc", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			pod:         makePod("abc", "abc", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("abcd", "abc", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			pod:         makePod("abcd", "ns1", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("abcde", "ns1", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("abcdef", "ns1", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			pod:         makePod("abcdef-aaaaa", "ns1", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("abcdef-aaaaa", "1ns1", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			pod:         makePod("abcdefg", "1ns1", map[string]string{}, map[string]string{}),
			expectError: true,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildCreatePodAttributes(test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
			err = handler.Validate(buildUpdatePodAttributes(test.pod, test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
			err = handler.Validate(buildDeletePodAttributes(test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}

	test.FakeNamespaceListerData[AntiTamperTestPrefix+"abc"] = nil
}

func TestCreateUpdateDeleteResourceOfProtectedNamespace(t *testing.T) {
	tests := []struct {
		pod         *v1.Pod
		expectError bool
	}{
		{
			pod:         makePod("xyz", "abc", map[string]string{}, map[string]string{}),
			expectError: false,
		},
		{
			pod:         makePod("xyz", AntiTamperTestPrefix+"reserved", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("xyz", "system-default", map[string]string{}, map[string]string{}),
			expectError: true,
		},
		{
			pod:         makePod("xyz", AntiTamperTestPrefix+"existing-reserved", map[string]string{}, map[string]string{}),
			expectError: true,
		},
	}

	handler := makeAdmissionController(t)

	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			err := handler.Validate(buildCreatePodAttributes(test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
			err = handler.Validate(buildUpdatePodAttributes(test.pod, test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
			err = handler.Validate(buildDeletePodAttributes(test.pod))
			if (err != nil) != test.expectError {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}

	test.FakeNamespaceListerData[AntiTamperTestPrefix+"abc"] = nil
}


func makeNamespace(name string, labels map[string]string, annotations map[string]string) *v1.Namespace {
	return &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        AntiTamperTestPrefix + name,
			Namespace:   "",
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

func buildCreateNamespaceAttributes(namespace *v1.Namespace) admission.Attributes {
	namespaceKind := api.Kind("Namespace").WithVersion("v1")

	return admission.NewAttributesRecord(namespace, nil, namespaceKind, "", namespace.Name,
		schema.GroupVersionResource{}, "",
		admission.Create, false,
		fakeUser)
}

func buildUpdateNamespaceAttributes(before *v1.Namespace, after *v1.Namespace) admission.Attributes {
	namespaceKind := api.Kind("Namespace").WithVersion("v1")

	return admission.NewAttributesRecord(after, before, namespaceKind, "", after.Name,
		schema.GroupVersionResource{}, "",
		admission.Update, false,
		fakeUser)
}

func buildDeleteNamespaceAttributes(namespaceName string) admission.Attributes {
	namespaceKind := api.Kind("Namespace").WithVersion("v1")

	return admission.NewAttributesRecord(nil, nil, namespaceKind, "", AntiTamperTestPrefix+namespaceName,
		schema.GroupVersionResource{}, "",
		admission.Delete, false,
		fakeUser)
}

func buildCreatePodAttributes(pod *v1.Pod) admission.Attributes {
	podKind := api.Kind("Pod").WithVersion("v1")

	return admission.NewAttributesRecord(pod, nil, podKind, pod.Namespace, pod.Name,
		schema.GroupVersionResource{}, "",
		admission.Create, false,
		fakeUser)
}

func buildUpdatePodAttributes(before *v1.Pod, after *v1.Pod) admission.Attributes {
	podKind := api.Kind("Pod").WithVersion("v1")

	return admission.NewAttributesRecord(after, before, podKind, after.Namespace, after.Name,
		schema.GroupVersionResource{}, "",
		admission.Update, false,
		fakeUser)
}

func buildDeletePodAttributes(pod *v1.Pod) admission.Attributes {
	podKind := api.Kind("Pod").WithVersion("v1")

	return admission.NewAttributesRecord(nil, nil, podKind, pod.Namespace, pod.Name,
		schema.GroupVersionResource{}, "",
		admission.Delete, false,
		fakeUser)
}

func makePod(name string, namespace string, labels map[string]string, annotations map[string]string) *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        AntiTamperTestPrefix + name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

// ========== utils test

func TestIsMultiTenancyWiseAdmin(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{name: "system:admin", expected: true},
		{name: "system:apiserver", expected: true},
		{name: "abc", expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if IsMultiTenancyWiseAdmin(test.name) != test.expected {
				t.Fatalf("wrong result with %s", test.name)
			}
		})
	}
}

func TestPassesAnyRule(t *testing.T) {
	tests := []struct {
		item        string
		filterRules []string
		expected    bool
	}{
		{
			item:        "abc",
			filterRules: []string{"*"},
			expected:    true,
		},
		{
			item:        "abc",
			filterRules: []string{"abc"},
			expected:    true,
		},
		{
			item:        "abc",
			filterRules: []string{"x"},
			expected:    false,
		},
		{
			item:        "abc",
			filterRules: []string{"x", "*"},
			expected:    true,
		},
		{
			item:        "abc",
			filterRules: []string{"a*"},
			expected:    true,
		},
		{
			item:        "abc",
			filterRules: []string{"*bc"},
			expected:    true,
		},
		{
			item:        "abc",
			filterRules: []string{"*a"},
			expected:    false,
		},
		{
			item:        "abc",
			filterRules: []string{"bc*"},
			expected:    false,
		},
		{
			item:        "abc",
			filterRules: []string{"x", "y", "ab", "bc"},
			expected:    false,
		},
	}
	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			if passesAnyRule(test.item, test.filterRules) != test.expected {
				t.Fatalf("wrong result with %s and %v", test.item, test.filterRules)
			}
		})
	}
}

func TestDoFilter(t *testing.T) {
	tests := []struct {
		items       []string
		filterRules []string
		expected    []string
	}{
		{
			items:       []string{"abc", "def"},
			filterRules: []string{"*"},
			expected:    []string{"abc", "def"},
		},
		{
			items:       []string{"abc", "def"},
			filterRules: []string{"abc"},
			expected:    []string{"abc"},
		},
		{
			items:       []string{"abc", "def"},
			filterRules: []string{"ab"},
			expected:    []string{},
		},
		{
			items:       []string{"test", "group1/abc", "group1/def", "group2/abc", "group2/def",},
			filterRules: []string{"test", "group1/*"},
			expected:    []string{"test", "group1/abc", "group1/def"},
		},
	}
	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			if !setEqual(doFilter(test.items, test.filterRules), test.expected) {
				t.Fatalf("wrong result with %s and %v, got %v",
					test.items, test.filterRules, doFilter(test.items, test.filterRules))
			}
		})
	}
}

func TestGetChangedKeys(t *testing.T) {
	tests := []struct {
		mapA     map[string]string
		mapB     map[string]string
		expected []string
	}{
		{
			mapA: map[string]string{
				"a": "a",
				"b": "a",
			},
			mapB: map[string]string{
				"a": "a",
				"b": "a",},
			expected: []string{},
		},
		{
			mapA: map[string]string{
				"a": "c",
				"b": "a",
			},
			mapB: map[string]string{
				"a": "a",
				"b": "a",},
			expected: []string{"a"},
		},
		{
			mapA: map[string]string{
				"a": "a",
				"b": "a",
			},
			mapB: map[string]string{
				"a": "c",
				"b": "a",},
			expected: []string{"a"},
		},
		{
			mapA: map[string]string{
				"a": "c",
				"b": "a",
			},
			mapB: map[string]string{
				"a": "a",
				"b": "c",
			},
			expected: []string{"a", "b"},
		},
		{
			mapA: map[string]string{
				"a": "a",
			},
			mapB: map[string]string{
				"a": "a",
				"b": "c",},
			expected: []string{"b"},
		},
		{
			mapA: map[string]string{
				"a": "a",
				"b": "a",
			},
			mapB: map[string]string{
				"a": "a",
			},
			expected: []string{"b"},
		},
	}
	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			if !setEqual(getChangedKeys(test.mapA, test.mapB), test.expected) {
				t.Fatalf("wrong result with %v and %v, got %v",
					test.mapA, test.mapB, getChangedKeys(test.mapA, test.mapB))
			}
		})
	}
}

func TestMatchesResourceIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		group      string
		version    string
		kind       string
		identifier ResourceIdentifier
		expected   bool
	}{
		{
			name:       "A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier(any, any, any, any, any),
			expected:   true,
		},
		{
			name:       "A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("A", "A", "A", "A", "A"),
			expected:   true,
		},
		{
			name:       "A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("A", "A", "A", "A", "B"),
			expected:   false,
		},
		{
			name:       "A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("A", "A", "A", "A", any),
			expected:   true,
		},
		{
			name:       "test-A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("test-*", any, any, any, any),
			expected:   true,
		},
		{
			name:       "a-test-A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("test-*", any, any, any, any),
			expected:   false,
		},
		{
			name:       "a-test-A",
			namespace:  "A",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("*test-A", any, any, any, any),
			expected:   true,
		},
		{
			name:       "a-test-A",
			namespace:  "ABC",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("*test-A", "B", any, any, any),
			expected:   false,
		},
		{
			name:       "a-test-A",
			namespace:  "ABC",
			group:      "A",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("*test-A", "A*", any, any, any),
			expected:   true,
		},
		{
			name:       "a-test-A",
			namespace:  "ABC",
			group:      "",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("*test-A", "A*", "A", any, any),
			expected:   false,
		},
		{
			name:       "a-test-A",
			namespace:  "ABC",
			group:      "",
			version:    "A",
			kind:       "A",
			identifier: makeResourceIdentifier("*test-A", "A*", any, any, any),
			expected:   true,
		},
	}
	for i, test := range tests {
		t.Run("test_"+strconv.Itoa(i), func(t *testing.T) {
			if matchesResourceIdentifier(test.group, test.version, test.kind, test.name, test.namespace, test.identifier) != test.expected {
				t.Fatalf("wrong result with %v", test)
			}
		})
	}
}

func TestVerboseLogIfNecessary(t *testing.T) {
	verboseLogIfNecessary("ABC", "anti-tamper-test-a")
}

func setEqual(setA []string, setB []string) bool {
	if len(setA) == 0 && len(setB) == 0 {
		return true
	}
	sort.Strings(setA)
	sort.Strings(setB)
	jsonA, _ := json.Marshal(setA)
	jsonB, _ := json.Marshal(setB)
	return string(jsonA) == string(jsonB)
}