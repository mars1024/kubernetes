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

package autopilot

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func TestKubeletParam_GetNodeInfo(t *testing.T) {
	type fields struct {
		Node           *v1.Node
		UpdateInterval time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   *v1.Node
	}{
		{
			name: "No node",
			fields: fields{
				Node:           nil,
				UpdateInterval: 10 * time.Second,
			},
			want: nil,
		},
		{
			name: "Normal node",
			fields: fields{
				Node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{},
				},
				UpdateInterval: 10 * time.Second,
			},
			want: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "127.0.0.1",
				},
				Status: v1.NodeStatus{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := &KubeletParam{
				Node:           tt.fields.Node,
				UpdateInterval: tt.fields.UpdateInterval,
			}
			if got := sp.GetNodeInfo(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KubeletParam.GetNodeInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubeletParam_GetUpdateInterval(t *testing.T) {
	type fields struct {
		Node           *v1.Node
		UpdateInterval time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
	}{
		{
			name: "Not to set interval",
			fields: fields{
				Node: nil,
			},
			want: 0,
		},
		{
			name: "Normal interval",
			fields: fields{
				Node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{},
				},
				UpdateInterval: 10 * time.Second,
			},
			want: 10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := &KubeletParam{
				Node:           tt.fields.Node,
				UpdateInterval: tt.fields.UpdateInterval,
			}
			if got := sp.GetUpdateInterval(); got != tt.want {
				t.Errorf("KubeletParam.GetUpdateInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

var config = &rest.Config{
	Host:    "http://127.0.0.1",
	QPS:     -1,
	Timeout: time.Second,
}

var heartbeatClient, _ = v1core.NewForConfig(config)

func TestNewAnnotationPara(t *testing.T) {
	type args struct {
		heartbeatClient v1core.CoreV1Interface
		nodeName        string
	}
	tests := []struct {
		name string
		args args
		want *AnnotationPara
	}{
		{
			name: "No heartbeatClient",
			args: args{
				heartbeatClient: nil,
				nodeName:        "127.0.0.1",
			},
			want: &AnnotationPara{
				heartbeatClient: nil,
				nodeName:        "127.0.0.1",
			},
		},
		{
			name: "Normal heartbeatClient",
			args: args{
				heartbeatClient: heartbeatClient,
				nodeName:        "127.0.0.1",
			},
			want: &AnnotationPara{
				heartbeatClient: heartbeatClient,
				nodeName:        "127.0.0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAnnotationPara(tt.args.heartbeatClient, tt.args.nodeName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAnnotationPara() = %v, want %v", got, tt.want)
			}
		})
	}
}

var annotationParaObject = NewAnnotationPara(heartbeatClient, "127.0.0.1")

func TestAnnotationPara_Sync(t *testing.T) {
	type fields struct {
		heartbeatClient v1core.CoreV1Interface
		nodeName        string
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[string]string
		wantErr bool
	}{
		{
			name: "No client",
			fields: fields{
				heartbeatClient: nil,
				nodeName:        "127.0.0.1",
			},
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name: "Normal client with error return",
			fields: fields{
				heartbeatClient: heartbeatClient,
				nodeName:        "127.0.0.1",
			},
			want:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := &AnnotationPara{
				heartbeatClient: tt.fields.heartbeatClient,
				nodeName:        tt.fields.nodeName,
			}
			got, err := ap.Sync()
			if (err != nil) != tt.wantErr {
				t.Errorf("AnnotationPara.Sync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AnnotationPara.Sync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestControllers_Start(t *testing.T) {
	type fields struct {
		syncer                    AnnotationSyncer
		services                  map[string]Service
		nodeStatusUpdateFrequency time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "empty object",
			fields: fields{
				syncer:                    annotationParaObject,
				services:                  make(map[string]Service),
				nodeStatusUpdateFrequency: 10 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controllers{
				syncer:                    tt.fields.syncer,
				services:                  tt.fields.services,
				nodeStatusUpdateFrequency: tt.fields.nodeStatusUpdateFrequency,
			}
			c.Start()
		})
	}
}

func TestControllers_Register(t *testing.T) {
	type fields struct {
		syncer                    AnnotationSyncer
		services                  map[string]Service
		nodeStatusUpdateFrequency time.Duration
	}
	type args struct {
		name    string
		service Service
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "empty object",
			fields: fields{
				services: make(map[string]Service),
			},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controllers{
				syncer:                    tt.fields.syncer,
				services:                  tt.fields.services,
				nodeStatusUpdateFrequency: tt.fields.nodeStatusUpdateFrequency,
			}
			if err := c.Register(tt.args.name, tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("Controllers.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
