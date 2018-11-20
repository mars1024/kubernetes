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

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	sketchtest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/testing"
)

func TestEndpoints_Summary(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.SketchSummary{
		Node: sketchapi.NodeSketch{
			Name: "test-node",
			CPU: &sketchapi.NodeCPUSketch{
				Usage: &sketchapi.SketchData{
					Latest: 1,
				},
			},
		},
	}
	snapshot.On("GetSummary").Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var summary sketchapi.SketchSummary
	err = json.NewDecoder(resp.Body).Decode(&summary)
	assert.NoError(t, err)
	assert.Equal(t, expect, &summary)
}

func TestEndpoints_SummaryWithException(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var nilSummary *sketchapi.SketchSummary
	snapshot.On("GetSummary").Return(nilSummary, sketch.ErrEmpty)

	resp, err := http.Get(server.URL + APIRootPath)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestEndpoints_Node(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.NodeSketch{
		Name: "test-node",
		CPU: &sketchapi.NodeCPUSketch{
			Usage: &sketchapi.SketchData{
				Latest: 1,
			},
		},
	}
	snapshot.On("GetNodeSketch").Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath + "/node")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var nodeSketch sketchapi.NodeSketch
	err = json.NewDecoder(resp.Body).Decode(&nodeSketch)
	assert.NoError(t, err)
	assert.Equal(t, expect, &nodeSketch)
}

func TestEndpoints_NodeWithException(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var nilSketch *sketchapi.NodeSketch
	snapshot.On("GetNodeSketch").Return(nilSketch, sketch.ErrEmpty)

	resp, err := http.Get(server.URL + APIRootPath + "/node")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestEndpoints_Pod(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.PodSketch{
		PodRef: sketchapi.PodReference{
			Name:      "test",
			Namespace: "test-namespaec",
			UID:       "123456",
		},
		Containers: []*sketchapi.ContainerSketch{
			{
				Name: "test-container",
				ID:   "abcdefg",
				CPU: &sketchapi.ContainerCPUSketch{
					UsageInLimit: &sketchapi.SketchData{
						Latest: 1,
					},
					UsageInRequest: &sketchapi.SketchData{
						Latest: 512,
					},
					LoadAverage: &sketchapi.SketchData{
						Latest: 1,
					},
				},
			},
		},
	}
	snapshot.On("GetPodSketch", "test-namespace", "test", types.UID("123456")).Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath + "/pod?namespace=test-namespace&name=test&uid=123456")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var podSketch sketchapi.PodSketch
	err = json.NewDecoder(resp.Body).Decode(&podSketch)
	assert.NoError(t, err)
	assert.Equal(t, expect, &podSketch)
}

func TestEndpoints_PodWithOnlyPodName(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.PodSketch{
		PodRef: sketchapi.PodReference{
			Name:      "test",
			Namespace: metav1.NamespaceDefault,
			UID:       "123456",
		},
		Containers: []*sketchapi.ContainerSketch{
			{
				Name: "test-container",
				ID:   "abcdefg",
				CPU: &sketchapi.ContainerCPUSketch{
					UsageInLimit: &sketchapi.SketchData{
						Latest: 1,
					},
					UsageInRequest: &sketchapi.SketchData{
						Latest: 512,
					},
					LoadAverage: &sketchapi.SketchData{
						Latest: 1,
					},
				},
			},
		},
	}
	snapshot.On("GetPodSketch", metav1.NamespaceDefault, "test", types.UID("")).Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath + "/pod?name=test")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var podSketch sketchapi.PodSketch
	err = json.NewDecoder(resp.Body).Decode(&podSketch)
	assert.NoError(t, err)
	assert.Equal(t, expect, &podSketch)
}

