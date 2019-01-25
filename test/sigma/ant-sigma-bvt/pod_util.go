package ant_sigma_bvt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	k8sApi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

const (
	UpgradeSuccessStr = "upgrade container success"
	StopSuccessStr    = "kill container success"
)

//LoadAlipayBasePod() Load base pod for sigma3.1, init some parameters.
func LoadAlipayBasePod(name string, expectStatus k8sApi.ContainerState, enableOverQuota string) (*v1.Pod, error) {
	podFile := filepath.Join(util.TestDataDir, "alipay-sigma3-base-pod.json")
	pod, err := util.LoadPodFromFile(podFile)
	if err != nil {
		return pod, err
	}
	pod.Name = name
	pod.Spec.Containers[0].Name = name
	framework.Logf("Load Base pod: %v", pod.Name)
	allocSpec := &k8sApi.AllocSpec{
		Containers: []k8sApi.Container{
			{
				Name: name,
				Resource: k8sApi.ResourceRequirements{
					CPU: k8sApi.CPUSpec{
						CPUSet: &k8sApi.CPUSetSpec{},
					},
				},
			},
		},
	}
	if enableOverQuota == "true" {
		nodeTerms := []v1.NodeSelectorRequirement{
			{
				Key:      k8sApi.LabelEnableOverQuota,
				Operator: "In",
				Values:   []string{"true"},
			},
		}
		if pod.Spec.Affinity == nil {
			pod.Spec.Affinity = &v1.Affinity{}
		}
		if pod.Spec.Affinity.NodeAffinity == nil {
			pod.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{}
		}
		if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{},
					},
				},
			}
		}
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions = append(
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, nodeTerms...)

		toleration := []v1.Toleration{
			{
				Key:      k8sApi.LabelEnableOverQuota,
				Operator: v1.TolerationOpEqual,
				Value:    "true",
				Effect:   v1.TaintEffectNoSchedule,
			},
		}
		pod.Spec.Tolerations = toleration
	} else {
		allocSpec.Containers[0].Resource.CPU.CPUSet.SpreadStrategy = "sameCoreFirst"
	}
	allocByte, _ := json.Marshal(allocSpec)
	pod.Annotations[k8sApi.AnnotationPodAllocSpec] = string(allocByte)

	containerState := k8sApi.ContainerStateSpec{
		States: map[k8sApi.ContainerInfo]k8sApi.ContainerState{
			k8sApi.ContainerInfo{name}: expectStatus,
		},
	}
	stateSpecStr, _ := json.Marshal(containerState)
	pod.Annotations[k8sApi.AnnotationContainerStateSpec] = string(stateSpecStr)
	pod.Spec.Hostname = pod.Name
	pod.Spec.DNSPolicy = v1.DNSDefault
	return pod, nil
}

//CreateSigmaPod() create sigma pod, return nil if pod status equals expect-status. if timeout, return error.
func CreateSigmaPod(client clientset.Interface, pod *v1.Pod) error {
	var err error
	defer func() {
		if err != nil && pod != nil {
			pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			framework.Logf("Wait pod ready failed, Pod info:%#v, err:%#v", DumpJson(pod), err)
		}
	}()
	_, err = client.CoreV1().Pods(pod.Namespace).Create(pod)
	if err != nil {
		return err
	}
	err = wait.PollImmediate(5*time.Second, 5*time.Minute, CheckPodIsReady(client, pod))
	return err
}

//CheckPodIsReady() check sigma pod status, return true if pod status equals expect-status.
func CheckPodIsReady(client clientset.Interface, pod *v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if pod.Status.PodIP == "" {
			return false, nil
		}
		updateStatusStr, exist := pod.Annotations[k8sApi.AnnotationPodUpdateStatus]
		if !exist {
			return false, nil
		}

		updateStatus := &k8sApi.ContainerStateStatus{}
		err = json.Unmarshal([]byte(updateStatusStr), updateStatus)
		if err != nil {
			return false, err
		}
		// expectState & currentState must be specified, if not skip.
		expectStateStr, exist := pod.Annotations[k8sApi.AnnotationContainerStateSpec]
		if !exist {
			return false, nil
		}

		//unmarshal struct.
		expectState := &k8sApi.ContainerStateSpec{}
		err = json.Unmarshal([]byte(expectStateStr), expectState)
		if err != nil {
			return false, err
		}
		for key, expect := range expectState.States {
			if value, ok := updateStatus.Statuses[key]; ok {
				if expect == value.CurrentState {
					return true, nil
				}
			}
		}
		return false, nil
	}
}

