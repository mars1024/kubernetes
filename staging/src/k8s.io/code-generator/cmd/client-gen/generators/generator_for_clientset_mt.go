// +build multitenancy

/*
Copyright 2016 The Kubernetes Authors.

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

package generators

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	clientgentypes "k8s.io/code-generator/cmd/client-gen/types"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

// genClientset generates a package for a clientset.
type genClientset struct {
	generator.DefaultGen
	groups             []clientgentypes.GroupVersions
	groupGoNames       map[clientgentypes.GroupVersion]string
	clientsetPackage   string
	outputPackage      string
	imports            namer.ImportTracker
	clientsetGenerated bool
}

var _ generator.Generator = &genClientset{}

func (g *genClientset) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

// We only want to call GenerateType() once.
func (g *genClientset) Filter(c *generator.Context, t *types.Type) bool {
	ret := !g.clientsetGenerated
	g.clientsetGenerated = true
	return ret
}

func (g *genClientset) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	imports = append(imports, "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy")
	imports = append(imports, `gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta`)
	imports = append(imports, "k8s.io/client-go/rest")
	imports = append(imports, "k8s.io/client-go/discovery")
	for _, group := range g.groups {
		for _, version := range group.Versions {
			typedClientPath := filepath.Join(g.clientsetPackage, "typed", group.PackageName, version.NonEmpty())
			groupAlias := strings.ToLower(g.groupGoNames[clientgentypes.GroupVersion{Group: group.Group, Version: version.Version}])
			imports = append(imports, strings.ToLower(fmt.Sprintf("%s%s \"%s\"", groupAlias, version.NonEmpty(), typedClientPath)))
		}
	}
	return
}

func (g *genClientset) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	// TODO: We actually don't need any type information to generate the clientset,
	// perhaps we can adapt the go2ild framework to this kind of usage.
	sw := generator.NewSnippetWriter(w, c, "$", "$")

	allGroups := clientgentypes.ToGroupVersionInfo(g.groups, g.groupGoNames)
	m := map[string]interface{}{
		"allGroups":                            allGroups,
		"Config":                               c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Config"}),
		"DefaultKubernetesUserAgent":           c.Universe.Function(types.Name{Package: "k8s.io/client-go/rest", Name: "DefaultKubernetesUserAgent"}),
		"RESTClientInterface":                  c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Interface"}),
		"DiscoveryInterface":                   c.Universe.Type(types.Name{Package: "k8s.io/client-go/discovery", Name: "DiscoveryInterface"}),
		"DiscoveryClient":                      c.Universe.Type(types.Name{Package: "k8s.io/client-go/discovery", Name: "DiscoveryClient"}),
		"NewDiscoveryClientForConfig":          c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClientForConfig"}),
		"NewDiscoveryClientForConfigOrDie":     c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClientForConfigOrDie"}),
		"NewDiscoveryClient":                   c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClient"}),
		"flowcontrolNewTokenBucketRateLimiter": c.Universe.Function(types.Name{Package: "k8s.io/client-go/util/flowcontrol", Name: "NewTokenBucketRateLimiter"}),
	}
	sw.Do(clientsetInterface, m)

	return sw.Error()
}

var clientsetInterface = `
func (c *Clientset) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.DiscoveryClient = copied.DiscoveryClient.ShallowCopyWithTenant(tenant).(*discovery.DiscoveryClient)

	$range .allGroups$ 
		copied.$.LowerCaseGroupGoName$$.Version$ = copied.$.LowerCaseGroupGoName$$.Version$.ShallowCopyWithTenant(tenant).(*$.PackageAlias$.$.GroupGoName$$.Version$Client)	
	$end$
	return &copied
}
`
