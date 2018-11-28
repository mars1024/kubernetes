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

// Package armory contains an admission controller that checks and modifies every new Pod
package armory

import (
	"io"

	"fmt"
	"github.com/golang/glog"
	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"sort"
	"strings"
)

// PluginName indicates name of admission plugin.
const PluginName = "Armory"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewArmory(), nil
	})
}

// Armory is an implementation of admission.Interface.
// It validates labels of pods which must meet sigma policy.
type Armory struct {
	*admission.Handler
}

var _ admission.MutationInterface = &Armory{}
var _ admission.ValidationInterface = &Armory{}

var (
	mustContainLabelsMap = map[string]struct{}{
		sigmaapi.LabelAppName: {},
		sigmaapi.LabelSite:    {},
	}
	cannotUpdateLabelsMap = map[string]struct{}{
		sigmaapi.LabelPodSn:   {},
		sigmaapi.LabelAppName: {},
		sigmaapi.LabelSite:    {},
		// TODO update labels in sigma api library
		"sigma.alibaba-inc.com/app-unit":  {},
		"sigma.alibaba-inc.com/app-stage": {},
	}
)

func (*Armory) admitCreate(attributes admission.Attributes) error {
	pod, _ := attributes.GetObject().(*api.Pod)
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	if _, ok := pod.Labels[sigmaapi.LabelPodSn]; !ok {
		pod.Labels[sigmaapi.LabelPodSn] = string(uuid.NewUUID())
	}
	return nil
}

func (*Armory) admitUpdate(attributes admission.Attributes) error {
	return nil
}

func (*Armory) validateCreate(attributes admission.Attributes) error {
	pod, _ := attributes.GetObject().(*api.Pod)
	var miss []string
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	for key, _ := range mustContainLabelsMap {
		if _, ok := pod.Labels[key]; !ok {
			miss = append(miss, key)
		}
	}
	if len(miss) > 0 {
		sort.Strings(miss)
		return admission.NewForbidden(attributes, fmt.Errorf("labels %s must be set", strings.Join(miss, ", ")))
	}
	return nil
}

func (*Armory) validateUpdate(attributes admission.Attributes) error {
	pod, _ := attributes.GetObject().(*api.Pod)

	oldPod, ok := attributes.GetOldObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	var extra []string
	glog.V(10).Infof("Armory validateUpdate pod: %#v", pod)
	for key, _ := range pod.Labels {
		// 先判断是否不可变更
		if _, ok := cannotUpdateLabelsMap[key]; ok {
			// 不可变更的标签存在，且被修改了
			if val, ok := oldPod.Labels[key]; ok && val != pod.Labels[key] {
				extra = append(extra, key)
			} else if !ok { // 不可变更的标签不存在，被添加了
				extra = append(extra, key)
			}
		}
	}
	if len(extra) > 0 {
		return admission.NewForbidden(attributes, fmt.Errorf("labels %s can not update", strings.Join(extra, ", ")))
	}
	return nil
}

// Admit makes an admission decision based on the request attributes
func (a *Armory) Admit(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	_, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	op := attributes.GetOperation()
	switch op {
	case admission.Create:
		return a.admitCreate(attributes)
	case admission.Update:
		return a.admitUpdate(attributes)
	}
	return apierrors.NewBadRequest("Armory Admission only handles Create or Update event")
}

// Validate makes sure that all containers are set to correct armory labels
func (a *Armory) Validate(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	_, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	op := attributes.GetOperation()
	switch op {
	case admission.Create:
		return a.validateCreate(attributes)
	case admission.Update:
		return a.validateUpdate(attributes)
	}

	return apierrors.NewBadRequest("Armory Admission only handles Create or Update event")
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}

	return false
}

// NewArmory creates a new armory admission control handler
func NewArmory() *Armory {
	return &Armory{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}
