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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"
	"unicode"

	"gopkg.in/yaml.v2"
)

type metric struct {
	Name   string   `yaml:"name"`
	Help   string   `yaml:"help"`
	Expr   string   `yaml:"expr"`
	Labels []string `yaml:"labels"`
}

type metricGroup struct {
	Name     string   `yaml:"name"`
	Help     string   `yaml:"help"`
	Type     string   `yaml:"type"`
	Scraper  string   `yaml:"scraper"`
	Interval string   `yaml:"interval"`
	Labels   []string `yaml:"labels"`
	Metrics  []metric `yaml:"metrics"`
}

type metricRuleFile struct {
	PackageName string        `yaml:"packageName"`
	Groups      []metricGroup `yaml:"groups"`
}

var metricFileTempate = template.Must(template.New("metric-template").Funcs(template.FuncMap{
	"title": strings.Title,
	"parseDuration": func(v string) (string, error) {
		duration, err := time.ParseDuration(v)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d * time.Second", duration/time.Second), nil
	},
	"normalizeMetricName": func(s string) string {
		c := 0
		var r []rune
		for i, v := range s {
			if i == 0 && unicode.IsLower(v) {
				r = append(r, unicode.ToUpper(v))
			} else if v == '_' {
				c = i + 1
				continue
			} else if c == i && c != 0 {
				r = append(r, unicode.ToUpper(v))
			} else {
				r = append(r, v)
			}
		}
		replacer := strings.NewReplacer("_", "", "Cpu", "CPU")
		return replacer.Replace(string(r))
	},
}).Parse(`
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

package {{.PackageName}}

import (
	"time"
)

// AllMetricGroups is MetricGroup set
var AllMetricGroups = []MetricGroup{ {{range .Groups}} {{if len .Metrics}}
	{{.Name | normalizeMetricName}}Group,{{end}} {{end}}
}

{{range .Groups}}
{{if len .Metrics }}
// {{.Name | normalizeMetricName}}Group {{.Help}}
var {{.Name | normalizeMetricName}}Group = MetricGroup{
	Name: "{{.Name}}",
	Help: "{{.Help}}",
	Type: {{if eq .Type "container"}}ContainerMetricType{{else if eq .Type "pod"}}PodMetricType{{else}}NodeMetricType{{end}},
	Scraper: "{{.Scraper}}",{{if ne .Interval ""}}
	Interval: {{.Interval | parseDuration}},{{end}}{{if len .Labels}}
	Labels: []string{ {{range .Labels}}
		"{{.}}",{{end}}
	},{{end}}
	Metrics: []Metric{ {{range .Metrics}}
		{{.Name | normalizeMetricName}}, {{end}}
	},	
}
{{end}}
{{end}}

{{range .Groups}}
{{range .Metrics}}
// {{.Name | normalizeMetricName}} {{.Help}}
var {{.Name | normalizeMetricName}} = Metric{
	Name: "{{.Name}}",{{if ne .Help ""}}
	Help: "{{.Help}}",{{end}}{{if ne .Expr ""}}
	Expr: {{.Expr|printf "%q"}},{{else}}
	Expr: "{{.Name}}",{{end}}{{if len .Labels}}
	Labels: []string{ {{range .Labels}}
		"{{.}}",{{end}}
	},
	{{end}}
}
{{end}}
{{end}}
`))

func main() {
	var metricFile string
	flag.StringVar(&metricFile, "f", "", "metric definition yaml file")
	flag.Parse()

	if metricFile == "" {
		flag.Usage()
		return
	}

	data, err := ioutil.ReadFile(metricFile)
	if err != nil {
		panic(err)
	}

	var rules metricRuleFile
	err = yaml.Unmarshal(data, &rules)
	if err != nil {
		panic(err)
	}

	err = metricFileTempate.Execute(os.Stdout, rules)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
