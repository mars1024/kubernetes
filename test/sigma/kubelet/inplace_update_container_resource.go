package kubelet

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

type InplaceUpdateContainerResourceTestCase struct {
	pod             *v1.Pod
	patchData       string
	expectSuccess   bool
	expectDiskQuota string
}

func doInplaceUpdateContainerResourceTestCase(f *framework.Framework, testCase *InplaceUpdateContainerResourceTestCase) {
	// set container init resource and mutate image before create.
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step 1: Create pod.
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step 2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step 3: Update container resource requirement to trigger update action.
	By("update container resource requirement")
	_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	framework.Logf("Patch pod return: %v", err)

	if testCase.expectSuccess {
		Expect(err).NotTo(HaveOccurred(), "patch pod err")
	}

	By("wait timeout for container update status")
	if testCase.expectSuccess {
		// Step 4: check container status from annotation.
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, "pod-base", 3*time.Minute, "update container success", true)
		Expect(err).NotTo(HaveOccurred(), "\"update container success\" does not appear in container update status")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		state, _ := getPod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
		framework.Logf("inplace update state after updating is: %s", state)
		Expect(state).Should(Equal(sigmak8sapi.InplaceUpdateStateSucceeded))

		if testCase.expectDiskQuota != "" {
			time.Sleep(time.Second * 30)
			// Check DiskQuota.
			checkCommand := "df -h | grep '/$' | awk '{print $2}'"
			result := f.ExecShellInContainer(testPod.Name, containerName, checkCommand)
			framework.Logf("command resut: %v", result)
			checkResult(checkMethodEqual, result, []string{testCase.expectDiskQuota})
		}
	} else {
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, "pod-base", 3*time.Minute, "failed to update container", false)
		Expect(err).NotTo(HaveOccurred(), "\"failed to update container\" does not appear in container update status")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		state, _ := getPod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
		framework.Logf("inplace update state after updating is: %s", state)
		Expect(state).Should(Equal(sigmak8sapi.InplaceUpdateStateFailed))
	}
}

var _ = Describe("[sigma-kubelet] inplace_update_001 update container's resource should be ok", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	initResource := getResourceRequirements(getResourceList("500m", "128Mi"), getResourceList("500m", "128Mi"))
	It("update container's resource requirement without QoS class change", func() {
		testCase := InplaceUpdateContainerResourceTestCase{
			pod:           generateRunningPodWithInitResource(initResource),
			patchData:     fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}},"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"cpu": "1000m", "memory": "256Mi"}, "limits": {"cpu": "1000m", "memory": "256Mi"}}}]}}`, sigmak8sapi.AnnotationPodInplaceUpdateState, sigmak8sapi.InplaceUpdateStateAccepted),
			expectSuccess: true,
		}

		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
})

var _ = Describe("[sigma-kubelet] inplace_update_002 update container's diskquota should be ok(DiskQuotaMode: .*)", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("update container's diskQuota when EphemeralStorage is changed", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		patchData := fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}},"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"ephemeral-storage": "5Gi"}, "limits":{"ephemeral-storage": "5Gi"}}}]}}`,
			sigmak8sapi.AnnotationPodInplaceUpdateState, sigmak8sapi.InplaceUpdateStateAccepted)
		testCase := InplaceUpdateContainerResourceTestCase{
			pod:             pod,
			patchData:       patchData,
			expectSuccess:   true,
			expectDiskQuota: "5.0G",
		}

		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
})

var _ = Describe("[sigma-kubelet] inplace_update_003 update container's diskquota should be ok(DiskQuotaMode: /)", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("update container's diskQuota when EphemeralStorage is changed", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = fmt.Sprintf(`{"containers":[{"name":"%s","hostConfig":{"diskQuotaMode":"/"}}]}`, pod.Spec.Containers[0].Name)
		patchData := fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}},"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"ephemeral-storage": "5Gi"}, "limits":{"ephemeral-storage": "5Gi"}}}]}}`,
			sigmak8sapi.AnnotationPodInplaceUpdateState, sigmak8sapi.InplaceUpdateStateAccepted)
		testCase := InplaceUpdateContainerResourceTestCase{
			pod:             pod,
			patchData:       patchData,
			expectSuccess:   true,
			expectDiskQuota: "5.0G",
		}

		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
})

var _ = Describe("[sigma-kubelet] [Disruptive] inplace_update_004 update container's resource with a large value, should return error", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	initResource := getResourceRequirements(getResourceList("500m", "128Mi"), getResourceList("500m", "128Mi"))
	It("update container's resource requirement with a large value", func() {
		// Ref: github.com/opencontainers/runc/libcontainer/cgroups/fs/apply_raw.go:L367
		// The maximum allowed cpu-shares is 262144
		testCase := InplaceUpdateContainerResourceTestCase{
			pod:           generateRunningPodWithInitResource(initResource),
			patchData:     fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}},"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"cpu": "2000", "memory": "256Mi"}, "limits": {"cpu": "2000", "memory": "256Mi"}}}]}}`, sigmak8sapi.AnnotationPodInplaceUpdateState, sigmak8sapi.InplaceUpdateStateAccepted),
			expectSuccess: false,
		}

		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
})

var _ = Describe("[sigma-kubelet] inplace_update_005 update container's memory swappiness in annotation should be ok", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	initResource := getResourceRequirements(getResourceList("500m", "128Mi"), getResourceList("500m", "128Mi"))
	It("update container's resource requirement without QoS class change", func() {
		pod := generateRunningPodWithInitResource(initResource)
		testCase := InplaceUpdateContainerResourceTestCase{
			pod: pod,
			patchData: fmt.Sprintf(`{"metadata": {"annotations": {%q:%q, %q:%q}}}`,
				sigmak8sapi.AnnotationPodInplaceUpdateState, sigmak8sapi.InplaceUpdateStateAccepted,
				sigmak8sapi.AnnotationPodAllocSpec, fmt.Sprintf(`{"containers":[{"name":"%s","hostConfig":{"memorySwappiness": 0}}]}`, pod.Spec.Containers[0].Name),
			),
			expectSuccess: true,
		}

		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
})
