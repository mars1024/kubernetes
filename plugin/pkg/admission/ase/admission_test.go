package ase

import (
	"encoding/json"
	"errors"
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/plugin/pkg/admission/ase/test"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

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

func TestNewAseAdmissionController(t *testing.T) {
	NewAseAdmissionController()
}

type FakeConfigMapLister struct {
}

type FakeHttpClient struct {
	http.Client
}

func (c *FakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{Body: &http.NoBody, Status: fakeHttpResponseStatus}, nil
}

var configMapListResult []*v1.ConfigMap
var fakeHttpResponseStatus string

var testClusterId = "0000000000000001"
var testClusterName = "testname"

var jsonManagedClusterInfoNonMatch, _ = json.Marshal(map[string]string{
	"clusterId":                "123",
	"clusterName":              "123",
	"clusterTenantId":          "123",
	"clusterTenantName":        "123",
	"clusterWorkspaceId":       "123",
	"clusterWorkspaceIdentity": "123",
	"clusterRegionId":          "123",
})
var jsonManagedClusterInfoMatch, _ = json.Marshal(map[string]interface{}{
	"clusterId":                testClusterId,
	"clusterName":              testClusterName,
	"clusterTenantId":          "123",
	"clusterTenantName":        "123",
	"clusterWorkspaceId":       "123",
	"clusterWorkspaceIdentity": "123",
	"clusterRegionId":          "123",
	"managedSubClusters":       []ManagedSubCluster{{SubClusterName: "abc"}},
})
var jsonSchedulerConfig, _ = json.Marshal(map[string]interface{}{
	"predicates":           []interface{}{},
	"priorities":           []interface{}{},
	"cpuOverScheduleRatio": 2,
	"hardCpuOverSchedule":  true,
})

var logtailImage = "logtail image"
var logtailUserDefinedId = "logtail user defined id"

var logContext, _ = json.Marshal(map[string]string{
	"userId":         "ABC",
	"logProjectName": "ABC",
	"logStoreName":   "ABC",
	"config":         "ABC",
	"tenantId":       "ABC",
	"image":          logtailImage,
})

var logContextWithUserDefinedId, _ = json.Marshal(map[string]string{
	"userId":         "ABC",
	"logProjectName": "ABC",
	"logStoreName":   "ABC",
	"userDefinedId":  logtailUserDefinedId,
	"config":         "ABC",
	"tenantId":       "ABC",
	"image":          logtailImage,
})

var logContextWithVolumeMountConfig, _ = json.Marshal(map[string]interface{}{
	"userId":         "ABC",
	"logProjectName": "ABC",
	"logStoreName":   "ABC",
	"config":         "ABC",
	"tenantId":       "ABC",
	"image":          logtailImage,
	"volumeMountConfigs": []VolumeMountConfigItem{
		{MountAs: "a", VolumeName: "name-a"},
		{MountAs: "b", VolumeName: "name-b"},
		{MountAs: "../c/", VolumeName: "name-c"},
	},
})

var configMapListResultValidConfigMapNonMatchClusterId = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoNonMatch)},
	},
}

var configMapListResultValidConfigMapMatchClusterId = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
}

var configMapListResultWithLogConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-pod-log-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{
			"config": string(configMapListResultWithLogConfigData),
		},
	},
}

var configMapListResultWithLogConfigWithVolumeMountConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-pod-log-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{
			"config": string(configMapListResultWithLogConfigDataWithVolumeMountConfig),
		},
	},
}

var configMapListResultWithLogConfigAndLogAgentRequestedResourceData, _ = json.Marshal(map[string]interface{}{
	"defaultLogUserId":  "defaultLogUserId",
	"defaultLogProject": "defaultLogProject",
	"defaultLogStore":   "defaultLogStore",
	"defaultLogConfig":  "defaultLogConfig",
	"defaultImage":      "defaultImage",
	"logAgentRequestedResource": map[string]string{
		"cpu":     "200m",
		"memory":  "10G",
		"storage": "100G",
	},
})