func TestEndpoints_PodWithNotFound(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var nilSketch *sketchapi.PodSketch
	snapshot.On("GetPodSketch", metav1.NamespaceDefault, "test", types.UID("")).Return(nilSketch, sketch.ErrNotFound)

	resp, err := http.Get(server.URL + APIRootPath + "/pod?name=test")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestEndpoints_PodWithNotFound2(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var nilSketch *sketchapi.PodSketch
	snapshot.On("GetPodSketch", "test-123", "test", types.UID("456")).Return(nilSketch, sketch.ErrNotFound)

	resp, err := http.Get(server.URL + APIRootPath + "/pod?namespace=test-123&name=test&uid=456")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestEndpoints_PodWithEmpty(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var nilSketch *sketchapi.PodSketch
	snapshot.On("GetPodSketch", "test-123", "test", types.UID("456")).Return(nilSketch, sketch.ErrEmpty)

	resp, err := http.Get(server.URL + APIRootPath + "/pod?namespace=test-123&name=test&uid=456")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestEndpoints_PodWithInvalidQueryParameters(t *testing.T) {
	provider := &sketchtest.Provider{}

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	resp, err := http.Get(server.URL + APIRootPath + "/pod?namespace=test-123")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoints_Container(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.ContainerSketch{
		Name: "test-container",
		ID:   "abcdefg",
		CPU: &sketchapi.ContainerCPUSketch{
			UsageInLimit: &sketchapi.SketchData{
				Latest: 1,
			},
			UsageInRequest: &sketchapi.SketchData{
				Latest: 512,
			},
			LoadAverage: &sketchapi.SketchData{
				Latest: 1,
			},
		},
	}
	snapshot.On("GetContainerSketchByName", metav1.NamespaceDefault, "test", types.UID(""), "test-container").
		Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath + "/container?podName=test&containerName=test-container")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var containerSketch sketchapi.ContainerSketch
	err = json.NewDecoder(resp.Body).Decode(&containerSketch)
	assert.NoError(t, err)
	assert.Equal(t, expect, &containerSketch)
}
func TestEndpoints_ContainerWithNamespace(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	expect := &sketchapi.ContainerSketch{
		Name: "test-container",
		ID:   "abcdefg",
		CPU: &sketchapi.ContainerCPUSketch{
			UsageInLimit: &sketchapi.SketchData{
				Latest: 1,
			},
			UsageInRequest: &sketchapi.SketchData{
				Latest: 512,
			},
			LoadAverage: &sketchapi.SketchData{
				Latest: 1,
			},
		},
	}
	snapshot.On("GetContainerSketchByName", "test-namespace", "test", types.UID("123"), "test-container").
		Return(expect, error(nil))

	resp, err := http.Get(server.URL + APIRootPath + "/container?namespace=test-namespace&podName=test&uid=123&containerName=test-container")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var containerSketch sketchapi.ContainerSketch
	err = json.NewDecoder(resp.Body).Decode(&containerSketch)
	assert.NoError(t, err)
	assert.Equal(t, expect, &containerSketch)
}

func TestEndpoints_ContainerWithEmpty(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var expect *sketchapi.ContainerSketch
	snapshot.On("GetContainerSketchByName", "test-namespace", "test", types.UID("123"), "test-container").
		Return(expect, sketch.ErrEmpty)

	resp, err := http.Get(server.URL + APIRootPath + "/container?namespace=test-namespace&podName=test&uid=123&containerName=test-container")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestEndpoints_ContainerWithoutContainerName(t *testing.T) {
	provider := &sketchtest.Provider{}
	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	resp, err := http.Get(server.URL + APIRootPath + "/container?namespace=test-namespace&podName=test&uid=123")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoints_ContainerWithoutPodNameAndUID(t *testing.T) {
	provider := &sketchtest.Provider{}
	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	resp, err := http.Get(server.URL + APIRootPath + "/container?namespace=test-namespace&containerName=xxxx")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoints_ContainerWithNotFound(t *testing.T) {
	snapshot := &sketchtest.Snapshoter{}
	provider := &sketchtest.Provider{}
	provider.On("GetSketch").Return(sketch.Snapshoter(snapshot))

	restContainer := restful.NewContainer()
	server := httptest.NewServer(restContainer)
	defer server.Close()

	restContainer.Add(CreateHandlers(APIRootPath, provider))

	var expect *sketchapi.ContainerSketch
	snapshot.On("GetContainerSketchByName", "test-namespace", "test", types.UID("123"), "test-container").
		Return(expect, sketch.ErrNotFound)

	resp, err := http.Get(server.URL + APIRootPath + "/container?namespace=test-namespace&podName=test&uid=123&containerName=test-container")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
