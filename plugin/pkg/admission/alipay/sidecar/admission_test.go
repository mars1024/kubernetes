package sidecar

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

const (
	defaultMOSNConfigMapTemplate = `
containers:
- name: mosn-sidecar-container
  image: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/image" "reg.docker.alibaba-inc.com/antmesh/mosn:1.0.2-5995f65" }}
  imagePullPolicy: IfNotPresent
  ports:
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/ingress-port" 12200 }}
    protocol: TCP
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/egress-port" 12220 }}
    protocol: TCP
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/registry-port" 13330 }}
    protocol: TCP
  lifecycle:
    postStart:
      exec:
        command:
        - bash
        - {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/post-start-command" "/home/admin/mosn/bin/process_checker.sh" }}
  terminationMessagePolicy: File
  resources:
    requests:
      {{ if isCPUSet .ObjectMeta }}
      cpu: {{ CPUSetToInt64 .PodSpec "1000m" }}
      {{ else }}
      cpu: 0
      {{ end }}
      memory: {{ convertMemoryBasedOnCPUCount .PodSpec "128Mi" }}
      ephemeral-storage: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/ephemeral-storage" "20G" }}
    limits:
      {{ if isCPUSet .ObjectMeta }}
      cpu: {{ CPUSetToInt64 .PodSpec "1000m" }}
      {{ else }}
      cpu: {{ CPUShareToInt64 .PodSpec "1000m" }}
      {{ end }}
      memory: {{ convertMemoryBasedOnCPUCount .PodSpec "128Mi" }}
      ephemeral-storage: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/ephemeral-storage" "20G" }}
  env:
  - name: ALIPAY_APP_ZONE
    value: {{ valueOfMap .ObjectMeta.Labels "meta.k8s.alipay.com/zone" "" | ToUpper }}
  - name: DBMODE
    value: prod
  - name: CONFREGURL
    value: confreg-pool.{{ valueOfMap .ObjectMeta.Labels "meta.k8s.alipay.com/zone" "" | ToLower }}.alipay.com
  - name: DOMAINNAME
    value: {{ valueOfMap .ObjectMeta.Labels "meta.k8s.alipay.com/zone" "" | ToLower }}.alipay.com
  volumeMounts:
  - name: mosn-conf
    mountPath: /home/admin/mosn/conf
volumes:
- flexVolume:
    driver: alipay/pouch-volume
    options:
      image: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/image" "reg.docker.alibaba-inc.com/antmesh/mosn:1.1.0-b9ea686" }}
      imagePath: /home/admin/mosn/conf/.
  name: mosn-conf
appEnvs:
- name: MOSN_ENABLE
  value: "true"
- name: MOSN_EGRESS_PORT
  value: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/egress-port" 12220 }}
- name: MOSN_REGISTRY_PORT
  value: {{ valueOfMap .ObjectMeta.Annotations "mosn.sidecar.k8s.alipay.com/registry-port" 13330 }}
- name: RPC_TR_PORT
  value: 12199
`

	defaultDBMeshConfigMapTemplate = `
containers:
- name: dbmesh-sidecar-container
  image: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/image" "reg.docker.alibaba-inc.com/antmesh/dbmesh:1.0.2-5995f65" }}
  imagePullPolicy: IfNotPresent
  ports:
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/ingress-port" 12200 }}
    protocol: TCP
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/egress-port" 12220 }}
    protocol: TCP
  - containerPort: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/registry-port" 13330 }}
    protocol: TCP
  lifecycle:
    postStart:
      exec:
        command:
        - bash
        - {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/post-start-command" "/home/admin/dbmesh/bin/process_checker.sh" }}
  terminationMessagePolicy: File
  resources:
    requests:
      {{ if isCPUSet .ObjectMeta }}
      cpu: {{ CPUSetToInt64 .PodSpec "1000m" }}
      {{ else }}
      cpu: 0
      {{ end }}
      memory: {{ convertMemoryBasedOnCPUCount .PodSpec "128Mi" }}
      ephemeral-storage: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/ephemeral-storage" "20G" }}
    limits:
      {{ if isCPUSet .ObjectMeta }}
      cpu: {{ CPUSetToInt64 .PodSpec "1000m" }}
      {{ else }}
      cpu: {{ CPUShareToInt64 .PodSpec "1000m" }}
      {{ end }}
      memory: {{ convertMemoryBasedOnCPUCount .PodSpec "128Mi" }}
      ephemeral-storage: {{ valueOfMap .ObjectMeta.Annotations "dbmesh.sidecar.k8s.alipay.com/ephemeral-storage" "20G" }}
  volumeMounts:
  - name: dbmesh-conf
    mountPath: /home/admin/dbmesh/conf
`
)

