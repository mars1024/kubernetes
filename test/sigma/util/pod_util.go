package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
)

const (
	// PodContainerStartTimeout pod container start time out
	PodContainerStartTimeout = 5 * time.Minute

	// Poll How often to Poll pods, nodes and claims.
	Poll = 2 * time.Second
)

// LoadPodFromFile create a pod object from file
func LoadPodFromFile(file string) (*v1.Pod, error) {
	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var pod *v1.Pod
	err = json.Unmarshal(fileContent, &pod)
	if err != nil {
		return nil, err
	}
	if env.GetTester() == env.TesterJituan {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, v1.Toleration{
			Key:      sigmak8sapi.LabelResourcePool,
			Operator: v1.TolerationOpExists,
			Effect:   v1.TaintEffectNoSchedule,
		})
	}
	if env.GetTester() == env.TesterAnt {
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		for _, kv := range [][2]string{
			{sigmak8sapi.LabelAppName, "ant-sigma-test-app"},
			{sigmak8sapi.LabelInstanceGroup, "ant-sigma-test-instance-group"},
			{sigmak8sapi.LabelDeployUnit, "ant-sigma-test-deploy-unit"},
			{sigmak8sapi.LabelSite, "ant-sigma-test-site"},
			{alipaysigmak8sapi.LabelZone, "ant-sigma-test-zone"},
		} {
			if _, exists := pod.Labels[kv[0]]; !exists {
				pod.Labels[kv[0]] = kv[1]
			}
		}
	}
	return pod, nil
}

// CreatePod create pod by using k8s api.
func CreatePod(client clientset.Interface, pod *v1.Pod, namespace string) (*v1.Pod, error) {
	return client.CoreV1().Pods(namespace).Create(pod)
}

// DeletePod delete pod by using k8s api, and check whether pod is really deleted within the timeout.
func DeletePod(client clientset.Interface, pod *v1.Pod) error {
	err := client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, metav1.NewDeleteOptions(5))
	if err != nil {
		return err
	}
	timeout := 5 * time.Minute
	t := time.Now()
	for {
		_, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), "not found") {
			framework.Logf("pod %s has been removed", pod.Name)
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("Gave up waiting for pod %s is removed after %v seconds",
				pod.Name, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether pod %s is removed", pod.Name)
		time.Sleep(5 * time.Second)
	}
}

// DeleteAllPodsInNamespace delete all pods in a namespace
func DeleteAllPodsInNamespace(client clientset.Interface, ns string) error {
	podList, err := client.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		err := client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		framework.Logf("delete pod[%s] in namespace %s", pod.Name, pod.Namespace)
		if err != nil {
			return err
		}
	}
	timeout := 5 * time.Minute
	t := time.Now()
	for {
		podList, err := client.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			framework.Logf("all pods in namespace[%s] are removed", ns)
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("gave up waiting for all pod in namespace %s are removed after %v seconds",
				ns, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether all pod in namespace %s are removed", ns)
		time.Sleep(5 * time.Second)
	}
}

func WaitPodNotExists(client clientset.Interface, pod *v1.Pod) error {
	return wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		_, err = client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), "not found") {
			framework.Logf("pod %s has been removed", pod.Name)
			return true, nil
		}
		framework.Logf("Retrying to check whether pod %s is removed", pod.Name)
		return false, err
	})
}

// WaitTimeoutForPodStatus check whether the pod status is same as expected status within the timeout.
func WaitTimeoutForPodStatus(client clientset.Interface, pod *v1.Pod, expectedStatus v1.PodPhase, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, CheckPodStatus(client, pod.Name, pod.Namespace, expectedStatus))
}

// WaitTimeoutForPodContainerStatus check whether the container in pod status ready within the timeout.
func WaitTimeoutForPodContainerStatusReady(client clientset.Interface, pod *v1.Pod, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, checkPodContainerStatusReady(client, pod.Name, pod.Namespace))
}

