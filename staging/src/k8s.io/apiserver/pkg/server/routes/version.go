/*
Copyright 2014 The Kubernetes Authors.

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

package routes

import (
	"net/http"

	"github.com/emicklei/go-restful"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"runtime"
	"fmt"
)

// Version provides a webservice with version information.
type Version struct {
	Version *version.Info
}

// Install registers the APIServer's `/version` handler.
func (v Version) Install(c *restful.Container) {
	if v.Version == nil {
		return
	}

	// Set up a service to return the git code version.
	versionWS := new(restful.WebService)
	versionWS.Path("/version")
	versionWS.Doc("git code version from which this is built")
	versionWS.Route(
		versionWS.GET("/").To(v.handleVersion).
			Doc("get the code version").
			Operation("getCodeVersion").
			Produces(restful.MIME_JSON).
			Consumes(restful.MIME_JSON).
			Writes(version.Info{}))

	c.Add(versionWS)
}

// handleVersion writes the server's version information.
func (v Version) handleVersion(req *restful.Request, resp *restful.Response) {
	versionInfo := version.Info{
		Major:        "1",
		Minor:        "12+",
		GitVersion:   "v1.12.5-aks.1.9",
		GitCommit:    "f932d2457aea933d640c81007760fc31afd2b775",
		GitTreeState: "clean",
		BuildDate:    "2019-06-12T21:00:00Z",
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
	responsewriters.WriteRawJSON(http.StatusOK, versionInfo, resp.ResponseWriter)
}
