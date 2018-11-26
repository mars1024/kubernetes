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

package adaptivescale

import (
	"time"

	"github.com/golang/glog"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

const (
	// autopilotServiceKey is the key of autopilot serivce status from node annotation.
	autopilotServiceKey = api.AnnotationAutopilot
	// autopilotServiceExecPeriodKey is the key of autopilot serivce execution period from node annotation.
	autopilotServiceExecPeriodKey = api.AnnotationAutopilot + "/" + "executionIntervalSeconds"
	// DefaultAutopilotExecPeriod is default autopilot service execution period.
	// if autopilot serive is started while autopilot execution period is empty,
	// autopilot execution period will be set to defaultAutopilotExecPeriod.
	DefaultAutopilotExecPeriod = 10
	// MinAutopilotExecPeriod is the minimum autopilot serivce execution period.
	minAutopilotExecPeriod = 1
)

// AdaptivescaleService is the Adaptivescale Service.
func (adaptivescaleController *ResourceAdjustController) AdaptivescaleService(annotations map[string]string) {

	autopilotServiceAnnotations, err := getAutopilotServiceAnnotations(annotations)
	if err != nil || len(autopilotServiceAnnotations) == 0 {
		return
	}

	if adaptivescaleController.GetNodeInfo() == nil {
		return
	}

	switch autopilotServiceAnnotations[autopilotServiceKey] {
	case "true":
		executionIntervalSeconds := autopilotServiceAnnotations[autopilotServiceExecPeriodKey].(time.Duration)
		startAutopilot(adaptivescaleController, executionIntervalSeconds)
	case "false":
		stopAutopilot(adaptivescaleController)
	}
}

// startAutopilot starts autopilot service.
func startAutopilot(adaptivescaleController *ResourceAdjustController, executionIntervalSeconds time.Duration) {
	adaptivescaleController.Start(executionIntervalSeconds)
}

// stopAutopilot stops autopilot service.
func stopAutopilot(adaptivescaleController *ResourceAdjustController) {
	adaptivescaleController.Stop()
}

func getAutopilotServiceAnnotations(annotations map[string]string) (map[string]interface{}, error) {
	autopilot, ok := annotations[autopilotServiceKey]
	if !ok {
		return map[string]interface{}{autopilotServiceKey: "false"}, nil
	}

	switch autopilot {
	case "true":
		return getStartAutopilotParameters(annotations)
	case "false":
		return map[string]interface{}{autopilotServiceKey: "false"}, nil
	default:
		glog.Errorf("can not get correct value of annotations")
		return map[string]interface{}{autopilotServiceKey: "false"}, nil
	}
}

func getStartAutopilotParameters(annotations map[string]string) (map[string]interface{}, error) {
	var startAutopilotParameters = make(map[string]interface{})
	startAutopilotParameters[autopilotServiceKey] = "true"
	executePeriod, ok := annotations[autopilotServiceExecPeriodKey]
	if !ok {
		startAutopilotParameters[autopilotServiceExecPeriodKey] = DefaultAutopilotExecPeriod * time.Second
	}

	startAutopilotParameters[autopilotServiceExecPeriodKey] = transferExecPeriodtoInt(executePeriod)
	return startAutopilotParameters, nil
}

func transferExecPeriodtoInt(execPeriod string) time.Duration {
	execInterval, err := time.ParseDuration(execPeriod)
	if err != nil {
		return DefaultAutopilotExecPeriod * time.Second
	}
	if execInterval < minAutopilotExecPeriod*time.Second {
		return minAutopilotExecPeriod * time.Second
	}
	return execInterval
}
