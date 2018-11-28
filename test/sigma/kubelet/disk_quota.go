package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type diskQuotaTestCase struct {
	pod            *v1.Pod
	checkCommand   string
	resultKeywords []string
	checkMethod    string
}

func doDiskQuotaTestCase(f *framework.Framework, testCase *diskQuotaTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Check command's result.
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command resut: %v", result)
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)
}

var _ = Describe("[sigma-kubelet][disk-quota] check disk quota", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	// Only if limitEphemeralStorage equals requestEphemeralStorage,  container has diskquota.
	It("[smoke] check disk quota: limitEphemeralStorage = requestEphemeralStorage", func() {
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
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"2.0G"},
			checkMethod:    checkMethodEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// If limitEphemeralStorage != requestEphemeralStorage, the container's diskquota is the size as physical machine.
	It("check disk quota: limitEphemeralStorage != requestEphemeralStorage", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G", "2.0G"},
			checkMethod:    checkMethodNotEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// If only limitEphemeralStorage defined, the requestEphemeralStorage will be set equal to limitEphemeralStorage.
	// So diskquota will be set.
	It("check disk quota: only limitEphemeralStorage", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G"},
			checkMethod:    checkMethodEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// If only requestEphemeralStorage defined, diskquota won't be set.
	It("check disk quota: only requestEphemeralStorage", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G"},
			checkMethod:    checkMethodNotEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// If limitEphemeralStorage is set and requestEphemeralStorage is 0, diskquota won't be set.
	It("check disk quota: only limitEphemeralStorage and 0 requestEphemeralStorage", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("0Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G"},
			checkMethod:    checkMethodNotEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// If limitEphemeralStorage and requestEphemeralStorage is not set, diskquota also won't be set.
	It("check disk quota: no requestEphemeralStorage and no requestEphemeralStorage", func() {
		pod := generateRunningPod()
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G", "2.0G"},
			checkMethod:    checkMethodNotEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
})
