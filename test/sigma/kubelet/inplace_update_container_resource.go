package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type InplaceUpdateContainerResourceTestCase struct {
	pod           *v1.Pod
	patchData     string
	expectSuccess bool
}

func doInplaceUpdateContainerResourceTestCase(f *framework.Framework, testCase *InplaceUpdateContainerResourceTestCase) {
	// set container init resource and mutate image before create.
	pod := testCase.pod

	// Step 1: Create pod.
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step 2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 30*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step 3: Update container resource requirement to trigger update action.
	By("update container resource requirement")
	_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	framework.Logf("Patch pod return: %v", err)

	if testCase.expectSuccess {
		Expect(err).NotTo(HaveOccurred(), "patch pod err")
	} else {
		Expect(err).To(HaveOccurred(), "should get validate err")
	}

	By("wait timeout for container update status")
	if testCase.expectSuccess {
		// Step 4: check container status from annotation.
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, "pod-base", 1*time.Minute, "update container success", true)
		Expect(err).NotTo(HaveOccurred(), "\"update container success\" does not appear in container update status")
	}
}

var _ = Describe("[sigma-kubelet] inplace_update_001 update container's resource should be ok", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	initResource := getResourceRequirements(getResourceList("500m", "128Mi"), getResourceList("500m", "128Mi"))
	It("update container's resource requirement without QoS class change", func() {
		testCase := InplaceUpdateContainerResourceTestCase{
			pod:           generateRunningPodWithInitResource(initResource),
			patchData:     `{"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"cpu": "1000m", "memory": "256Mi"}, "limits": {"cpu": "1000m", "memory": "256Mi"}}}]}}`,
			expectSuccess: true,
		}
		doInplaceUpdateContainerResourceTestCase(f, &testCase)
	})
	// It("update container's resource requirement with QoS class change", func() {
	// 	testCase := InplaceUpdateContainerResourceTestCase{
	// 		pod:           generateRunningPodWithInitResource(initResource),
	// 		patchData:     `{"spec":{"containers":[{"name":"pod-base","resources":{"requests": {"cpu": "1000m", "memory": "256Mi"}, "limits": {"cpu": "2000m", "memory": "256Mi"}}}]}}`,
	// 		expectSuccess: false,
	// 	}
	// 	doInplaceUpdateContainerResourceTestCase(f, &testCase)
	// })
})