func addDefaultConfigMap(sidecar *alipaySidecar) {
	informerFactory := informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	sidecar.SetInternalKubeInformerFactory(informerFactory)
	// First add the existing classes to the cache.
	informerFactory.Core().InternalVersion().ConfigMaps().Informer().GetStore().Add(&api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-template",
			Namespace: "mosn-system",
		},
		Data: map[string]string{
			sidecarTemplateKey: defaultMOSNConfigMapTemplate,
		},
	})
	informerFactory.Core().InternalVersion().ConfigMaps().Informer().GetStore().Add(&api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-template",
			Namespace: "dbmesh-system",
		},
		Data: map[string]string{
			sidecarTemplateKey: defaultDBMeshConfigMapTemplate,
		},
	})
	informerFactory.Core().InternalVersion().ConfigMaps().Informer().GetStore().Add(&api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sidecars",
			Namespace: metav1.NamespaceSystem,
		},
		Data: map[string]string{
			supportedSidecarKey: "[\"mosn\", \"dbmesh\"]",
		},
	})
}

func TestAdmit(t *testing.T) {
	podToBeInjected := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-mosn-sidecar",
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				alipaysigmak8sapi.LabelZone: "A001",
			},
			Annotations: map[string]string{
				alipaysigmak8sapi.MOSNSidecarInject: string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled),
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name: "app-c1",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{
							api.ResourceCPU:    *resource.NewQuantity(8, resource.DecimalSI),
							api.ResourceMemory: *resource.NewQuantity(8*1024, resource.DecimalSI),
						},
						Limits: api.ResourceList{
							api.ResourceCPU:    *resource.NewQuantity(8, resource.DecimalSI),
							api.ResourceMemory: *resource.NewQuantity(8*1024, resource.DecimalSI),
						},
					},
					VolumeMounts: []api.VolumeMount{
						{
							Name:      "logs-volume",
							MountPath: defaultLogsDir,
						},
					},
				},
				{
					Name: "app-c2",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{
							api.ResourceCPU:    *resource.NewQuantity(4, resource.DecimalSI),
							api.ResourceMemory: *resource.NewQuantity(4*1024, resource.DecimalSI),
						},
						Limits: api.ResourceList{
							api.ResourceCPU:    *resource.NewQuantity(4, resource.DecimalSI),
							api.ResourceMemory: *resource.NewQuantity(4*1024, resource.DecimalSI),
						},
					},
				},
			},
		},
	}

	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "app-c1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{
							SpreadStrategy: sigmak8sapi.SpreadStrategySameCoreFirst,
						},
					},
				},
			},
			{
				Name: "app-c2",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{
							SpreadStrategy: sigmak8sapi.SpreadStrategySameCoreFirst,
						},
					},
				},
			},
		},
	}

	allocSpecWithMOSN := allocSpec
	allocSpecWithMOSN.Containers = append(allocSpecWithMOSN.Containers,
		sigmak8sapi.Container{
			Name: "mosn-sidecar-container",
		})

	podToBeInjectedWithCPUSet := podToBeInjected.DeepCopy()
	allocSpecBytes, _ := json.Marshal(allocSpec)
	podToBeInjectedWithCPUSet.Annotations[sigmak8sapi.AnnotationPodAllocSpec] =
		string(allocSpecBytes)

	podToBeInjectedWithCPUShare := podToBeInjected.DeepCopy()
	allocSpecWithMOSNBytes, _ := json.Marshal(allocSpecWithMOSN)
	podToBeInjectedWithCPUSet.Annotations[sigmak8sapi.AnnotationPodAllocSpec] =
		string(allocSpecWithMOSNBytes)

	podWithoutInjection := podToBeInjected.DeepCopy()
	podWithoutInjection.Annotations[alipaysigmak8sapi.MOSNSidecarInject] =
		string(alipaysigmak8sapi.SidecarInjectionPolicyDisabled)

	podWithWrongInjection := podToBeInjected.DeepCopy()
	podWithWrongInjection.Annotations[alipaysigmak8sapi.MOSNSidecarInject] = "wrong-value"

	testCases := []struct {
		name              string
		podToBeInjected   *api.Pod
		podAfterInjection *api.Pod
		operation         admission.Operation
		expectedError     bool
	}{
		{
			name:            "cpuset admit success",
			podToBeInjected: podToBeInjectedWithCPUSet,
			operation:       admission.Create,
			expectedError:   false,
		},
		{
			name:            "cpushare admit success",
			podToBeInjected: podToBeInjectedWithCPUShare,
			operation:       admission.Create,
			expectedError:   false,
		},
		{
			name:            "ignore admit",
			podToBeInjected: podWithoutInjection,
			operation:       admission.Create,
			expectedError:   false,
		},
		{
			name:            "operation not support",
			podToBeInjected: podWithWrongInjection,
			operation:       admission.Delete,
			expectedError:   false,
		},
	}

	for i, testCase := range testCases {
		glog.V(4).Infof("starting test case %q", testCase.name)
		sidecar := newAlipaySidecarPlugin()

		// Add default configmap.
		addDefaultConfigMap(sidecar)

		attrs := admission.NewAttributesRecord(
			testCase.podToBeInjected,
			nil,
			api.Kind("Pod").WithVersion("version"),
			testCase.podToBeInjected.ObjectMeta.Namespace,
			"",
			api.Resource("pods").WithVersion("version"),
			"",
			testCase.operation,
			false,
			nil,
		)

		err := sidecar.Admit(attrs)
		if testCase.expectedError {
			if err == nil {
				t.Errorf("Case[%d] with name: %s should return error", i, testCase.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("Case[%d] with name: %s return unexpected error: %v", i, testCase.name, err)
			continue
		}
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		pod           *api.Pod
		operation     admission.Operation
		expectedError bool
	}{
		{
			name: "create operation validate success",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-with-mosn-sidecar",
					Namespace: metav1.NamespaceSystem,
					Annotations: map[string]string{
						alipaysigmak8sapi.MOSNSidecarInject: string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled),
					},
				},
			},
			operation:     admission.Create,
			expectedError: false,
		},
		{
			name: "update operation validate success",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-with-mosn-sidecar",
					Namespace: metav1.NamespaceSystem,
					Annotations: map[string]string{
						alipaysigmak8sapi.MOSNSidecarInject: string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled),
					},
				},
			},
			operation:     admission.Update,
			expectedError: false,
		},
		{
			name: "create operation validate failed",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-with-mosn-sidecar",
					Namespace: metav1.NamespaceSystem,
					Annotations: map[string]string{
						alipaysigmak8sapi.MOSNSidecarInject: "wrong-value",
					},
				},
			},
			operation:     admission.Create,
			expectedError: true,
		},
		{
			name: "operation not support",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-with-mosn-sidecar",
					Namespace: metav1.NamespaceSystem,
					Annotations: map[string]string{
						alipaysigmak8sapi.MOSNSidecarInject: "wrong-value",
					},
				},
			},
			operation:     admission.Delete,
			expectedError: false,
		},
	}

	for i, testCase := range testCases {
		glog.V(4).Infof("starting test case %q", testCase.name)
		sidecar := newAlipaySidecarPlugin()

		// Add default configmap.
		addDefaultConfigMap(sidecar)

		attrs := admission.NewAttributesRecord(
			testCase.pod,
			nil,
			api.Kind("Pod").WithVersion("version"),
			testCase.pod.ObjectMeta.Namespace,
			"",
			api.Resource("pods").WithVersion("version"),
			"",
			testCase.operation,
			false,
			nil,
		)

		err := sidecar.Validate(attrs)
		if testCase.expectedError {
			if err == nil {
				t.Errorf("Case[%d] with name: %s should return error", i, testCase.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("Case[%d] with name: %s return unexpected error: %v", i, testCase.name, err)
			continue
		}
	}
}
