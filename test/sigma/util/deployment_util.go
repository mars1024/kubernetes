package util

import (
	"encoding/json"
	"io/ioutil"
	"time"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/apimachinery/pkg/api/errors"
	"fmt"
)

// LoadDeploymentFromFile create a deployment object from file
func LoadDeploymentFromFile(file string) (*v1beta1.Deployment, error) {
	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var deployment *v1beta1.Deployment
	err = json.Unmarshal(fileContent, &deployment)
	if err != nil {
		return nil, err
	}
	if env.GetTester() == env.TesterJituan {
		deployment.Spec.Template.Spec.Tolerations = append(deployment.Spec.Template.Spec.Tolerations, v1.Toleration{
			Key:      sigmak8sapi.LabelResourcePool,
			Operator: v1.TolerationOpExists,
			Effect:   v1.TaintEffectNoSchedule,
		})
	}
	return deployment, nil
}

func WaitTimeoutForPodReplicas(client clientset.Interface, deployment *v1beta1.Deployment, expectReplicas int, checkPeriod, timeout time.Duration) error {
	return wait.PollImmediate(checkPeriod, timeout, checkPodReplicas(client, deployment, expectReplicas))
}

func checkPodReplicas(client clientset.Interface, deployment *v1beta1.Deployment, expectReplicas int) wait.ConditionFunc {
	return func() (bool, error) {
		pods, err := ListDeploymentPods(client, deployment)
		if err != nil {
			return false, err
		}

		framework.Logf("deployment %v/%v pods number is %v, expect %v", deployment.Namespace, deployment.Name, len(pods.Items), expectReplicas)
		if len(pods.Items) == expectReplicas {
			return true, nil
		}
		return false, nil
	}
}

// ListDeploymentPods get pods from cache
// while GetDeploymentPods get pods from apiserver
func ListDeploymentPods(client clientset.Interface, deployment *v1beta1.Deployment) (*v1.PodList, error) {
	return client.CoreV1().Pods(deployment.Namespace).List(
		metav1.ListOptions{
			LabelSelector: labels.FormatLabels(deployment.Spec.Selector.MatchLabels),
		})
}

func GetDeploymentPods(client clientset.Interface, deployment *v1beta1.Deployment) ([]*v1.Pod, error) {
	pods, err := ListDeploymentPods(client, deployment)
	if err != nil {
		return nil, err
	}

	var latestPods []*v1.Pod
	for _, pod := range pods.Items {
		latestPod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		latestPods = append(latestPods, latestPod)
	}
	return latestPods, nil
}

func WaitDeploymentPodsRunning(client clientset.Interface, deployment *v1beta1.Deployment, expectedReplicas int) error {
	err := WaitTimeoutForPodReplicas(client, deployment, expectedReplicas, time.Second, 3*time.Minute)
	if err != nil {
		return err
	}

	pods, err := ListDeploymentPods(client, deployment)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		err = WaitTimeoutForPodStatus(client, &pod, v1.PodRunning, 3*time.Minute)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateDeploymentReplicas(client clientset.Interface, deployment *v1beta1.Deployment, expectedReplicas int32) error {
	for i := 1; i <= 5; i++ {
		deployment, err := client.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		deployment.Spec.Replicas = &expectedReplicas
		_, err = client.ExtensionsV1beta1().Deployments(deployment.Namespace).Update(deployment)
		if err != nil {
			if errors.IsConflict(err) {
				time.Sleep(time.Second)
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("update deployment replicas failed finally")
}

func DeleteDeploymentPods(client clientset.Interface, namespace, name string) {
	/* 'DeletePropagationForceBackground' doesn't work as expected
	policy := metav1.DeletePropagationForceBackground
	client.ExtensionsV1beta1().Deployments(deployment.Namespace).Delete(
		deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy})
	*/
	deploy, err := client.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		framework.Logf("get deployment %v/%v failed, err: %v", deploy.Namespace, deploy.Name, err)
	}
	var replicas int32 = 0
	deploy.Spec.Replicas = &replicas
	_, err = client.ExtensionsV1beta1().Deployments(namespace).Update(deploy)
	if err != nil {
		framework.Logf("delete deployment pods %v/%v failed, err: %v", namespace, name)
	}
}
