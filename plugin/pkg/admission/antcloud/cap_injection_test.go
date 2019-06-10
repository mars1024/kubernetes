package antcloud

import (
	api "k8s.io/kubernetes/pkg/apis/core"
	"testing"
)

func TestInjectPodCap(t *testing.T) {
	pod := makePod("sys_admin")
	pod.Spec.Containers[0].SecurityContext.Capabilities = &api.Capabilities{
		Drop: []api.Capability{"sys_module"},
	}
	err := InjectPodCap(pod)
	if err != nil {
		t.Errorf("inject failed")
	}
	containers := pod.Spec.Containers
	matched := false
	for _, container := range containers {
		for _, c := range container.SecurityContext.Capabilities.Drop {
			if c == "sys_resource" {
				matched = true
			}
		}
	}
	if !matched {
		t.Errorf("failed to find sys_resource cap")
	}
}

func TestInjectPodCap_Resource(t *testing.T) {
	pod := makePod("sys_admin")
	pod.Spec.Containers[0].SecurityContext.Capabilities.Add = []api.Capability{
		api.Capability("sys_resource"),
	}
	err := InjectPodCap(pod)
	if err != nil {
		t.Errorf("inject failed")
	}
	containers := pod.Spec.Containers
	matched := false
	for _, container := range containers {
		for _, c := range container.SecurityContext.Capabilities.Drop {
			if c == "sys_resource" {
				matched = true
			}
		}
	}
	if matched {
		t.Errorf("should not find the sys_resource cap")
	}
}

func TestInjectPodCap_2(t *testing.T) {
	pod := makeEmptyPod()
	err := InjectPodCap(pod)
	if err != nil {
		t.Errorf("inject failed")
	}
	containers := pod.Spec.Containers
	matched := false
	for _, container := range containers {
		for _, c := range container.SecurityContext.Capabilities.Drop {
			if c == "sys_resource" {
				matched = true
			}
		}
	}
	if !matched {
		t.Errorf("failed to find sys_resource cap")
	}
}

func TestInjectPodCap_3(t *testing.T) {
	pod := makeEmptyPod()
	p := false
	pod.Spec.Containers[0].SecurityContext = &api.SecurityContext{
		Privileged: &p,
	}
	err := InjectPodCap(pod)
	if err != nil {
		t.Errorf("inject failed")
	}
	containers := pod.Spec.Containers
	matched := false
	for _, container := range containers {
		for _, c := range container.SecurityContext.Capabilities.Drop {
			if c == "sys_resource" {
				matched = true
			}
		}
	}
	if !matched {
		t.Errorf("failed to find sys_resource cap")
	}
}

func makePod(cap string) *api.Pod {
	pod := &api.Pod{
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					SecurityContext: &api.SecurityContext{
						Capabilities: &api.Capabilities{
							Add: []api.Capability{
								api.Capability(cap),
							},
						},
					},
				},
			},
		},
	}
	return pod
}

func makeEmptyPod() *api.Pod {
	pod := &api.Pod{
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name: "test",
				},
			},
		},
	}
	return pod
}
