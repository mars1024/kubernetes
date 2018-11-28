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

// Package NetworkStatus contains an admission controller that checks and modifies every new Pod
package networkstatus

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/glog"
	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

// PluginName indicates name of admission plugin.
const PluginName = "NetworkStatus"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewNetworkStatus(), nil
	})
}

// NetworkStatus is an implementation of admission.Interface.
// It validates the annotation sigmaapi.AnnotationPodNetworkStats which can not update once created .
type NetworkStatus struct {
	*admission.Handler
}

//var _ admission.MutationInterface = &NetworkStatus{}
var _ admission.ValidationInterface = &NetworkStatus{}

/*
func (*NetworkStatus) admitUpdate(attributes admission.Attributes) error {
	return nil
}
*/

func (*NetworkStatus) validateUpdate(attributes admission.Attributes) error {
	pod, _ := attributes.GetObject().(*api.Pod)

	oldPod, ok := attributes.GetOldObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	glog.V(10).Infof("NetworkStatus validateUpdate pod: %#v, old pod: %#v", pod, oldPod)
	if pod.Annotations == nil {
		return nil
	}

	oldStatusBytes, oldStatusExist := oldPod.Annotations[sigmaapi.AnnotationPodNetworkStats]
	var oldStatus sigmaapi.NetworkStatus
	if oldStatusExist {
		err := json.Unmarshal([]byte(oldStatusBytes), &oldStatus)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not update due to json unmarshal error `%s`", sigmaapi.AnnotationPodNetworkStats, err))
		}

		err = validateNetworkStatus(&oldStatus)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not update due to %s", sigmaapi.AnnotationPodNetworkStats, err))
		}
	}

	statusBytes, statusExist := pod.Annotations[sigmaapi.AnnotationPodNetworkStats]
	var status sigmaapi.NetworkStatus
	if statusExist {
		err := json.Unmarshal([]byte(statusBytes), &status)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not update due to json unmarshal error `%s`", sigmaapi.AnnotationPodNetworkStats, err))
		}

		err = validateNetworkStatus(&status)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not update due to %s", sigmaapi.AnnotationPodNetworkStats, err))
		}
	}

	// !oldStatusExist && !statusExist 不需要处理
	// oldStatusExist && !statusExist 释放逻辑，不需要处理
	// !oldStatusExist && statusExist 直接修改
	if oldStatusExist && statusExist {
		if err := validateNetworkStatusUpdate(&oldStatus, &status); err != nil {
			return admission.NewForbidden(attributes, fmt.Errorf("annotation %s can not update: %v", sigmaapi.AnnotationPodNetworkStats, err))
		}
	}

	return nil
}

func validateNetworkStatusUpdate(old, new *sigmaapi.NetworkStatus) error {
	munged := *new
	munged.SandboxId = old.SandboxId
	if !reflect.DeepEqual(old, &munged) {
		return fmt.Errorf("old=%#v, new=%#v", old, new)
	}
	return nil
}

func validateNetworkStatus(status *sigmaapi.NetworkStatus) error {
	var errMessages []string

	if len(status.VlanID) > 0 {
		// VlanID - (0, 4096)
		if vlan, err := strconv.Atoi(status.VlanID); err != nil {
			errMessages = append(errMessages, "invalid field: vlan "+err.Error())
		} else {
			if r := validation.IsInRange(vlan, 1, 4095); r != nil {
				errMessages = append(errMessages, "invalid field: vlan "+strings.Join(r, ""))
			}
		}
	}

	// NetworkPrefixLength - [0, 32]
	if r := validation.IsInRange(int(status.NetworkPrefixLength), 0, 32); r != nil {
		errMessages = append(errMessages, "invalid field: networkPrefixLen "+strings.Join(r, ""))
	}

	// Gateway - 必须为IPv4地址
	if r := validation.IsValidIP(status.Gateway); r != nil {
		errMessages = append(errMessages, "invalid field: gateway "+strings.Join(r, ""))
	}

	// MACAddress - 必须为合法MAC地址
	if _, err := net.ParseMAC(status.MACAddress); len(status.MACAddress) > 0 && err != nil {
		errMessages = append(errMessages, "invalid field: macAddress "+err.Error())
	}

	if len(errMessages) > 0 {
		return fmt.Errorf(strings.Join(errMessages, ", "))
	}
	return nil
}

/*
// Admit makes an admission decision based on the request attributes
func (n *NetworkStatus) Admit(attributes admission.Attributes) (err error) {
	// Ignore all calls to subresources or resources other than pods.
	if shouldIgnore(attributes) {
		return nil
	}

	_, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	return n.admitUpdate(attributes)
}
*/

// Validate makes sure that all containers are set to correct NetworkStatus labels
func (n *NetworkStatus) Validate(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	_, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	op := attributes.GetOperation()
	if op == admission.Update {
		return n.validateUpdate(attributes)
	}
	return apierrors.NewBadRequest("NetworkStatus Admission only handles Update event")
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}

	return false
}

// NewNetworkStatus creates a new NetworkStatus admission control handler
func NewNetworkStatus() *NetworkStatus {
	return &NetworkStatus{
		Handler: admission.NewHandler(admission.Update),
	}
}
