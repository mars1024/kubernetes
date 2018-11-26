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

package sketch

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
)

func Test_snapshoterImpl_GetSummary(t *testing.T) {
	tests := map[string]struct {
		summary interface{}
		want    *sketchapi.SketchSummary
		err     error
		wantErr bool
	}{
		"nil": {
			err:     ErrEmpty,
			wantErr: true,
		},
		"normal": {
			summary: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					Memory: &sketchapi.NodeMemorySketch{
						MemorySketch: sketchapi.MemorySketch{
							AvailableBytes: 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					Memory: &sketchapi.NodeMemorySketch{
						MemorySketch: sketchapi.MemorySketch{
							AvailableBytes: 1024,
						},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSnapshotImpl(tt.summary)
			got, err := s.GetSummary()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("snapshoterImpl.GetSummary() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.err != nil && err != tt.err {
					t.Errorf("snapshoterImpl.GetSummary() error = %v, expect err: %v", err, tt.err)
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("snapshoterImpl.GetSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_snapshoterImpl_GetNodeSketch(t *testing.T) {
	tests := map[string]struct {
		summary interface{}
		want    *sketchapi.NodeSketch
		err     error
		wantErr bool
	}{
		"nil": {
			err:     ErrEmpty,
			wantErr: true,
		},
		"normal": {
			summary: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					Memory: &sketchapi.NodeMemorySketch{
						MemorySketch: sketchapi.MemorySketch{
							AvailableBytes: 1024,
						},
					},
				},
			},
			want: &sketchapi.NodeSketch{
				Name: "test",
				Memory: &sketchapi.NodeMemorySketch{
					MemorySketch: sketchapi.MemorySketch{
						AvailableBytes: 1024,
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSnapshotImpl(tt.summary)
			got, err := s.GetNodeSketch()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("snapshoterImpl.GetNodeSketch() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.err != nil && err != tt.err {
					t.Errorf("snapshoterImpl.GetNodeSketch() error = %v, expect err: %v", err, tt.err)
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("snapshoterImpl.GetNodeSketch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_snapshoterImpl_GetPodSketch(t *testing.T) {
	tests := map[string]struct {
		summary   interface{}
		namespace string
		podName   string
		podUID    string
		want      *sketchapi.PodSketch
		err       error
		wantErr   bool
	}{
		"nil": {
			err:     ErrEmpty,
			wantErr: true,
		},
		"normal": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			podName:   "test-pod-1",
			podUID:    "123456",
			want: &sketchapi.PodSketch{
				PodRef: sketchapi.PodReference{
					Namespace: "test-ns",
					Name:      "test-pod-1",
					UID:       "123456",
				},
				Containers: []*sketchapi.ContainerSketch{
					&sketchapi.ContainerSketch{
						Name: "test-container",
						ID:   "test-contaienr-normal",
						Memory: &sketchapi.ContainerMemorySketch{
							MemorySketch: sketchapi.MemorySketch{
								AvailableBytes: 1024,
							},
						},
					},
				},
			},
		},
		"find-without-uid": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			podName:   "test-pod-1",
			want: &sketchapi.PodSketch{
				PodRef: sketchapi.PodReference{
					Namespace: "test-ns",
					Name:      "test-pod-1",
					UID:       "123456",
				},
				Containers: []*sketchapi.ContainerSketch{
					&sketchapi.ContainerSketch{
						Name: "test-container",
						ID:   "test-contaienr-normal",
						Memory: &sketchapi.ContainerMemorySketch{
							MemorySketch: sketchapi.MemorySketch{
								AvailableBytes: 1024,
							},
						},
					},
				},
			},
		},
		"find-without-podRef-UID": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podUID:  "123456",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-namespace": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podName: "test-pod-1",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-name": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			err:       ErrNotFound,
			wantErr:   true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSnapshotImpl(tt.summary)
			got, err := s.GetPodSketch(tt.namespace, tt.podName, types.UID(tt.podUID))
			if err != nil {
				if !tt.wantErr {
					t.Errorf("snapshoterImpl.GetPodSketch() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.err != nil && err != tt.err {
					t.Errorf("snapshoterImpl.GetPodSketch() error = %v, expect err: %v", err, tt.err)
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("snapshoterImpl.GetPodSketch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_snapshoterImpl_GetContainerSketchByName(t *testing.T) {
	tests := map[string]struct {
		summary       interface{}
		namespace     string
		podName       string
		podUID        string
		containerName string
		want          *sketchapi.ContainerSketch
		err           error
		wantErr       bool
	}{
		"nil": {
			err:     ErrEmpty,
			wantErr: true,
		},
		"normal": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:     "test-ns",
			podName:       "test-pod-1",
			podUID:        "123456",
			containerName: "test-container",
			want: &sketchapi.ContainerSketch{
				Name: "test-container",
				ID:   "test-contaienr-normal",
				Memory: &sketchapi.ContainerMemorySketch{
					MemorySketch: sketchapi.MemorySketch{
						AvailableBytes: 1024,
					},
				},
			},
		},
		"find-without-uid": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:     "test-ns",
			podName:       "test-pod-1",
			containerName: "test-container",
			want: &sketchapi.ContainerSketch{
				Name: "test-container",
				ID:   "test-contaienr-normal",
				Memory: &sketchapi.ContainerMemorySketch{
					MemorySketch: sketchapi.MemorySketch{
						AvailableBytes: 1024,
					},
				},
			},
		},
		"find-without-podRef-UID": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podUID:  "123456",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-namespace": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podName: "test-pod-1",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-pod-name": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			err:       ErrNotFound,
			wantErr:   true,
		},
		"find-without-container-name": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			podName:   "test-pod-1",
			err:       ErrNotFound,
			wantErr:   true,
		},
		"find-with-other-container-name": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:     "test-ns",
			podName:       "test-pod-1",
			containerName: "test-other-container",
			err:           ErrNotFound,
			wantErr:       true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSnapshotImpl(tt.summary)
			got, err := s.GetContainerSketchByName(tt.namespace, tt.podName, types.UID(tt.podUID), tt.containerName)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("snapshoterImpl.GetContainerSketchByName() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.err != nil && err != tt.err {
					t.Errorf("snapshoterImpl.GetContainerSketchByName() error = %v, expect err: %v", err, tt.err)
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("snapshoterImpl.GetContainerSketchByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_snapshoterImpl_GetContainerSketchByID(t *testing.T) {
	tests := map[string]struct {
		summary     interface{}
		namespace   string
		podName     string
		podUID      string
		containerID string
		want        *sketchapi.ContainerSketch
		err         error
		wantErr     bool
	}{
		"nil": {
			err:     ErrEmpty,
			wantErr: true,
		},
		"normal": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:   "test-ns",
			podName:     "test-pod-1",
			podUID:      "123456",
			containerID: "test-contaienr-normal",
			want: &sketchapi.ContainerSketch{
				Name: "test-container",
				ID:   "test-contaienr-normal",
				Memory: &sketchapi.ContainerMemorySketch{
					MemorySketch: sketchapi.MemorySketch{
						AvailableBytes: 1024,
					},
				},
			},
		},
		"find-without-uid": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:   "test-ns",
			podName:     "test-pod-1",
			containerID: "test-contaienr-normal",
			want: &sketchapi.ContainerSketch{
				Name: "test-container",
				ID:   "test-contaienr-normal",
				Memory: &sketchapi.ContainerMemorySketch{
					MemorySketch: sketchapi.MemorySketch{
						AvailableBytes: 1024,
					},
				},
			},
		},
		"find-without-podRef-UID": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podUID:  "123456",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-namespace": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			podName: "test-pod-1",
			err:     ErrNotFound,
			wantErr: true,
		},
		"find-without-pod-name": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			err:       ErrNotFound,
			wantErr:   true,
		},
		"find-without-container-id": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			podName:   "test-pod-1",
			err:       ErrNotFound,
			wantErr:   true,
		},
		"find-with-other-container-id": {
			summary: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Namespace: "test-ns",
							Name:      "test-pod-1",
							UID:       "123456",
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: "test-container",
								ID:   "test-contaienr-normal",
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										AvailableBytes: 1024,
									},
								},
							},
						},
					},
				},
			},
			namespace:   "test-ns",
			podName:     "test-pod-1",
			containerID: "test-contaienr-normal-other",
			err:         ErrNotFound,
			wantErr:     true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSnapshotImpl(tt.summary)
			got, err := s.GetContainerSketchByID(tt.namespace, tt.podName, types.UID(tt.podUID), tt.containerID)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("snapshoterImpl.GetContainerSketchByID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.err != nil && err != tt.err {
					t.Errorf("snapshoterImpl.GetContainerSketchByID() error = %v, expect err: %v", err, tt.err)
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("snapshoterImpl.GetContainerSketchByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
