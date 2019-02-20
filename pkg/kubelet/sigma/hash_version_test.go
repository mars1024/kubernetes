package sigma

import (
	"testing"

	"github.com/stretchr/testify/assert"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

func TestHackContainerHashFunc(t *testing.T) {
	testCases := []struct {
		origin   string
		expected string
		hackFunc hashutil.DecorateFunc
	}{
		{
			`(*v1.Container){Name:(string)business Image:(string)reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1 Command:([]string)<nil> Args:([]string)<nil> WorkingDir:(string) Ports:([]v1.ContainerPort)<nil> EnvFrom:([]v1.EnvFromSource)<nil> Env:([]v1.EnvVar)[{Name:(string)DEMO_GREETING Value:(string)aaaaa ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO2 Value:(string)bbbbb ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO3 Value:(string)cccccc ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO4 Value:(string)1111dddddd ValueFrom:(*v1.EnvVarSource)<nil>}] Resources:(v1.ResourceRequirements){Limits:(v1.ResourceList)<nil> Requests:(v1.ResourceList)<nil>} VolumeMounts:([]v1.VolumeMount)[{Name:(string)default-token-hwfv9 ReadOnly:(bool)true MountPath:(string)/var/run/secrets/kubernetes.io/serviceaccount SubPath:(string) MountPropagation:(*v1.MountPropagationMode)<nil>}] VolumeDevices:([]v1.VolumeDevice)<nil> LivenessProbe:(*v1.Probe)<nil> ReadinessProbe:(*v1.Probe)<nil> Lifecycle:(*v1.Lifecycle)<nil> TerminationMessagePath:(string)/dev/termination-log TerminationMessagePolicy:(v1.TerminationMessagePolicy)File ImagePullPolicy:(v1.PullPolicy) SecurityContext:(*v1.SecurityContext){Capabilities:(*v1.Capabilities){Add:([]v1.Capability)<nil> Drop:([]v1.Capability)<nil>} Privileged:(*bool)false SELinuxOptions:(*v1.SELinuxOptions)<nil> RunAsUser:(*int64)<nil> RunAsGroup:(*int64)<nil> RunAsNonRoot:(*bool)<nil> ReadOnlyRootFilesystem:(*bool)false AllowPrivilegeEscalation:(*bool)<nil> ProcMount:(*v1.ProcMountType)Default} Stdin:(bool)false StdinOnce:(bool)false TTY:(bool)false}`,
			`(*v1.Container){Name:(string)business Image:(string)reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1 Command:([]string)<nil> Args:([]string)<nil> WorkingDir:(string) Ports:([]v1.ContainerPort)<nil> EnvFrom:([]v1.EnvFromSource)<nil> Env:([]v1.EnvVar)[{Name:(string)DEMO_GREETING Value:(string)aaaaa ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO2 Value:(string)bbbbb ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO3 Value:(string)cccccc ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO4 Value:(string)1111dddddd ValueFrom:(*v1.EnvVarSource)<nil>}] Resources:(v1.ResourceRequirements){Limits:(v1.ResourceList)<nil> Requests:(v1.ResourceList)<nil>} VolumeMounts:([]v1.VolumeMount)[{Name:(string)default-token-hwfv9 ReadOnly:(bool)true MountPath:(string)/var/run/secrets/kubernetes.io/serviceaccount SubPath:(string) MountPropagation:(*v1.MountPropagationMode)<nil>}] VolumeDevices:([]v1.VolumeDevice)<nil> LivenessProbe:(*v1.Probe)<nil> ReadinessProbe:(*v1.Probe)<nil> Lifecycle:(*v1.Lifecycle)<nil> TerminationMessagePath:(string)/dev/termination-log TerminationMessagePolicy:(v1.TerminationMessagePolicy)File ImagePullPolicy:(v1.PullPolicy) SecurityContext:(*v1.SecurityContext){Capabilities:(*v1.Capabilities){Add:([]v1.Capability)<nil> Drop:([]v1.Capability)<nil>} Privileged:(*bool)false SELinuxOptions:(*v1.SELinuxOptions)<nil> RunAsUser:(*int64)<nil> RunAsGroup:(*int64)<nil> RunAsNonRoot:(*bool)<nil> ReadOnlyRootFilesystem:(*bool)false AllowPrivilegeEscalation:(*bool)<nil>} Stdin:(bool)false StdinOnce:(bool)false TTY:(bool)false}`,
			hackGoString112to110,
		},
		{
			`(*v1.Container){Name:(string)business Image:(string)reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1 Command:([]string)<nil> Args:([]string)<nil> WorkingDir:(string) Ports:([]v1.ContainerPort)<nil> EnvFrom:([]v1.EnvFromSource)<nil> Env:([]v1.EnvVar)[{Name:(string)DEMO_GREETING Value:(string)aaaaa ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO2 Value:(string)bbbbb ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO3 Value:(string)cccccc ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO4 Value:(string)1111dddddd ValueFrom:(*v1.EnvVarSource)<nil>}] Resources:(v1.ResourceRequirements){Limits:(v1.ResourceList)<nil> Requests:(v1.ResourceList)<nil>} VolumeMounts:([]v1.VolumeMount)[{Name:(string)default-token-hwfv9 ReadOnly:(bool)true MountPath:(string)/var/run/secrets/kubernetes.io/serviceaccount SubPath:(string) MountPropagation:(*v1.MountPropagationMode)<nil>}] VolumeDevices:([]v1.VolumeDevice)<nil> LivenessProbe:(*v1.Probe)<nil> ReadinessProbe:(*v1.Probe)<nil> Lifecycle:(*v1.Lifecycle)<nil> TerminationMessagePath:(string)/dev/termination-log TerminationMessagePolicy:(v1.TerminationMessagePolicy)File ImagePullPolicy:(v1.PullPolicy) SecurityContext:(*v1.SecurityContext){Capabilities:(*v1.Capabilities){Add:([]v1.Capability)<nil> Drop:([]v1.Capability)<nil>} Privileged:(*bool)false SELinuxOptions:(*v1.SELinuxOptions)<nil> RunAsUser:(*int64)<nil> RunAsGroup:(*int64)<nil> RunAsNonRoot:(*bool)<nil> ReadOnlyRootFilesystem:(*bool)false AllowPrivilegeEscalation:(*bool)<nil> ProcMount:(*v1.ProcMountType)<nil>} Stdin:(bool)false StdinOnce:(bool)false TTY:(bool)false}`,
			`(*v1.Container){Name:(string)business Image:(string)reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1 Command:([]string)<nil> Args:([]string)<nil> WorkingDir:(string) Ports:([]v1.ContainerPort)<nil> EnvFrom:([]v1.EnvFromSource)<nil> Env:([]v1.EnvVar)[{Name:(string)DEMO_GREETING Value:(string)aaaaa ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO2 Value:(string)bbbbb ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO3 Value:(string)cccccc ValueFrom:(*v1.EnvVarSource)<nil>} {Name:(string)DEMO4 Value:(string)1111dddddd ValueFrom:(*v1.EnvVarSource)<nil>}] Resources:(v1.ResourceRequirements){Limits:(v1.ResourceList)<nil> Requests:(v1.ResourceList)<nil>} VolumeMounts:([]v1.VolumeMount)[{Name:(string)default-token-hwfv9 ReadOnly:(bool)true MountPath:(string)/var/run/secrets/kubernetes.io/serviceaccount SubPath:(string) MountPropagation:(*v1.MountPropagationMode)<nil>}] VolumeDevices:([]v1.VolumeDevice)<nil> LivenessProbe:(*v1.Probe)<nil> ReadinessProbe:(*v1.Probe)<nil> Lifecycle:(*v1.Lifecycle)<nil> TerminationMessagePath:(string)/dev/termination-log TerminationMessagePolicy:(v1.TerminationMessagePolicy)File ImagePullPolicy:(v1.PullPolicy) SecurityContext:(*v1.SecurityContext){Capabilities:(*v1.Capabilities){Add:([]v1.Capability)<nil> Drop:([]v1.Capability)<nil>} Privileged:(*bool)false SELinuxOptions:(*v1.SELinuxOptions)<nil> RunAsUser:(*int64)<nil> RunAsGroup:(*int64)<nil> RunAsNonRoot:(*bool)<nil> ReadOnlyRootFilesystem:(*bool)false AllowPrivilegeEscalation:(*bool)<nil>} Stdin:(bool)false StdinOnce:(bool)false TTY:(bool)false}`,
			hackGoString112to110,
		},
	}

	for _, tc := range testCases {
		actual := tc.hackFunc(tc.origin)
		if actual != tc.expected {
			t.Errorf("hack container hash error, expected: \n %s\nactual: \n%s\n", tc.expected, actual)
		}
	}

}

func TestCompareHashVersion(t *testing.T) {
	testCases := []struct {
		versionStr1  string
		versionStr2  string
		errorOccurs  bool
		expectResult int
	}{
		{
			versionStr1:  "1.12.2",
			versionStr2:  "1.10.2",
			errorOccurs:  false,
			expectResult: 1,
		},
		{
			versionStr1:  "1.12.2",
			versionStr2:  "1.10.5",
			errorOccurs:  false,
			expectResult: 1,
		},
		{
			versionStr1:  "2.11.0",
			versionStr2:  "1.12.5",
			errorOccurs:  false,
			expectResult: 1,
		},
		{
			versionStr1:  "2.11.0",
			versionStr2:  VERSION_LOWEST,
			errorOccurs:  false,
			expectResult: 1,
		},
		{
			versionStr1:  "1.12.2",
			versionStr2:  "1.12.2",
			errorOccurs:  false,
			expectResult: 0,
		},
		{
			versionStr1:  "1.11.3",
			versionStr2:  "1.12.2",
			errorOccurs:  false,
			expectResult: -1,
		},
		{
			versionStr1:  VERSION_LOWEST,
			versionStr2:  "1.0.0",
			errorOccurs:  false,
			expectResult: -1,
		},

		{
			versionStr1:  "1.11.",
			versionStr2:  "1.12.2",
			errorOccurs:  true,
			expectResult: 0,
		},
		{
			versionStr1:  "1.11.0",
			versionStr2:  "1.12",
			errorOccurs:  true,
			expectResult: 0,
		},
	}

	for _, tc := range testCases {
		result, err := CompareHashVersion(tc.versionStr1, tc.versionStr2)
		if tc.errorOccurs && err == nil || !tc.errorOccurs && err != nil {
			t.Errorf("Error occurs, case: %v", tc)
		}
		assert.Equal(t, tc.expectResult, result)
	}

}
