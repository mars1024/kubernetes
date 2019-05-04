package privatecloud

import (
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	coreapi "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	namespaceslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

const (
	// PluginName indicates name of admission plugin.
	// This plugin will try to inject the tenant info from the namespace
	// Note: currently this plugin is not applied to cluster-scoped resources
	PluginName = "PrivateCloud"

	tenantLabel    = "cafe.sofastack.io/tenant"
	clusterLabel   = "cafe.sofastack.io/cluster"
	workspaceLabel = "cafe.sofastack.io/workspace"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// privateCloudPlugin is an implementation of admission.Interface.
type privateCloudPlugin struct {
	*admission.Handler
	lister namespaceslisters.NamespaceLister
}

var _ admission.MutationInterface = &privateCloudPlugin{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&privateCloudPlugin{})

// NewPlugin creates a new privateCloud admission plugin.
func NewPlugin() *privateCloudPlugin {
	return &privateCloudPlugin{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (plugin *privateCloudPlugin) ValidateInitialization() error {
	if plugin.lister == nil {
		return fmt.Errorf("%s requires a lister", PluginName)
	}
	return nil
}

func (plugin *privateCloudPlugin) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	namespaceInformer := f.Core().InternalVersion().Namespaces()
	plugin.lister = namespaceInformer.Lister()
	plugin.SetReadyFunc(namespaceInformer.Informer().HasSynced)
}

// Admit injects the tenant info from the namespace to the runtime.Object.
func (plugin *privateCloudPlugin) Admit(a admission.Attributes) error {
	// Ignore all non-namespaced resources.
	// Ignore all calls to subresources or namespaces resources.
	if a.GetNamespace() == coreapi.NamespaceNone || len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() == coreapi.Resource("namespaces") {
		return nil
	}
	// Ignore all operations other than CREATE, UPDATE.
	if a.GetOperation() != admission.Create && a.GetOperation() != admission.Update {
		return nil
	}

	if a.GetOperation() == admission.Update {
		glog.V(5).Infof("Checking tenant labels...")
		err := checkTenantLabels(a.GetObject(), a.GetOldObject())
		if err != nil {
			return err
		}
	}

	namespace, err := plugin.lister.Get(a.GetNamespace())
	if err != nil {
		return fmt.Errorf("getting namespace %q failed: %v", a.GetNamespace(), err)
	}

	glog.V(5).Infof("Trying to get tenant info from namespace %q...", namespace.Name)
	tenant, err := getTenantInfoFromLabelsOrAnnotations(*namespace)
	if err != nil {
		glog.Errorf("Failed to get tenant info from namespace %q with error: %v", namespace.Name, err)
		return err
	}

	glog.V(5).Infof("Trying to inject tenant info %+v to %s/%s/%s labels", tenant, a.GetNamespace(), a.GetResource(), a.GetName())
	err = injectTenantLabels(a.GetObject(), transferTenantInfoToLabels(tenant))
	if err != nil {
		glog.Errorf("Failed to inject tenant labels to %s/%s/%s with error: %v", a.GetNamespace(), a.GetResource(), a.GetName(), err)
		return err
	}

	glog.V(5).Infof("Trying to inject tenant info %+v to %s/%s/%s annotations", tenant, a.GetNamespace(), a.GetResource(), a.GetName())
	err = injectTenantAnnotations(a.GetObject(), multitenancyutil.TransformTenantInfoToAnnotations(tenant))
	if err != nil {
		glog.Errorf("Failed to inject tenant annotations to %s/%s/%s with error: %v", a.GetNamespace(), a.GetResource(), a.GetName(), err)
		return err
	}

	return nil
}

func getTenantInfoFromLabelsOrAnnotations(namespace coreapi.Namespace) (multitenancy.TenantInfo, error) {
	missingFields := []string{}
	tenantID, ok := namespace.Labels[tenantLabel]
	if !ok {
		tenantID, ok = namespace.Annotations[multitenancy.MultiTenancyAnnotationKeyTenantID]
		if !ok {
			missingFields = append(missingFields, "tenantID")
		}
	}
	workspaceID, ok := namespace.Labels[workspaceLabel]
	if !ok {
		workspaceID, ok = namespace.Annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID]
		if !ok {
			missingFields = append(missingFields, "workspaceID")
		}
	}
	clusterID, ok := namespace.Labels[clusterLabel]
	if !ok {
		clusterID, ok = namespace.Annotations[multitenancy.MultiTenancyAnnotationKeyClusterID]
		if !ok {
			missingFields = append(missingFields, "clusterID")
		}
	}
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing or malformed tenant info %v in namespace %q labels and annotations", missingFields, namespace.Name)
	}
	return multitenancy.NewTenantInfo(tenantID, workspaceID, clusterID), nil
}

func transferTenantInfoToLabels(tenant multitenancy.TenantInfo) map[string]string {
	labels := make(map[string]string)
	if len(tenant.GetTenantID()) > 0 {
		labels[tenantLabel] = tenant.GetTenantID()
	}
	if len(tenant.GetWorkspaceID()) > 0 {
		labels[workspaceLabel] = tenant.GetWorkspaceID()
	}
	if len(tenant.GetClusterID()) > 0 {
		labels[clusterLabel] = tenant.GetClusterID()
	}
	return labels

}

func injectTenantLabels(obj runtime.Object, tenantLabels map[string]string) error {
	accessor := meta.NewAccessor()
	currentLabels, err := accessor.Labels(obj)
	if err != nil {
		return err
	}

	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}
	for key, val := range tenantLabels {
		if _, ok := currentLabels[key]; ok {
			glog.V(5).Infof("Skip updating existing label %q", key)
			continue
		}
		currentLabels[key] = val
	}
	return accessor.SetLabels(obj, currentLabels)
}

