package podpreset

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/util/slice"
)

const PluginName = "AlipayZAppInfo"

type AlipayZAppInfo struct {
	*admission.Handler
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayZAppInfo(), nil
	})
}

var (
	_ admission.ValidationInterface = &AlipayZAppInfo{}
	_ admission.MutationInterface   = &AlipayZAppInfo{}
)

func NewAlipayZAppInfo() *AlipayZAppInfo {
	return &AlipayZAppInfo{Handler: admission.NewHandler(admission.Create, admission.Update)}
}

func (z *AlipayZAppInfo) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}
	if a.GetOperation() != admission.Create {
		return nil
	}

	pod := a.GetObject().(*core.Pod)

	zappinfo, err := getPodZAppInfo(pod)
	if err != nil && err != errPodZAppInfoNotFound {
		return err
	}

	if err = z.setPodZAppInfo(pod, zappinfo); err != nil {
		return err
	}

	return nil
}

func (z *AlipayZAppInfo) Validate(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	pod := a.GetObject().(*core.Pod)

	switch a.GetOperation() {
	case admission.Create:
		if err = z.ValidateZAppInfoAnnotations(pod); err != nil {
			return admission.NewForbidden(a, fmt.Errorf("ValidateZAppInfoAnnotations: %v", err))
		}
	case admission.Update:
		old := a.GetOldObject().(*core.Pod)
		if err = z.ValidateImmutableZAppInfoAnnotations(old, pod); err != nil {
			return admission.NewForbidden(a, fmt.Errorf("ValidateImmutableZAppInfoAnnotations: %v", err))
		}
	}

	return nil
}

var (
	choicesOfServerType = []string{
		string(alipaysigmak8sapi.ZappinfoServerTypeDockerVM),
		string(alipaysigmak8sapi.ZappinfoServerTypeDocker),
	}
	defaultServerType = alipaysigmak8sapi.ZappinfoServerTypeDocker

	errPodZAppInfoNotFound = fmt.Errorf("annotation %s required", alipaysigmak8sapi.AnnotationZappinfo)
)

func getPodZAppInfo(pod *core.Pod) (*alipaysigmak8sapi.PodZappinfo, error) {
	data, exists := pod.Annotations[alipaysigmak8sapi.AnnotationZappinfo]
	if !exists {
		return nil, errPodZAppInfoNotFound
	}

	var zappinfo alipaysigmak8sapi.PodZappinfo
	if err := json.Unmarshal([]byte(data), &zappinfo); err != nil {
		return nil, fmt.Errorf("unmarshal ZAppInfo error: %v", err)
	}

	return &zappinfo, nil
}

func (z *AlipayZAppInfo) setPodZAppInfo(pod *core.Pod, info *alipaysigmak8sapi.PodZappinfo) error {
	if info == nil {
		info = &alipaysigmak8sapi.PodZappinfo{}
	}

	if info.Status != nil && info.Status.Registered {
		return nil
	}

	if info.Spec == nil {
		info.Spec = &alipaysigmak8sapi.PodZappinfoSpec{}
	}

	if info.Spec.AppName == "" {
		info.Spec.AppName = pod.Labels[sigmak8sapi.LabelAppName]
	}
	if info.Spec.Zone == "" {
		info.Spec.Zone = pod.Labels[alipaysigmak8sapi.LabelZone]
	}

	if info.Spec.ServerType == "" {
		info.Spec.ServerType = pod.Labels[sigmak8sapi.LabelPodContainerModel]
		if !slice.ContainsString(choicesOfServerType, info.Spec.ServerType, nil) {
			info.Spec.ServerType = string(defaultServerType)
		}
	}

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 1)
	}
	pod.Annotations[alipaysigmak8sapi.AnnotationZappinfo] = string(data)
	return nil
}

func (z *AlipayZAppInfo) ValidateZAppInfoAnnotations(pod *core.Pod) error {
	zappinfo, err := getPodZAppInfo(pod)
	if err != nil {
		return err
	}
	if zappinfo.Status != nil && zappinfo.Status.Registered {
		return nil
	}

	if zappinfo.Spec == nil {
		return fmt.Errorf("zappinfo.spec is required")
	}

	var errs []error
	if len(zappinfo.Spec.AppName) == 0 {
		errs = append(errs, fmt.Errorf("zappinfo.spec.appName is required"))
	}
	if len(zappinfo.Spec.Zone) == 0 {
		errs = append(errs, fmt.Errorf("zappinfo.spec.zone is required"))
	}

	if len(zappinfo.Spec.ServerType) == 0 {
		errs = append(errs, fmt.Errorf("zappinfo.spec.serverType is required"))
	} else if !slice.ContainsString(choicesOfServerType, zappinfo.Spec.ServerType, nil) {
		errs = append(errs, fmt.Errorf("zappinfo.spec.serverType must be one if %v", choicesOfServerType))
	}

	return errors.NewAggregate(errs)
}

func (z *AlipayZAppInfo) ValidateImmutableZAppInfoAnnotations(old, new *core.Pod) error {
	newZAppInfo, err := getPodZAppInfo(new)
	if err != nil {
		return err
	}

	oldZAppInfo, err := getPodZAppInfo(old)
	if err != nil {
		return err
	}

	key, _ := cache.MetaNamespaceKeyFunc(new)
	if !reflect.DeepEqual(newZAppInfo.Spec, oldZAppInfo.Spec) {
		return fmt.Errorf("pod %s zappinfo.spec is immutable", key)
	}

	return nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != core.Resource("pods") {
		return true
	}

	_, ok := a.GetObject().(*core.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() == admission.Update {
		if _, ok := a.GetOldObject().(*core.Pod); !ok {
			glog.Errorf("expected pod but got %s", a.GetOldObject().GetObjectKind().GroupVersionKind().Kind)
			return true
		}
	}

	return false
}