var configMapListResultWithLogConfigData, _ = json.Marshal(map[string]string{
	"defaultLogUserId":  "defaultLogUserId",
	"defaultLogProject": "defaultLogProject",
	"defaultLogStore":   "defaultLogStore",
	"defaultLogConfig":  "defaultLogConfig",
	"defaultImage":      "defaultImage",
})

var configMapListResultWithLogConfigDataWithVolumeMountConfig, _ = json.Marshal(map[string]interface{}{
	"defaultLogUserId":  "defaultLogUserId",
	"defaultLogProject": "defaultLogProject",
	"defaultLogStore":   "defaultLogStore",
	"defaultLogConfig":  "defaultLogConfig",
	"defaultImage":      "defaultImage",
	"defaultVolumeMountConfigs": []VolumeMountConfigItem{
		{MountAs: "a", VolumeName: "name-a"},
		{MountAs: "b", VolumeName: "name-b"},
	},
})

var configMapListResultWithLogConfigAndLogAgentRequestedResource = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-pod-log-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{
			"config": string(configMapListResultWithLogConfigAndLogAgentRequestedResourceData),
		},
	},
}

var imageConfigData, _ = json.Marshal(map[string]string{"checkImagePermissionUrl": "http://"})
var emptyImageConfigData, _ = json.Marshal(map[string]string{})

var configMapListResultWithImageConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-image-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(imageConfigData)},
	},
}

var configMapListResultWithEmptyImageConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-image-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(emptyImageConfigData)},
	},
}

var configMapListResultWithNodeGroupConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "aks-sub-cluster-node-group-config-abc-ng",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": "{}"},
	},
}

var configMapListResultInvalidConfigMap = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "a",
			Annotations: map[string]string{
			},
			Labels: map[string]string{LabelCluster: testClusterName},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
	},
}

var configMapListResultNoValidConfigMap = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "a",
			Annotations: map[string]string{
			},
			Labels: map[string]string{LabelCluster: testClusterName},
		},
	},
}

var configMapListResultWithSchedulerConfig = []*v1.ConfigMap{
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-managed-cluster",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{"config": string(jsonManagedClusterInfoMatch)},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ase-sub-cluster-scheduler-config-abc",
			Labels: map[string]string{
				LabelCluster: testClusterName,
			},
		},
		Data: map[string]string{
			"config": string(jsonSchedulerConfig),
		},
	},
}

func (FakeConfigMapLister) List(selector labels.Selector) (ret []*v1.ConfigMap, err error) {
	return configMapListResult, nil
}

func (FakeConfigMapLister) ConfigMaps(namespace string) listercorev1.ConfigMapNamespaceLister {
	panic("implement me")
}

func TestValidateInitialization(t *testing.T) {
	handler := &Ase{Handler: &admission.Handler{}}
	err := handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &Ase{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	err = handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &Ase{Handler: &admission.Handler{}}
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
		FakeOptions: test.FakeOptions{UseConfigMapLister: FakeConfigMapLister{}},
	})
	err = handler.ValidateInitialization()
	if err == nil {
		t.Errorf("ValidateInitialization should fail")
	}

	handler = &Ase{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
		FakeOptions: test.FakeOptions{UseConfigMapLister: FakeConfigMapLister{}},
	})
	err = handler.ValidateInitialization()
	if err != nil {
		t.Errorf("ValidateInitialization shouldn't fail")
	}
}