func injectTenantAnnotations(obj runtime.Object, tenantAnnotations map[string]string) error {
	accessor := meta.NewAccessor()
	currentAnns, err := accessor.Annotations(obj)
	if err != nil {
		return err
	}

	if currentAnns == nil {
		currentAnns = make(map[string]string)
	}
	for key, val := range tenantAnnotations {
		if _, ok := currentAnns[key]; ok {
			glog.V(5).Infof("Skip updating existing annotation %q", key)
			continue
		}
		currentAnns[key] = val
	}
	return accessor.SetAnnotations(obj, currentAnns)
}

func checkTenantLabels(newObj runtime.Object, oldObj runtime.Object) error {
	accessor := meta.NewAccessor()
	newLabels, err := accessor.Labels(newObj)
	if err != nil {
		return err
	}
	oldLabels, err := accessor.Labels(oldObj)
	if err != nil {
		return err
	}

	deletedField := []string{}
	changedField := []string{}
	tenantLabels := []string{tenantLabel, workspaceLabel, clusterLabel}
	for _, label := range tenantLabels {
		changed, deleted := inspectChangedField(oldLabels, newLabels, label)
		if changed {
			changedField = append(changedField, label)
		}
		if deleted {
			deletedField = append(deletedField, label)
		}
	}

	var errMsg string
	if len(deletedField) != 0 {
		errMsg += fmt.Sprintf("not allowed to delete tenant field: %s;", strings.Join(deletedField, ","))
	}
	if len(changedField) != 0 {
		errMsg += fmt.Sprintf("not allowed to modify tenant field: %s;", strings.Join(changedField, ","))
	}

	if len(errMsg) > 0 {
		return errors.NewBadRequest(errMsg)
	}

	return nil
}

func inspectChangedField(oldLabels, newLabels map[string]string, fieldName string) (changed, deleted bool) {
	oldVal, oldExists := oldLabels[fieldName]
	newVal, newExists := newLabels[fieldName]

	if oldExists {
		if newExists {
			if oldVal != newVal {
				changed = true
			}
		} else {
			deleted = true
		}
	} else {
		if newExists {
			// should be auto injected
			// we assume this as changed
			changed = true
		}
	}

	return changed, deleted
}
