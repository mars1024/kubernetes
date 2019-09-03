// +build multitenancy

/*
Copyright 2015 The Kubernetes Authors.

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
	"io"
	"path/filepath"
	"strings"

	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"

	"k8s.io/code-generator/cmd/client-gen/generators/util"
)

// genClientForType produces a file for each top-level type.
type genClientForType struct {
	generator.DefaultGen
	outputPackage    string
	clientsetPackage string
	group            string
	version          string
	groupGoName      string
	typeToMatch      *types.Type
	imports          namer.ImportTracker
}

var _ generator.Generator = &genClientForType{}

// Filter ignores all but one type because we're making a single file per type.
func (g *genClientForType) Filter(c *generator.Context, t *types.Type) bool { return t == g.typeToMatch }

func (g *genClientForType) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *genClientForType) Imports(c *generator.Context) (imports []string) {
	imports = g.imports.ImportLines()
	imports = append(imports, "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy")
	imports = append(imports, `gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta`)
	imports = append(imports, "k8s.io/client-go/rest")
	return imports
}

// Ideally, we'd like genStatus to return true if there is a subresource path
// registered for "status" in the API server, but we do not have that
// information, so genStatus returns true if the type has a status field.
func genStatus(t *types.Type) bool {
	// Default to true if we have a Status member
	hasStatus := false
	for _, m := range t.Members {
		if m.Name == "Status" {
			hasStatus = true
			break
		}
	}
	return hasStatus && !util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...)).NoStatus
}

// GenerateType makes the body of a file implementing the individual typed client for type t.
func (g *genClientForType) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "$", "$")
	pkg := filepath.Base(t.Name.Package)
	tags, err := util.ParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
	if err != nil {
		return err
	}
	type extendedInterfaceMethod struct {
		template string
		args     map[string]interface{}
	}
	for _, e := range tags.Extensions {
		inputType := *t
		resultType := *t
		// TODO: Extract this to some helper method as this code is copied into
		// 2 other places.
		if len(e.InputTypeOverride) > 0 {
			if name, pkg := e.Input(); len(pkg) > 0 {
				newType := c.Universe.Type(types.Name{Package: pkg, Name: name})
				inputType = *newType
			} else {
				inputType.Name.Name = e.InputTypeOverride
			}
		}
		if len(e.ResultTypeOverride) > 0 {
			if name, pkg := e.Result(); len(pkg) > 0 {
				newType := c.Universe.Type(types.Name{Package: pkg, Name: name})
				resultType = *newType
			} else {
				resultType.Name.Name = e.ResultTypeOverride
			}
		}
	}
	m := map[string]interface{}{
		"type":                 t,
		"inputType":            t,
		"resultType":           t,
		"package":              pkg,
		"Package":              namer.IC(pkg),
		"namespaced":           !tags.NonNamespaced,
		"Group":                namer.IC(g.group),
		"subresource":          false,
		"subresourcePath":      "",
		"GroupGoName":          g.groupGoName,
		"Version":              namer.IC(g.version),
		"DeleteOptions":        c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "DeleteOptions"}),
		"ListOptions":          c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "ListOptions"}),
		"GetOptions":           c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/apis/meta/v1", Name: "GetOptions"}),
		"PatchType":            c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/types", Name: "PatchType"}),
		"watchInterface":       c.Universe.Type(types.Name{Package: "k8s.io/apimachinery/pkg/watch", Name: "Interface"}),
		"RESTClientInterface":  c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Interface"}),
		"schemeParameterCodec": c.Universe.Variable(types.Name{Package: filepath.Join(g.clientsetPackage, "scheme"), Name: "ParameterCodec"}),
	}

	sw.Do(shallowCopyTemplate, m)

	return sw.Error()
}

// adjustTemplate adjust the origin verb template using the expansion name.
// TODO: Make the verbs in templates parametrized so the strings.Replace() is
// not needed.
func adjustTemplate(name, verbType, template string) string {
	return strings.Replace(template, " "+strings.Title(verbType), " "+name, -1)
}

var shallowCopyTemplate = `
func (c *$.type|privatePlural$) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(rest.Interface)
	return &copied
}
`