//StopOrStartSigmaPod() start or stop contaienr use annotation/container-state-spec.
func StopOrStartSigmaPod(client clientset.Interface, pod *v1.Pod, expectStatus k8sApi.ContainerState) error {
	pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	containerState := k8sApi.ContainerStateSpec{
		States: map[k8sApi.ContainerInfo]k8sApi.ContainerState{
			k8sApi.ContainerInfo{pod.Spec.Containers[0].Name}: expectStatus,
		},
	}
	stateSpecStr, _ := json.Marshal(containerState)
	pod.Annotations[k8sApi.AnnotationContainerStateSpec] = string(stateSpecStr)
	_, err = client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		return err
	}
	return wait.PollImmediate(5*time.Second, 5*time.Minute, CheckPodIsReady(client, pod))
}

func NewUpgradePod(env []v1.EnvVar) *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Env: env,
				},
			},
		},
	}
}

func NewUpdatePod(resource v1.ResourceRequirements) *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Resources: resource,
				},
			},
		},
	}
}

var upgradeEnv = []v1.EnvVar{
	{
		Name:  "SIGMA3_UPGRADE_TEST",
		Value: "test",
	},
}

var upgradeEnv2 = []v1.EnvVar{
	{
		Name:  "SIGMA3_UPGRADE_TEST2",
		Value: "test2",
	},
}