func TestShouldIgnore(t *testing.T) {
	podKind := api.Kind("Pod").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	configMapKind := api.Kind("ConfigMap").WithVersion("version")
	configMapRes := api.Resource("configmaps").WithVersion("version")

	handler := makeAse(t)
	//handler.ShouldIgnore()

	tests := []struct {
		name                string
		subresource         string
		attributes          admission.Attributes
		expected            bool
		configMapListResult []*v1.ConfigMap
	}{
		{
			name:       "nil",
			attributes: nil,
			expected:   true,
		}, {
			name:       "sub resource",
			attributes: admission.NewAttributesRecord(nil, nil, podKind, "test", "a", podRes, "subresource", admission.Create, false, nil),
			expected:   true,
		}, {
			name:       "no body",
			attributes: admission.NewAttributesRecord(nil, nil, configMapKind, "test", AseManagedClusterConfigMapName, configMapRes, "", admission.Create, false, nil),
			expected:   true,
		}, {
			name: "special config map",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AseManagedClusterConfigMapName,
					Namespace: "test",
				}}, nil, configMapKind, "test", AseManagedClusterConfigMapName, configMapRes, "", admission.Create, false, nil),
			expected: true,
		}, {
			name: "no annotation",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected: true,
		}, {
			name: "no sub cluster name in annotation",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "name",
					Namespace:   "test",
					Annotations: map[string]string{"foo": "bar"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected: true,
		}, {
			name: "no cluster id in annotation",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Labels:    map[string]string{LabelSubCluster: "abc"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected: true,
		}, {
			name: "not a managed cluster",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Labels:    map[string]string{LabelCluster: "x", LabelSubCluster: "abc"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected:            true,
			configMapListResult: configMapListResultNoValidConfigMap,
		}, {
			name: "managedCluster config map invalid",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Labels:    map[string]string{LabelCluster: "x", LabelSubCluster: "abc"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected:            true,
			configMapListResult: configMapListResultInvalidConfigMap,
		}, {
			name: "sub cluster name not found",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Labels:    map[string]string{LabelCluster: "x", LabelSubCluster: "abc"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected:            true,
			configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		}, {
			name: "shouldn't ignore",
			attributes: admission.NewAttributesRecord(&api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
				}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
			expected:            false,
			configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		},
	}

	for _, tc := range tests {
		fmt.Println("Running " + tc.name)
		configMapListResult = tc.configMapListResult
		result := handler.ShouldIgnore(tc.attributes, "")
		if result != tc.expected {
			t.Errorf("%s: expected %t got %t", tc.name, tc.expected, result)
			continue
		}
	}
}

func TestAdmit(t *testing.T) {
	podKind := api.Kind("Pod").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	configMapKind := api.Kind("ConfigMap").WithVersion("version")
	configMapRes := api.Resource("configmaps").WithVersion("version")

	handler := makeAse(t)
	//handler.ShouldIgnore()

	testClusterName := "testname"

	tests := []struct {
		name                string
		subresource         string
		attributes          admission.Attributes
		expectError         bool
		configMapListResult []*v1.ConfigMap
		valueChecker        func(admission.Attributes) error
	}{{
		name: "test AdmitFuncAddAseAnnotationsAndLabels",
		attributes: admission.NewAttributesRecord(&api.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, configMapKind, "test", "name", configMapRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		valueChecker: func(attributes admission.Attributes) error {
			resourceLabels := attributes.GetObject().(KubeResource).GetLabels()
			labelValue := resourceLabels[LabelCluster]
			if labelValue != testClusterName {
				return errors.New("wrong label " + labelValue + " vs " + testClusterName)
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer, with no configmap: ase-sub-cluster-pod-log-config-clustername",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		valueChecker: func(attributes admission.Attributes) error {
			// 不会添加
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 0 {
				t.Errorf("should not have containers")
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer, log container already exists",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Name: AseLogAgentContainerName},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		valueChecker: func(attributes admission.Attributes) error {
			// 不会添加
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContext),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithLogConfig,
		valueChecker: func(attributes admission.Attributes) error {
			// 会自动添加一个logContainer
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			if len(pod.Spec.Containers[0].VolumeMounts) != 0 {
				t.Errorf("should have no volume mount")
				return nil
			}

			logContainer := pod.Spec.Containers[0]
			if logContainer.Name != AseLogAgentContainerName {
				t.Errorf("wrong container name")
			}

			if len(logContainer.Env) != 3 || logContainer.Env[1].Value != "ABC/ABC/ABC/ABC" {
				t.Errorf("wrong container env")
			}

			if logContainer.Image != logtailImage {
				t.Errorf("wrong container image")
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer with DefaultVolumeMountConfigs",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContext),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithLogConfigWithVolumeMountConfig,
		valueChecker: func(attributes admission.Attributes) error {
			// 会自动添加一个logContainer
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			if len(pod.Spec.Containers[0].VolumeMounts) != 2 {
				t.Errorf("should have 2 volume mounts")
				return nil
			}

			if pod.Spec.Containers[0].VolumeMounts[0].Name != "name-a" || pod.Spec.Containers[0].VolumeMounts[0].MountPath != "/home/admin/logs/a" {
				t.Errorf("incorrect volume mount")
				return nil
			}

			logContainer := pod.Spec.Containers[0]
			if logContainer.Name != AseLogAgentContainerName {
				t.Errorf("wrong container name")
			}

			if len(logContainer.Env) != 3 || logContainer.Env[1].Value != "ABC/ABC/ABC/ABC" {
				t.Errorf("wrong container env")
			}

			if logContainer.Image != logtailImage {
				t.Errorf("wrong container image")
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer with volumeMountConfigs in annotation",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithVolumeMountConfig),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithLogConfig,
		valueChecker: func(attributes admission.Attributes) error {
			// 会自动添加一个logContainer
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			if len(pod.Spec.Containers[0].VolumeMounts) != 3 {
				t.Errorf("should have 3 volume mounts")
				return nil
			}

			if pod.Spec.Containers[0].VolumeMounts[2].Name != "name-c" || pod.Spec.Containers[0].VolumeMounts[2].MountPath != "/home/admin/logs/___c_" {
				t.Errorf("incorrect volume mount")
				return nil
			}

			logContainer := pod.Spec.Containers[0]
			if logContainer.Name != AseLogAgentContainerName {
				t.Errorf("wrong container name")
			}

			if len(logContainer.Env) != 3 || logContainer.Env[1].Value != "ABC/ABC/ABC/ABC" {
				t.Errorf("wrong container env")
			}

			if logContainer.Image != logtailImage {
				t.Errorf("wrong container image")
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer with user defined id",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithLogConfig,
		valueChecker: func(attributes admission.Attributes) error {
			// 会自动添加一个logContainer
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			logContainer := pod.Spec.Containers[0]
			if logContainer.Name != AseLogAgentContainerName {
				t.Errorf("wrong container name")
			}

			if len(logContainer.Env) != 3 || logContainer.Env[1].Value != logtailUserDefinedId {
				t.Errorf("wrong container env")
			}

			if logContainer.Image != logtailImage {
				t.Errorf("wrong container image")
			}

			if logContainer.Resources.Requests != nil {
				t.Errorf("should not have resource requests")
			}
			return nil
		},
	}, {
		name: "test makeLogAgentContainer with logAgentRequestedResource",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithLogConfigAndLogAgentRequestedResource,
		valueChecker: func(attributes admission.Attributes) error {
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			logContainer := pod.Spec.Containers[0]
			if logContainer.Resources.Requests == nil {
				t.Errorf("should have resource requests")
			}
			return nil
		},
	}, {
		name: "test AdmitFuncPodOverSchedule",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Resources: api.ResourceRequirements{
						Requests: api.ResourceList{
							api.ResourceName(api.ResourceCPU):    resource.MustParse("8000m"),
							api.ResourceName(api.ResourceMemory): resource.MustParse("200G"),
						},
						Limits: api.ResourceList{
							api.ResourceName(api.ResourceCPU):    resource.MustParse("16000m"),
							api.ResourceName(api.ResourceMemory): resource.MustParse("300G"),
						},
					}},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			},
		}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithSchedulerConfig,
		valueChecker: func(attributes admission.Attributes) error {
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			container := pod.Spec.Containers[0]
			if container.Resources.Requests == nil {
				t.Errorf("should have resource requests")
			}

			cpuRequestValue := container.Resources.Requests.Cpu().Value()
			if cpuRequestValue != 4 {
				t.Errorf("wrong cpu request value")
			}

			if container.Resources.Requests.Cpu().Value() != container.Resources.Limits.Cpu().Value() {
				t.Errorf("cpu request and limit should be the same with hard overschedule")
			}
			return nil
		},
	}, {
		name: "test AdmitFuncPodOverSchedule using annotation",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Resources: api.ResourceRequirements{
						Requests: api.ResourceList{
							api.ResourceName(api.ResourceCPU):    resource.MustParse("8000m"),
							api.ResourceName(api.ResourceMemory): resource.MustParse("200G"),
						},
						Limits: api.ResourceList{
							api.ResourceName(api.ResourceCPU):    resource.MustParse("16000m"),
							api.ResourceName(api.ResourceMemory): resource.MustParse("300G"),
						},
					}},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext:      string(logContextWithUserDefinedId),
					AnnotationHardCpuOverSchedule:  "false",
					AnnotationCpuOverScheduleRatio: "4",
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			},
		}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultWithSchedulerConfig,
		valueChecker: func(attributes admission.Attributes) error {
			pod := attributes.GetObject().(*api.Pod)
			if len(pod.Spec.Containers) != 1 {
				t.Errorf("should have 1 container")
				return nil
			}

			container := pod.Spec.Containers[0]
			if container.Resources.Requests == nil {
				t.Errorf("should have resource requests")
			}

			cpuRequestValue := container.Resources.Requests.Cpu().Value()
			if cpuRequestValue != 2 {
				t.Errorf("wrong cpu request value")
			}

			return nil
		},
	},
	}

	for _, tc := range tests {
		fmt.Println("Running " + tc.name)
		configMapListResult = tc.configMapListResult
		err := handler.Admit(tc.attributes)

		if !tc.expectError && (err != nil) {
			t.Errorf("Admit got unexpected error %v", err)
			continue
		}

		if tc.expectError && (err == nil) {
			t.Errorf("expected error but didn't get error")
			continue
		}

		valueCheckerErr := tc.valueChecker(tc.attributes)
		if valueCheckerErr != nil {
			t.Errorf("valueChecker failed %v", valueCheckerErr)
			continue
		}
	}
}

func TestValidate(t *testing.T) {
	podKind := api.Kind("Pod").WithVersion("version")
	nodeKind := api.Kind("Node").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	nodeRes := api.Resource("nodes").WithVersion("version")
	//configMapKind := api.Kind("ConfigMap").WithVersion("version")
	//configMapRes := api.Resource("configmaps").WithVersion("version")

	handler := makeAse(t)

	testClusterName := "testname"

	InjectHttpClient(func() HttpClient {
		return &FakeHttpClient{}
	})

	tests := []struct {
		name                string
		subresource         string
		attributes          admission.Attributes
		expectError         bool
		configMapListResult []*v1.ConfigMap
		httpResponseStatus  string
		valueChecker        func(admission.Attributes) error
	}{{
		name: "test validate ok",
		attributes: admission.NewAttributesRecord(&api.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		configMapListResult: configMapListResultValidConfigMapMatchClusterId,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test image permission",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Image: "image"},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		httpResponseStatus:  "200",
		configMapListResult: configMapListResultWithImageConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test no image permission",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Image: "image"},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         true,
		httpResponseStatus:  "401",
		configMapListResult: configMapListResultWithImageConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test empty image config",
		attributes: admission.NewAttributesRecord(&api.Pod{
			Spec: api.PodSpec{
				Containers: []api.Container{
					{Image: "image"},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
					AnnotationLogAgentContext: string(logContextWithUserDefinedId),
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, podKind, "test", "name", podRes, "", admission.Create, false, nil),
		expectError:         false,
		httpResponseStatus:  "200",
		configMapListResult: configMapListResultWithEmptyImageConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test validate nodes missing required annotations",
		attributes: admission.NewAttributesRecord(&api.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Annotations: map[string]string{
				},
				Labels: map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc"},
			}}, nil, nodeKind, "test", "name", nodeRes, "", admission.Create, false, nil),
		expectError:         true,
		httpResponseStatus:  "401",
		configMapListResult: configMapListResultWithImageConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test validate nodes missing config map",
		attributes: admission.NewAttributesRecord(&api.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc", LabelNodeGroupName: "ng",},
			}}, nil, nodeKind, "test", "name", nodeRes, "", admission.Create, false, nil),
		expectError:         true,
		httpResponseStatus:  "401",
		configMapListResult: configMapListResultWithImageConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	}, {
		name: "test validate nodes successful",
		attributes: admission.NewAttributesRecord(&api.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "test",
				Labels:    map[string]string{LabelCluster: testClusterName, LabelSubCluster: "abc", LabelNodeGroupName: "ng",},
			}}, nil, nodeKind, "test", "name", nodeRes, "", admission.Create, false, nil),
		expectError:         false,
		httpResponseStatus:  "200",
		configMapListResult: configMapListResultWithNodeGroupConfig,
		valueChecker: func(attributes admission.Attributes) error {
			return nil
		},
	},
	}

	for _, tc := range tests {
		fmt.Println("Running " + tc.name)
		configMapListResult = tc.configMapListResult
		fakeHttpResponseStatus = tc.httpResponseStatus
		err := handler.Validate(tc.attributes)

		if !tc.expectError && (err != nil) {
			t.Errorf("Admit got unexpected error %v", err)
			continue
		}

		if tc.expectError && (err == nil) {
			t.Errorf("expected error but didn't get error")
			continue
		}

		valueCheckerErr := tc.valueChecker(tc.attributes)
		if valueCheckerErr != nil {
			t.Errorf("valueChecker failed %v", valueCheckerErr)
			continue
		}
	}
}

func makeAse(t *testing.T) *Ase {
	handler := &Ase{Handler: &admission.Handler{}}
	handler.SetExternalKubeClientSet(&test.FakeClient{})
	handler.SetExternalKubeInformerFactory(&test.FakeSharedInformerFactory{
		FakeOptions: test.FakeOptions{UseConfigMapLister: FakeConfigMapLister{}},
	})
	err := handler.ValidateInitialization()
	if err != nil {
		t.Errorf("ValidateInitialization failed %v", err)
	}
	return handler
}

func TestGenerateSignedUrl(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		requestUrl := req.URL.String()
		if requestUrl != "/privateapi/ase/v1/filestorage/getSignedDownloadUrl?filePath=Test+Path%2Fabc&tenantName=TestTenant" {
			t.Errorf("unexpected request url %s", requestUrl)
			return
		}
		res.Write([]byte(`{"success": true, "data": "http://signed-url"}`))
	}))
	os.Setenv(EnvAseSystemUrl, testServer.URL)
	defer func() {
		os.Setenv(EnvAseSystemUrl, "")
	}()
	defer func() { testServer.Close() }()

	result, err := getSignedUrl("TestTenant", "Test Path/abc")

	fmt.Printf("%s %v\n", result, err)

	if err != nil {
		t.Errorf("%v", err)
		return
	}

	if result != "http://signed-url" {
		t.Errorf("unexpected result %s", result)
		return
	}

}

func TestGenerateSignedUrlAdmitFunc(t *testing.T) {
	ase := makeAse(t)
	podKind := api.Kind("Pod").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationGenerateSignedUrl: "Test Path/abc",
			},
			Labels: map[string]string{
				LabelTenant: "TestTenant",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: "Test",
				},
			},
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		requestUrl := req.URL.String()
		if requestUrl != "/privateapi/ase/v1/filestorage/getSignedDownloadUrl?filePath=Test+Path%2Fabc&tenantName=TestTenant" {
			t.Errorf("unexpected request url %s", requestUrl)
			return
		}
		res.Write([]byte(`{"success": true, "data": "http://signed-url"}`))
	}))

	os.Setenv(EnvAseSystemUrl, testServer.URL+"/")
	defer func() {
		os.Setenv(EnvAseSystemUrl, "")
		testServer.Close()
	}()

	record := admission.NewAttributesRecord(pod, nil, podKind, "test", "a", podRes, "subresource", admission.Create, false, nil)
	err := generateSignedUrl(ase, record, )

	if err != nil {
		t.Errorf("%v", err)
		return
	}

	envVar := pod.Spec.Containers[0].Env[0]
	if envVar.Name != EnvVarSignedUrl || envVar.Value != "http://signed-url" {
		t.Errorf("invalid envvar %v", envVar)
	}
}

