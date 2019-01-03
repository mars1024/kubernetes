/*
Copyright 2017 The Kubernetes Authors.

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

package apiserver

import (
	"net/http"
	"strings"
	"sync"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/apiextensions-apiserver/pkg/util"
)

type versionDiscoveryHandler struct {
	// TODO, writing is infrequent, optimize this
	discoveryLock sync.RWMutex
	discovery     map[util.TenantGroupVersion]*discovery.APIVersionHandler

	delegate http.Handler
}

func (r *versionDiscoveryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pathParts := splitPath(req.URL.Path)
	// only match /apis/<group>/<version>
	if len(pathParts) != 3 || pathParts[0] != "apis" {
		r.delegate.ServeHTTP(w, req)
		return
	}
	user, _ := request.UserFrom(req.Context())
	tenant, err := multitenancyutil.TransformTenantInfoFromUser(user)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("group version discovery failed: %v", err))
	}
	tgv := util.TenantGroupVersion{
		multitenancyutil.GetHashFromTenant(tenant),
		schema.GroupVersion{Group: pathParts[1], Version: pathParts[2]},
	}
	discovery, ok := r.getDiscovery(tgv)
	if !ok {
		r.delegate.ServeHTTP(w, req)
		return
	}

	discovery.ServeHTTP(w, req)
}

func (r *versionDiscoveryHandler) getDiscovery(tgv util.TenantGroupVersion) (*discovery.APIVersionHandler, bool) {
	r.discoveryLock.RLock()
	defer r.discoveryLock.RUnlock()

	ret, ok := r.discovery[tgv]
	return ret, ok
}

func (r *versionDiscoveryHandler) setDiscovery(tgv util.TenantGroupVersion, discovery *discovery.APIVersionHandler) {
	r.discoveryLock.Lock()
	defer r.discoveryLock.Unlock()

	r.discovery[tgv] = discovery
}

func (r *versionDiscoveryHandler) unsetDiscovery(tgv util.TenantGroupVersion) {
	r.discoveryLock.Lock()
	defer r.discoveryLock.Unlock()

	delete(r.discovery, tgv)
}

type groupDiscoveryHandler struct {
	// TODO, writing is infrequent, optimize this
	discoveryLock sync.RWMutex
	discovery     map[util.TenantGroup]*discovery.APIGroupHandler

	delegate http.Handler
}

func (r *groupDiscoveryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pathParts := splitPath(req.URL.Path)
	// only match /apis/<group>
	if len(pathParts) != 2 || pathParts[0] != "apis" {
		r.delegate.ServeHTTP(w, req)
		return
	}
	user, _ := request.UserFrom(req.Context())
	tenant, err := multitenancyutil.TransformTenantInfoFromUser(user)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("group version discovery failed: %v", err))
	}
	discovery, ok := r.getDiscovery(util.TenantGroup{
		TenantHash: multitenancyutil.GetHashFromTenant(tenant),
		Group:      pathParts[1],
	})
	if !ok {
		r.delegate.ServeHTTP(w, req)
		return
	}

	discovery.ServeHTTP(w, req)
}

func (r *groupDiscoveryHandler) getDiscovery(tg util.TenantGroup) (*discovery.APIGroupHandler, bool) {
	r.discoveryLock.RLock()
	defer r.discoveryLock.RUnlock()

	ret, ok := r.discovery[tg]
	return ret, ok
}

func (r *groupDiscoveryHandler) setDiscovery(tg util.TenantGroup, discovery *discovery.APIGroupHandler) {
	r.discoveryLock.Lock()
	defer r.discoveryLock.Unlock()

	r.discovery[tg] = discovery
}

func (r *groupDiscoveryHandler) unsetDiscovery(tg util.TenantGroup) {
	r.discoveryLock.Lock()
	defer r.discoveryLock.Unlock()

	delete(r.discovery, tg)
}

// splitPath returns the segments for a URL path.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