var updateResource1 = v1.ResourceRequirements{
	Requests: v1.ResourceList{
		v1.ResourceCPU:              *resource.NewQuantity(2, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(2*1024*1024*1024, resource.DecimalSI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(2*1024*1024*1024, resource.DecimalSI),
	},
	Limits: v1.ResourceList{
		v1.ResourceCPU:              *resource.NewQuantity(2, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(2*1024*1024*1024, resource.DecimalSI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(2*1024*1024*1024, resource.DecimalSI),
	},
}

var updateResource2 = v1.ResourceRequirements{
	Requests: v1.ResourceList{
		v1.ResourceCPU:              *resource.NewQuantity(1, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(1*1024*1024*1024, resource.DecimalSI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(1*1024*1024*1024, resource.DecimalSI),
	},
	Limits: v1.ResourceList{
		v1.ResourceCPU:              *resource.NewQuantity(1, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(1*1024*1024*1024, resource.DecimalSI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(1*1024*1024*1024, resource.DecimalSI),
	},
}

//UpgradeSigmaPod() upgrade container env, wait upgraded and return true.
func UpgradeSigmaPod(client clientset.Interface, pod *v1.Pod, upgradePod *v1.Pod, expectStatus k8sApi.ContainerState) error {
	var err error
	defer func() {
		if err != nil {
			framework.Logf("Upgrade pod failed, pod:%#v, err:%#v", DumpJson(pod), err)
		}
	}()
	pod, err = client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	containerState := k8sApi.ContainerStateSpec{
		States: map[k8sApi.ContainerInfo]k8sApi.ContainerState{
			k8sApi.ContainerInfo{pod.Spec.Containers[0].Name}: expectStatus,
		},
	}
	stateSpecStr, _ := json.Marshal(containerState)
	pod.Annotations[k8sApi.AnnotationContainerStateSpec] = string(stateSpecStr)
	pod.Spec.Containers[0].Env = upgradePod.Spec.Containers[0].Env
	_, err = client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	if expectStatus == k8sApi.ContainerStateExited {
		err = util.WaitTimeoutForContainerUpdateStatus(client, pod, pod.Spec.Containers[0].Name, 3*time.Minute, StopSuccessStr, true)
	} else {
		err = util.WaitTimeoutForContainerUpdateStatus(client, pod, pod.Spec.Containers[0].Name, 3*time.Minute, UpgradeSuccessStr, true)
	}
	return err
}

// UpdateSigmaPod() update container's resource, wait update and return true.
func UpdateSigmaPod(client clientset.Interface, pod *v1.Pod, updatePod *v1.Pod, expectStatus k8sApi.ContainerState) error {
	var err error
	defer func() {
		if err != nil {
			framework.Logf("Update pod failed, pod: %#v, err: %#v", DumpJson(pod), err)
		}
	}()
	pod, err = client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	containerState := k8sApi.ContainerStateSpec{
		States: map[k8sApi.ContainerInfo]k8sApi.ContainerState{
			k8sApi.ContainerInfo{pod.Spec.Containers[0].Name}: expectStatus,
		},
	}
	stateSpecStr, _ := json.Marshal(containerState)
	pod.Annotations[k8sApi.AnnotationContainerStateSpec] = string(stateSpecStr)
	pod.Annotations[k8sApi.AnnotationPodInplaceUpdateState] = k8sApi.InplaceUpdateStateCreated
	pod.Spec.Containers[0].Resources = updatePod.Spec.Containers[0].Resources
	_, err = client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	err = wait.PollImmediate(5*time.Second, 5*time.Minute, CheckUpdatePodReady(client, pod))
	return err
}

// CheckUpdatePodReady() check sigma pod update result, return true if pod status equals expect-status.
func CheckUpdatePodReady(client clientset.Interface, pod *v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if pod.Status.PodIP == "" {
			return false, nil
		}

		updateStatusStr, exist := pod.Annotations[k8sApi.AnnotationPodUpdateStatus]
		if !exist {
			return false, nil
		}

		inplaceUpdateState, exist := pod.Annotations[k8sApi.AnnotationPodInplaceUpdateState]
		if !exist {
			return false, nil
		}

		updateStatus := &k8sApi.ContainerStateStatus{}
		err = json.Unmarshal([]byte(updateStatusStr), updateStatus)
		if err != nil {
			return false, err
		}
		// expectState & currentState must be specified, if not skip.
		expectStateStr, exist := pod.Annotations[k8sApi.AnnotationContainerStateSpec]
		if !exist {
			return false, nil
		}

		// unmarshal struct.
		expectState := &k8sApi.ContainerStateSpec{}
		err = json.Unmarshal([]byte(expectStateStr), expectState)
		if err != nil {
			return false, err
		}
		for key, expect := range expectState.States {
			if value, ok := updateStatus.Statuses[key]; ok {
				if expect == value.CurrentState &&
					inplaceUpdateState == k8sApi.InplaceUpdateStateSucceeded {
					return true, nil
				}
			}
		}
		return false, nil
	}
}

//CheckPodNameSpace() check namespace, if exist, skip else create a new one.
func CheckPodNameSpace(kubeClient clientset.Interface, podNameSpace string) error {
	appNameSpace, err := kubeClient.CoreV1().Namespaces().Get(podNameSpace, metav1.GetOptions{})
	if err != nil {
		if apierrs.IsNotFound(err) {
			//Create NameSpace
			nameSpace := &v1.Namespace{
				TypeMeta: metav1.TypeMeta{Kind: "Objects", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name: podNameSpace,
					Labels: map[string]string{
						"creator": podNameSpace,
					},
				},
			}
			nameSpace, err := kubeClient.CoreV1().Namespaces().Create(nameSpace)
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	if appNameSpace != nil && appNameSpace.Name == podNameSpace {
		framework.Logf("NameSpace %s has been created.", podNameSpace)
		return nil
	}
	return fmt.Errorf("check NameSpace failed")
}

//GetNodeSite() get node site info.
func GetNodeSite(kubeClient clientset.Interface) (string, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("No nodes exist.")
	}
	site := ""
	for _, node := range nodes.Items {
		if node.Labels == nil {
			continue
		}
		if node.Labels[k8sApi.LabelSite] != "" {
			site = node.Labels[k8sApi.LabelSite]
		}
	}
	if site == "" {
		return "ant-sigma-test-site", nil
		//return "", fmt.Errorf("No label site in node.")
	}
	return site, nil
}

func IsEnableOverQuota() string {
	enableOverQuota, ok := os.LookupEnv("ENABLEOVERQUOTA")
	if !ok {
		enableOverQuota = "false"
	}
	return enableOverQuota
}