func TestGetAseUrl(t *testing.T) {
	defer func() {
		_ = os.Setenv(EnvAseSystemUrl, "")
	}()
	if getAseUrl() != "http://10.252.1.103:8341" {
		t.Error("get ase url error")
		return
	}
	_ = os.Setenv(EnvAseSystemUrl, "ABC")
	if getAseUrl() != "ABC" {
		t.Error("get ase url error")
		return
	}
}

func TestIdempotency(t *testing.T) {
	ase := makeAse(t)
	configMapListResult = configMapListResultValidConfigMapMatchClusterId

	podKind := api.Kind("Pod").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				LabelTenant:     "TestTenant",
				LabelCluster:    testClusterName,
				LabelSubCluster: "abc",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: "Test",
				},
			},
		},
	}
	attributes := admission.NewAttributesRecord(pod, nil, podKind, "test", "a", podRes, "", admission.Create, false, nil)
	err := ase.Admit(attributes)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	annotations := attributes.GetObject().(KubeResource).GetAnnotations()
	if annotations == nil {
		t.Errorf("annotations should not be nil")
	}

	alreadyProcessed := alreadyProcessedByAse(attributes)
	if !alreadyProcessed {
		t.Errorf("should be already processed")
	}

}

func TestFasInteropInjectLogPath(t *testing.T) {
	ase := makeAse(t)
	podKind := api.Kind("Pod").WithVersion("version")
	podRes := api.Resource("pods").WithVersion("version")
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				LabelTenant:     "TestTenant",
				LabelCluster:    testClusterName,
				LabelSubCluster: "abc",
				LabelAppVersion: "test-v",
			},
			Annotations: map[string]string{
				AnnotationFasInteropInjectLogPath: "true",
			},
			GenerateName: "test-pod-",
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: "Test",
				},
			},
			Volumes: []core.Volume{
				{Name: "AA1", VolumeSource: core.VolumeSource{HostPath: &core.HostPathVolumeSource{Path: "aaa", Type: nil}}},
				{Name: "AA2", VolumeSource: core.VolumeSource{HostPath: &core.HostPathVolumeSource{Path: InteropFasLogsRootPath + "abc", Type: nil}}},
				{Name: "AA3", VolumeSource: core.VolumeSource{HostPath: &core.HostPathVolumeSource{Path: InteropFasLogsRootPath + "abcd/", Type: nil}}},
				{Name: "test-on-fasagent", VolumeSource: core.VolumeSource{HostPath: &core.HostPathVolumeSource{Path: InteropFasLogsRootPath + "abcde/", Type: nil}}},
				{Name: "AA4", VolumeSource: core.VolumeSource{PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{ClaimName: "testclaim2"}}},
			},
		},
	}
	attributes := admission.NewAttributesRecord(pod, nil, podKind, "test", "a", podRes, "", admission.Create, false, nil)

	_ = injectFasLogPath(ase, attributes)

	if !strings.HasPrefix(pod.Spec.Containers[0].Env[0].Value, "test-pod-") || len(pod.Spec.Containers[0].Env[0].Value) != 14 {
		t.Fatalf("wrong container env")
	}

	if !strings.HasPrefix(pod.Spec.Volumes[1].HostPath.Path, "/home/admin/fas-logs/abc/test-pod-") ||
		!strings.HasPrefix(pod.Spec.Volumes[2].HostPath.Path, "/home/admin/fas-logs/abcd/test-pod-") ||
		!strings.HasPrefix(pod.Spec.Volumes[3].HostPath.Path, "/home/admin/fas-logs/abcde/test-") {
		t.Fatalf("wrong host path")
	}

}