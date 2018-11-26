/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adaptivescale

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/configmap"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	podtest "k8s.io/kubernetes/pkg/kubelet/pod/testing"
	"k8s.io/kubernetes/pkg/kubelet/secret"
)

func Test_transferExecPeriodtoInt(t *testing.T) {
	type args struct {
		execPeriod string
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "correct case",
			args: args{
				execPeriod: "20s",
			},
			want: 20 * time.Second,
		},
		{
			name: "incorrect case",
			args: args{
				execPeriod: "s20s",
			},
			want: 10 * time.Second,
		},
		{
			name: "incorrect case",
			args: args{
				execPeriod: "0s",
			},
			want: 1 * time.Second,
		},
		{
			name: "incorrect case",
			args: args{
				execPeriod: "-100s",
			},
			want: 1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transferExecPeriodtoInt(tt.args.execPeriod); got != tt.want {
				t.Errorf("transferExecPeriodtoInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStartAutopilotParameters(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "correct case",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: "30s"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: 30 * time.Second},
			wantErr: false,
		},
		{
			name: "correct case",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "true"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: 10 * time.Second},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getStartAutopilotParameters(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("getStartAutopilotParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getStartAutopilotParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAutopilotServiceAnnotations(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "correct case 01",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: "30s"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: 30 * time.Second},
			wantErr: false,
		},
		{
			name: "correct case 02",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: "30s"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "true", autopilotServiceExecPeriodKey: 30 * time.Second},
			wantErr: false,
		},
		{
			name: "correct case 03",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "false"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "false"},
			wantErr: false,
		},
		{
			name: "correct case 04",
			args: args{
				annotations: map[string]string{autopilotServiceKey: "haha"},
			},
			want:    map[string]interface{}{autopilotServiceKey: "false"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAutopilotServiceAnnotations(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAutopilotServiceAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAutopilotServiceAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_startAutopilot(t *testing.T) {
	tests := []struct {
		name string
		args *ResourceAdjustController
	}{
		{
			name: "run stop autopilot",
			args: mockResourceAdjustControllerWithRunStatus(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startAutopilot(tt.args, 10*time.Second)
		})
	}
}

func mockResourceAdjustControllerWithRunStatus(runStatus bool) *ResourceAdjustController {
	cpm := podtest.NewMockCheckpointManager()
	podManager := kubepod.NewBasicPodManager(podtest.NewFakeMirrorClient(), secret.NewFakeManager(), configmap.NewFakeManager(), cpm)
	return NewController(
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "127.0.0.1",
			},
			Status: v1.NodeStatus{
				Conditions: []v1.NodeCondition{
					{Type: v1.NodeOutOfDisk, Status: v1.ConditionTrue},
				},
			},
		},
		podManager,
		&fakeSummaryProvider{},
		mockContainerRuntime{},
		runStatus,
		1,
	)
}

func Test_stopAutopilot(t *testing.T) {
	tests := []struct {
		name string
		args *ResourceAdjustController
	}{
		{
			name: "run stop autopilot",
			args: mockResourceAdjustControllerWithRunStatus(false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopAutopilot(tt.args)
		})
	}
}