// checkPodStatus check whether pod status is same as expected status.
func CheckPodStatus(client clientset.Interface, podName, namespace string, expectedStatus v1.PodPhase) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		framework.Logf("pod[%s] status phase is %v", podName, pod.Status.Phase)
		if pod.Status.Phase == expectedStatus {
			return true, nil
		}
		return false, nil
	}
}

// WaitTimeoutForPodStatusByContainerName check whether the pod status is same as expected status within the timeout.
func WaitTimeoutForPodStatusByContainerName(client clientset.Interface, pod *v1.Pod, expectedStatus v1.PodPhase, containerName string, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, CheckPodStatusByContainerName(client, containerName, pod.Namespace, expectedStatus))
}

// CheckPodStatusByContainerName check whether pod status is same as expected status.
func CheckPodStatusByContainerName(client clientset.Interface, containerName, namespace string, expectedStatus v1.PodPhase) wait.ConditionFunc {
	containerLabel := map[string]string{
		alipaysigmak8sapi.LabelPodContainerName: containerName,
	}
	return func() (bool, error) {
		pod, err := GetPodsByLabel(client, namespace, containerLabel, false)
		if err != nil {
			return false, err
		}
		framework.Logf("pod[%s] status phase is %v", containerName, pod.Status.Phase)
		if pod.Status.Phase == expectedStatus {
			return true, nil
		}
		return false, nil
	}
}

func GetPodsByLabel(client clientset.Interface, namespace string, containerLabel map[string]string, initializer bool) (*v1.Pod, error) {
	labelSelector := labels.Set(containerLabel).AsSelectorPreValidated()
	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector:        labelSelector.String(),
		IncludeUninitialized: true,
	})
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("Unexpected pod num: %v", len(pods.Items))
	}
	return &pods.Items[0], err
}

// checkPodContainerStatus check whether container in pod status is ready.
func checkPodContainerStatusReady(client clientset.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		framework.Logf("pod[%s] container ready status phase is %v", podName, pod.Status.ContainerStatuses[0].Ready)
		if pod.Status.ContainerStatuses[0].Ready == true {
			return true, nil
		}
		return false, nil
	}
}

// WaitTimeoutForContainerUpdateStatus check pod's updateStatus to wait the action such as start, stop, update, upgrade is finished.
func WaitTimeoutForContainerUpdateStatus(client clientset.Interface, pod *v1.Pod, containerName string,
	timeout time.Duration, keyWord string, expectedSuccess bool) error {
	options := metav1.SingleObject(metav1.ObjectMeta{Name: pod.Name})
	w, err := client.CoreV1().Pods(pod.Namespace).Watch(options)
	if err != nil {
		return err
	}
	_, err = watch.Until(timeout, w, func(event watch.Event) (bool, error) {
		switch pod := event.Object.(type) {
		case *v1.Pod:
			updateStatusStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
			if !exists {
				framework.Logf("[WaitTimeoutForContainerUpdateStatus] update status doesn't exist")
				return false, nil
			}
			framework.Logf("[WaitTimeoutForContainerUpdateStatus] updateStatusStr: %v", updateStatusStr)
			containerStatus := sigmak8sapi.ContainerStateStatus{}
			if err := json.Unmarshal([]byte(updateStatusStr), &containerStatus); err != nil {
				framework.Logf("[WaitTimeoutForContainerUpdateStatus] unmarshal failed")
				return false, nil
			}
			for containerInfo, containerStatus := range containerStatus.Statuses {
				if containerInfo.Name == containerName {
					if containerStatus.Success == expectedSuccess &&
						strings.Contains(containerStatus.Message, keyWord) {
						framework.Logf("[WaitTimeoutForContainerUpdateStatus] container's updateStatus is matched expected status")
						return true, nil
					}
				}
			}
			return false, nil

		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("[WaitTimeoutForContainerUpdateStatus] timeout")
	}

	return nil
}

// GetContainerIDFromPod get the first container ID in the specified pod
func GetContainerIDFromPod(pod *v1.Pod) string {
	containerID := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "//")[1]
	rs := []rune(containerID)
	return string(rs[0:6])
}

// GetInplaceSetNameFromPod get the inplaceset name of pod if exists
func GetInplaceSetNameFromPod(pod *v1.Pod) string {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "InPlaceSet" {
			return ownerRef.Name
		}
	}
	return ""
}

