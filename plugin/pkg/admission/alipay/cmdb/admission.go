package cmdb

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

const PluginName = "AlipayCMDB"

type AlipayCMDB struct {
	*admission.Handler
}

var (
	_ admission.ValidationInterface = &AlipayCMDB{}
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayCMDB(), nil
	})
}

func NewAlipayCMDB() *AlipayCMDB {
	return &AlipayCMDB{Handler: admission.NewHandler(admission.Create, admission.Update)}
}

var (
	mustRequiredCMDBLabels = []string{
		sigmak8sapi.LabelAppName,
		sigmak8sapi.LabelPodSn,
		sigmak8sapi.LabelDeployUnit,
		sigmak8sapi.LabelSite,
		sigmak8sapi.LabelInstanceGroup,
	}
)

func (c *AlipayCMDB) Validate(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	var (
		pod  = a.GetObject().(*core.Pod)
		errs []error
	)

	switch a.GetOperation() {
	case admission.Create:
		for _, k := range mustRequiredCMDBLabels {
			if _, exists := pod.Labels[k]; !exists {
				errs = append(errs, fmt.Errorf("label %s is required", k))
			}
		}
	case admission.Update:
		old := a.GetOldObject().(*core.Pod)
		for _, k := range mustRequiredCMDBLabels {
			if pod.Labels[k] != old.Labels[k] {
				errs = append(errs, fmt.Errorf("label %s is immutable", k))
			}
		}
	}

	if len(errs) > 0 {
		return admission.NewForbidden(a, errors.NewAggregate(errs))
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
