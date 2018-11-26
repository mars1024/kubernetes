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
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/handler"
)

func main() {
	sketch.AddFlagSet(pflag.CommandLine)
	addGlogFlags(pflag.CommandLine)
	pflag.Parse()

	options := sketch.NewOptions()
	provider, err := sketch.New(options, nil)
	if err != nil {
		glog.Fatalf("failed to sketch.New, err: %s", err.Error())
	}
	if err = provider.Start(); err != nil {
		glog.Fatalf("failed to provider.Start, err: %s", err.Error())
	}
	defer provider.Stop()
	glog.Info("provider start succeed")

	restContainer := restful.NewContainer()
	restContainer.Add(handler.CreateHandlers("/", provider))
	restContainer.ServeMux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    ":9091",
		Handler: restContainer,
	}

	errChan := make(chan error)
	go func() {
		errChan <- server.ListenAndServe()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Kill, os.Interrupt)

	for {
		select {
		case err := <-errChan:
			glog.Errorln("http server err:", err)
			return

		case sig := <-sigChan:
			glog.Infoln("got signal:", sig)
			return

		case <-time.After(5 * time.Second):
			summary, err := provider.GetSketch().GetSummary()
			if err != nil {
				glog.Errorf("failed to query summary, err: %s", err.Error())
				continue
			}
			printSummary(summary)
		}
	}
}

func printSummary(summary *sketchapi.SketchSummary) {
	var buff bytes.Buffer
	fmt.Fprintln(&buff, "node.name:", summary.Node.Name)
	printNodeCPUSketch(&buff, summary.Node.CPU)
	printNodeMemorySketch(&buff, summary.Node.Memory)
	fmt.Println(buff.String())
}

func printNodeCPUSketch(w io.Writer, s *sketchapi.NodeCPUSketch) {
	if s == nil {
		return
	}
	fmt.Fprintln(w, "node.cpu.time:", s.Time)
	printSketchData(w, "node.cpu.usage", s.Usage)
}

func printNodeMemorySketch(w io.Writer, s *sketchapi.NodeMemorySketch) {
	if s == nil {
		return
	}
	fmt.Fprintln(w, "node.memory.time:", s.Time)
	fmt.Fprintln(w, "node.memory.available:", s.AvailableBytes)
	fmt.Fprintln(w, "node.memory.usage:", s.UsageBytes)
	fmt.Fprintln(w, "node.memory.workingset:", s.WorkingSetBytes)
}

func printSketchData(w io.Writer, prefix string, s *sketchapi.SketchData) {
	if s == nil {
		return
	}
	fmt.Fprintln(w, prefix+".latest:", s.Latest)
	printSketchCumulation(w, prefix+".min1", &s.Min1)
	printSketchCumulation(w, prefix+".min5", &s.Min5)
	printSketchCumulation(w, prefix+".min15", &s.Min15)
}

func printSketchCumulation(w io.Writer, prefix string, s *sketchapi.SketchCumulation) {
	fmt.Fprintln(w, prefix+".max:", s.Max)
	fmt.Fprintln(w, prefix+".min:", s.Min)
	fmt.Fprintln(w, prefix+".avg:", s.Avg)
	fmt.Fprintln(w, prefix+".p99:", s.P99)
	fmt.Fprintln(w, prefix+".predict:", s.Predict)
}

// register adds a flag to local that targets the Value associated with the Flag named globalName in global
func register(global *flag.FlagSet, local *pflag.FlagSet, globalName string) {
	if f := global.Lookup(globalName); f != nil {
		pflagFlag := pflag.PFlagFromGoFlag(f)
		pflagFlag.Name = normalize(pflagFlag.Name)
		local.AddFlag(pflagFlag)
	} else {
		panic(fmt.Sprintf("failed to find flag in global flagset (flag): %s", globalName))
	}
}

func normalize(s string) string {
	return strings.Replace(s, "_", "-", -1)
}

func addGlogFlags(fs *pflag.FlagSet) {
	// lookup flags in global flag set and re-register the values with our flagset
	global := flag.CommandLine
	local := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	register(global, local, "logtostderr")
	register(global, local, "alsologtostderr")
	register(global, local, "v")
	register(global, local, "stderrthreshold")
	register(global, local, "vmodule")
	register(global, local, "log_backtrace_at")
	register(global, local, "log_dir")

	fs.AddFlagSet(local)
}