func GenerateStatePatchData(containerStateSpec sigmak8sapi.ContainerStateSpec) (string, error) {
	stateSpec, err := json.Marshal(containerStateSpec)
	if err != nil {
		return "", err
	}
	patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":%q}}}`,
		sigmak8sapi.AnnotationContainerStateSpec, string(stateSpec))
	return patchData, nil
}

// GenerateContainerStatePatchData can generate patch data to start, stop or pause a container.
func GenerateContainerStatePatchData(containerName string, desireState sigmak8sapi.ContainerState) (string, error) {
	stateSpec := sigmak8sapi.ContainerStateSpec{
		States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
			sigmak8sapi.ContainerInfo{Name: containerName}: desireState,
		},
	}
	return GenerateStatePatchData(stateSpec)
}

// PauseContainer can pause a container by setting desired state to "paused".
func PauseContainer(client clientset.Interface, pod *v1.Pod, namespace string, containerName string) error {
	return patchContainerToDesiredState(client, pod, namespace, containerName,
		"pause container success", sigmak8sapi.ContainerStatePaused)
}

// StartContainer can start a container by setting desired state to "running".
func StartContainer(client clientset.Interface, pod *v1.Pod, namespace string, containerName string) error {
	return patchContainerToDesiredState(client, pod, namespace, containerName,
		"start container success", sigmak8sapi.ContainerStateRunning)
}

// StopContainer can stop a container by setting desired state to "exited".
func StopContainer(client clientset.Interface, pod *v1.Pod, namespace string, containerName string) error {
	return patchContainerToDesiredState(client, pod, namespace, containerName,
		"kill container success", sigmak8sapi.ContainerStateExited)
}

// SuspendContainer can suspend a container by setting desired state to "suspended".
func SuspendContainer(client clientset.Interface, pod *v1.Pod, namespace string, containerName string) error {
	return patchContainerToDesiredState(client, pod, namespace, containerName,
		"suspend container success", sigmak8sapi.ContainerStateSuspended)
}

// UnsuspendContainer can unsuspend a container by setting desired state from "suspended" to "running".
func UnsuspendContainer(client clientset.Interface, pod *v1.Pod, namespace string, containerName string) error {
	return patchContainerToDesiredState(client, pod, namespace, containerName,
		"unsuspend container success", sigmak8sapi.ContainerStateRunning)
}

func patchContainerToDesiredState(client clientset.Interface, pod *v1.Pod, namespace, containerName,
	successStr string, desiredContainerState sigmak8sapi.ContainerState) error {
	patchData, err := GenerateContainerStatePatchData(containerName, desiredContainerState)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Pods(namespace).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
	if err != nil {
		return err
	}

	err = WaitTimeoutForContainerUpdateStatus(client, pod, containerName, 3*time.Minute, successStr, true)
	if err != nil {
		return err
	}

	return nil
}

// PodExec wraps RunKubectl to execute a bash cmd in target pod
func PodExec(pod *v1.Pod, bashExec string) (string, error) {
	return framework.RunKubectl("exec", fmt.Sprintf("--namespace=%s", pod.Namespace), pod.Name, "--", "/bin/sh", "-c", bashExec)
}

// Waits an extended restart of time (PodContainerStartTimeout) for the container in pod to become running.
func WaitForPodContainerRestartInNamespace(c clientset.Interface, pod *v1.Pod, podStartTime time.Time) error {
	return WaitTimeoutForPodContainerRestartInNamespace(c, pod.Name, pod.Namespace, PodContainerStartTimeout, podStartTime)
}

