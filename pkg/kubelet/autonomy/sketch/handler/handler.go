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
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch"
)

// APIRootPath is the definition of the default Sketch API Root Path
const APIRootPath = "/sketch/v1alpha1"

type handler struct {
	sketchProvider sketch.Provider
}

// CreateHandlers creates WebService for sketch
func CreateHandlers(rootPath string, sketchProvider sketch.Provider) *restful.WebService {
	h := &handler{
		sketchProvider: sketchProvider,
	}

	ws := &restful.WebService{}
	ws.Path(rootPath).
		Produces(restful.MIME_JSON)

	endpoints := []struct {
		path    string
		handler restful.RouteFunction
	}{
		// sketch handler for aggregated node, pod and container metrics
		{"/", h.handleSketchSummary},
		{"/node", h.handleSketchNode},
		{"/pod", h.handleSketchPod},
		{"/container", h.handleSketchPodContainer},
	}

	for _, e := range endpoints {
		for _, method := range []string{"GET", "POST"} {
			ws.Route(ws.
				Method(method).
				Path(e.path).
				To(e.handler))
		}
	}

	return ws
}

// handleError serializes an error object into an HTTP response.
// request is provided for logging.
func handleError(response *restful.Response, request string, err error) {
	switch err {
	case sketch.ErrNotFound:
		response.WriteError(http.StatusNotFound, err)
	default:
		msg := fmt.Sprintf("Internal Error: %v", err)
		glog.Errorf("HTTP InternalServerError serving %s: %s", request, msg)
		response.WriteErrorString(http.StatusInternalServerError, msg)
	}
}

func (h *handler) handleSketchSummary(request *restful.Request, response *restful.Response) {
	summary, err := h.sketchProvider.GetSketch().GetSummary()
	if err != nil {
		handleError(response, "/sketch", err)
	} else {
		response.WriteAsJson(summary)
	}
}

// Handles node sketch requests to /sketch/node
func (h *handler) handleSketchNode(request *restful.Request, response *restful.Response) {
	nodeSketch, err := h.sketchProvider.GetSketch().GetNodeSketch()
	if err != nil {
		handleError(response, "/sketch/node", err)
	} else {
		response.WriteAsJson(nodeSketch)
	}
}

// Handles kubernetes pod sketch requests to:
// /sketch/pod?{namespace}=&{podName}=&{uid}=
func (h *handler) handleSketchPod(request *restful.Request, response *restful.Response) {
	uid := request.QueryParameter("uid")
	podName := request.QueryParameter("name")
	namespace := request.QueryParameter("namespace")

	if podName == "" && uid == "" {
		response.WriteErrorString(http.StatusBadRequest, "invalid query parameters")
		return
	}
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	podSketch, err := h.sketchProvider.GetSketch().GetPodSketch(namespace, podName, types.UID(uid))
	if err != nil {
		handleError(response, request.Request.URL.String(), err)
		return
	}
	response.WriteAsJson(podSketch)
}

// Handles kubernetes pod/container sketch requests to:
// /sketch/container?{namespace}=&{podName}=&{uid}=&{containerName}=
func (h *handler) handleSketchPodContainer(request *restful.Request, response *restful.Response) {
	uid := request.QueryParameter("uid")
	podName := request.QueryParameter("podName")
	namespace := request.QueryParameter("namespace")
	containerName := request.QueryParameter("containerName")

	if podName == "" && uid == "" || containerName == "" {
		response.WriteErrorString(http.StatusBadRequest, "invalid query parameters")
		return
	}
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	containerSketch, err := h.sketchProvider.GetSketch().GetContainerSketchByName(
		namespace,
		podName,
		types.UID(uid),
		containerName)
	if err != nil {
		handleError(response, request.Request.URL.String(), err)
		return
	}
	response.WriteAsJson(containerSketch)
}
