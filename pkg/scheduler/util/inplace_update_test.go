package util

import (
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

func TestStoreLastSpecIfNeeded(t *testing.T) {
	tests := []struct {
		oldPod         *v1.Pod
		newPod         *v1.Pod
		expLastSpecStr string
	}{
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodInplaceUpdateState: sigmak8sapi.InplaceUpdateStateCreated,
					},
				},
				Spec: v1.PodSpec{},
			},
			expLastSpecStr: "{\"containers\":[{\"name\":\"c1\",\"resources\":{\"requests\":{\"cpu\":\"2\",\"memory\":\"100Mi\"}}},{\"name\":\"c2\",\"resources\":{\"requests\":{\"cpu\":\"3\",\"memory\":\"200Mi\"}}}]}",
		},
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodInplaceUpdateState: sigmak8sapi.InplaceUpdateStateCreated,
						sigmak8sapi.AnnotationPodLastSpec:           "{\"containers\":[{\"name\":\"c1\",\"resources\":{\"requests\":{\"cpu\":\"3\",\"memory\":\"100Mi\"}}},{\"name\":\"c2\",\"resources\":{\"requests\":{\"cpu\":\"6\",\"memory\":\"200Mi\"}}}]}",
					},
				},
				Spec: v1.PodSpec{},
			},
			expLastSpecStr: "{\"containers\":[{\"name\":\"c1\",\"resources\":{\"requests\":{\"cpu\":\"1\",\"memory\":\"100Mi\"}}},{\"name\":\"c2\",\"resources\":{\"requests\":{\"cpu\":\"2\",\"memory\":\"200Mi\"}}}]}",
		},
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: v1.PodSpec{},
			},
			expLastSpecStr: "",
		},
	}

	for i, test := range tests {
		StoreLastSpecIfNeeded(test.oldPod, test.newPod)
		lastSpecStr, _ := test.newPod.Annotations[sigmak8sapi.AnnotationPodLastSpec]
		if lastSpecStr != test.expLastSpecStr {
			t.Errorf("Case[%d], lastSpecStr: %s not equal to expLastSpecStr: %s", i, lastSpecStr, test.expLastSpecStr)
		}
	}
}

func TestIsCPUResourceChanged(t *testing.T) {
	tests := []struct {
		oldPod *v1.Pod
		newPod *v1.Pod
		expRet bool
	}{
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			expRet: true,
		},
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			expRet: false,
		},
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					}},
			},
			expRet: true,
		},
		{
			oldPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
			newPod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "n1",
					Containers: []v1.Container{
						{
							Name: "c1",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
								},
							},
						},
						{
							Name: "c2",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
									v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
								},
							},
						},
					}},
			},
			expRet: true,
		},
	}

	for i, test := range tests {
		ret := IsCPUResourceChanged(&test.oldPod.Spec, &test.newPod.Spec)
		if ret != test.expRet {
			t.Errorf("Case[%d], ret: %t not equal to expRet: %t", i, ret, test.expRet)
		}
	}
}