func WaitTimeoutForPodContainerRestartInNamespace(c clientset.Interface, podName, namespace string, timeout time.Duration, podStartTime time.Time) error {
	return wait.PollImmediate(Poll, timeout, containerRestart(c, podName, namespace, podStartTime))
}

// containerRestart get pod from apiserver, if container status is running and after pod start time, return true
func containerRestart(c clientset.Interface, podName, namespace string, podStartTime time.Time) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, value := range pod.Status.ContainerStatuses {
			if value.State.Running == nil {
				return false, nil
			}
			if !value.State.Running.StartedAt.After(podStartTime) {
				return false, nil
			}
		}
		return true, nil
	}
}

// WaitTimeoutForPodFinalizer check whether expect finalizer is patched into pod within the timeout.
func WaitTimeoutForPodFinalizer(f *framework.Framework, podName, expectFinalizer string, timeout time.Duration) error {
	return wait.PollImmediate(3*time.Second, timeout, checkExpectFinalizer(f, podName, expectFinalizer))
}

// checkExpectFinalizer() check Finalizer is existed.
func checkExpectFinalizer(f *framework.Framework, podName string, expectFinalizer string) wait.ConditionFunc {
	return func() (bool, error) {
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, finalizer := range getPod.Finalizers {
			if finalizer == expectFinalizer {
				return true, nil
			}
		}
		framework.Logf("Pod %v finalizer:%v", podName, getPod.Finalizers)
		return false, nil
	}
}

// Get container's update status from pod.
func GetContainerUpdateStatus(pod *v1.Pod, containerName string) *sigmak8sapi.ContainerStatus {
	updateStatusStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	if !exists {
		framework.Logf("[GetContainerUpdateStatus] update status doesn't exist")
		return nil
	}
	framework.Logf("[GetContainerUpdateStatus] updateStatusStr: %v", updateStatusStr)
	containerStatus := sigmak8sapi.ContainerStateStatus{}
	if err := json.Unmarshal([]byte(updateStatusStr), &containerStatus); err != nil {
		framework.Logf("[GetContainerUpdateStatus] unmarshal failed")
		return nil
	}
	for containerInfo, containerStatus := range containerStatus.Statuses {
		if containerInfo.Name == containerName {
			return &containerStatus
		}
	}
	framework.Logf("[GetContainerUpdateStatus] No update status found for container: %s", containerName)
	return nil
}

// WaitTimeoutForContainerUpdateRetryCount check pod's update status retryCount to wait the action failed.
func WaitTimeoutForContainerUpdateRetryCount(client clientset.Interface, pod *v1.Pod, containerName string, timeout time.Duration, retryCount int) error {
	options := metav1.SingleObject(metav1.ObjectMeta{Name: pod.Name})
	w, err := client.CoreV1().Pods(pod.Namespace).Watch(options)
	if err != nil {
		return err
	}
	_, err = watch.Until(timeout, w, func(event watch.Event) (bool, error) {
		switch pod := event.Object.(type) {
		case *v1.Pod:
			containerStatus := GetContainerUpdateStatus(pod, containerName)
			if containerStatus == nil {
				return false, nil
			}
			if containerStatus.RetryCount > retryCount || containerStatus.RetryCount == retryCount {
				framework.Logf("[WaitTimeoutForContainerUpdateRetryCount] container's updateStatus is updated successfully")
				return true, nil
			}
			return false, nil

		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("[WaitTimeoutForContainerUpdateRetryCount] timeout")
	}

	return nil
}

//  DeletePodWithTimeout delete pod by using k8s api, and check whether pod is really deleted within the timeout.
func DeletePodWithTimeout(client clientset.Interface, pod *v1.Pod, timeout time.Duration) error {
	err := client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, metav1.NewDeleteOptions(5))
	if err != nil {
		return err
	}
	t := time.Now()
	for {
		_, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), "not found") {
			framework.Logf("pod %s has been removed", pod.Name)
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("Gave up waiting for pod %s is removed after %v seconds",
				pod.Name, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether pod %s is removed", pod.Name)
		time.Sleep(5 * time.Second)
	}
}
