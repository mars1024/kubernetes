package antitamper

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	genericadmissioninitializer "k8s.io/apiserver/pkg/admission/initializer"
	authenticationuser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
)

const PluginName = "AntiTamper"

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, Factory)
}

type AntiTamper struct {
	*admission.Handler
	client          kubernetes.Interface
	namespaceLister v1.NamespaceLister
}

func Factory(config io.Reader) (admission.Interface, error) {
	return NewAntiTamperAdmissionController(), nil
}

func NewAntiTamperAdmissionController() *AntiTamper {
	return &AntiTamper{
		Handler: admission.NewHandler(admission.Create, admission.Update, admission.Delete),
	}
}

var _ = genericadmissioninitializer.WantsExternalKubeClientSet(&AntiTamper{})

func (ac *AntiTamper) SetExternalKubeClientSet(client kubernetes.Interface) {
	ac.client = client
}

func (ac *AntiTamper) ValidateInitialization() error {
	if ac.client == nil {
		return fmt.Errorf("[anti-tamper ac] %s requires a client", PluginName)
	}

	if ac.namespaceLister == nil {
		return fmt.Errorf("[anti-tamper ac] %s requires a namespaceLister", PluginName)
	}

	glog.V(4).Infof("[anti-tamper ac] Initialization OK")
	return nil
}

var _ = genericadmissioninitializer.WantsExternalKubeInformerFactory(&AntiTamper{})

func (ac *AntiTamper) SetExternalKubeInformerFactory(f informers.SharedInformerFactory) {
	ac.namespaceLister = f.Core().V1().Namespaces().Lister()
	ac.SetReadyFunc(f.Core().V1().Namespaces().Informer().HasSynced)
}

var _ admission.ValidationInterface = &AntiTamper{}

func (ac *AntiTamper) Validate(attributes admission.Attributes) (err error) {

	if attributes == nil {
		return nil
	}

	// Ignore all calls to subresources
	if len(attributes.GetSubresource()) != 0 {
		return nil
	}
	
	// prevent panicking
	defer func() {
		if r := recover(); r != nil {
			err = admission.NewForbidden(attributes, errors.New(fmt.Sprintf("fatal error: %v", r)))
		}
	}()

	resourceName := getResourceNameFromAttributes(attributes)

	verboseLogIfNecessary("Validate() started", resourceName)

	if multitenancyutil.IsMultiTenancyWiseAdmin(attributes.GetUserInfo().GetName()) {
		verboseLogIfNecessary("isAdmin, skipping", resourceName)
		return nil
	}

	tenant, err := multitenancyutil.TransformTenantInfoFromUser(attributes.GetUserInfo())
	if err != nil {
		glog.Warning("fail to extract tenant info from user: %v", attributes.GetUserInfo())
		return admission.NewForbidden(attributes, errors.New("no tenant info from user"))
	}
	if multitenancyutil.IsMultiTenancyWiseTenant(tenant) {
		verboseLogIfNecessary("isAdmin, skipping", resourceName)
		return nil
	}


 	user := attributes.GetUserInfo()
	verboseLogIfNecessary(fmt.Sprintf("%s is not Admin", user.GetName()), resourceName)

	ac = ac.ShallowCopyWithTenant(tenant).(*AntiTamper)

	context := &validateContext{attributes: attributes, errorMessage: "", namespaceLister: &ac.namespaceLister}

	if tryingToUpdateImmutableLabelsOrAnnotations(context) {
		verboseLogIfNecessary("tryingToUpdateImmutableLabelsOrAnnotations hit", resourceName)
		return admission.NewForbidden(attributes, errors.New(context.errorMessage))
	}

	if tryingToCreateUpdateOrDeleteProtectedResource(context) {
		verboseLogIfNecessary("tryingToCreateUpdateOrDeleteProtectedResource hit", resourceName)
		return admission.NewForbidden(attributes, errors.New(context.errorMessage))
	}

	verboseLogIfNecessary("validation passed", resourceName)
	return nil
}

func tryingToUpdateImmutableLabelsOrAnnotations(context *validateContext) bool {
	name := getResourceNameFromAttributes(context.attributes)

	if context.attributes.GetOperation() != admission.Update {
		verboseLogIfNecessary("tryingToUpdateImmutableLabelsOrAnnotations not Update, skipping", name)
		return false
	}

	newMeta, err := meta.Accessor(context.attributes.GetObject())
	if err != nil {
		verboseLogIfNecessary("newMeta not available", name)
		return false
	}

	oldMeta, err := meta.Accessor(context.attributes.GetOldObject())
	if err != nil {
		verboseLogIfNecessary("oldMeta not available", name)
		return false
	}

	verboseLogIfNecessary(fmt.Sprintf("newMeta=%v, oldMeta=%v", newMeta, oldMeta), name)

	changedAnnotations := getChangedKeys(newMeta.GetAnnotations(), oldMeta.GetAnnotations())
	verboseLogIfNecessary(fmt.Sprintf("changedAnnotations=%v", changedAnnotations), name)

	changedImmutableAnnotations := doFilter(changedAnnotations, immutableAnnotations)
	verboseLogIfNecessary(fmt.Sprintf("changedImmutableAnnotations=%v", changedImmutableAnnotations), name)

	if len(changedImmutableAnnotations) > 0 {
		context.errorMessage = "updating immutable annotations is not allowed (" + strings.Join(changedImmutableAnnotations, ",") + ")"
		return true
	}

	changedLabels := getChangedKeys(newMeta.GetLabels(), oldMeta.GetLabels())
	verboseLogIfNecessary(fmt.Sprintf("changedLabels=%v", changedLabels), name)

	changedImmutableLabels := doFilter(changedLabels, immutableLabels)
	verboseLogIfNecessary(fmt.Sprintf("changedImmutableLabels=%v", changedImmutableLabels), name)

	if len(changedImmutableLabels) > 0 {
		context.errorMessage = "updating immutable labels is not allowed (" + strings.Join(changedImmutableLabels, ",") + ")"
		return true
	}

	verboseLogIfNecessary("tryingToUpdateImmutableLabelsOrAnnotations false", name)

	return false
}

func tryingToCreateUpdateOrDeleteProtectedResource(context *validateContext) bool {
	gvk := context.attributes.GetKind()

	group := gvk.Group
	version := gvk.Version
	kind := gvk.Kind
	name := getResourceNameFromAttributes(context.attributes)
	namespace := context.attributes.GetNamespace()

	verboseLogIfNecessary(fmt.Sprintf("tryingToCreateUpdateOrDeleteProtectedResource %s %s %s %s %s", group, version, kind, name, namespace), name)

	if isResourceSpecifiedInProtectedResourceList(group, version, kind, name, namespace, context) {
		verboseLogIfNecessary("isResourceSpecifiedInProtectedResourceList hit", name)
		context.errorMessage = "resource is protected as it is a critical system resource"
		return true
	}
	if isResourceFromSystemReservedNamespace(group, version, kind, name, namespace, context) {
		verboseLogIfNecessary("isResourceFromSystemReservedNamespace hit", name)
		context.errorMessage = "resource is protected as it is from a system reserved namespace"
		return true
	}
	if isResourceItselfASystemReservedNamespace(group, version, kind, name, namespace, context) {
		verboseLogIfNecessary("isResourceItselfASystemReservedNamespace hit", name)
		context.errorMessage = "resource is protected as it is a system reserved namespace"
		return true
	}

	verboseLogIfNecessary("tryingToCreateUpdateOrDeleteProtectedResource false", name)

	return false
}

func isResourceFromSystemReservedNamespace(group string, version string, kind string, name string, namespace string, context *validateContext) bool {
	return isSystemReservedNamespace(namespace, context)
}

func isResourceSpecifiedInProtectedResourceList(group string, version string, kind string, name string, namespace string, context *validateContext) bool {
	for _, resourceIdentifier := range protectedResources {
		if matchesResourceIdentifier(group, version, kind, name, namespace, resourceIdentifier) {
			verboseLogIfNecessary("isResourceSpecifiedInProtectedResourceList hit", name)
			return true
		}
	}
	return false
}

func isResourceItselfASystemReservedNamespace(group string, version string, kind string, name string, namespace string, context *validateContext) bool {
	if !isTypeNamespace(group, version, kind) {
		name := getResourceNameFromAttributes(context.attributes)
		verboseLogIfNecessary(fmt.Sprintf("isResourceItselfASystemReservedNamespace false as it is not a Namespace (%s, %s, %s)", group, version, kind), name)
		return false
	}

	return isSystemReservedNamespace(name, context)
}

func isAdmin(user authenticationuser.Info) bool {
	return IsMultiTenancyWiseAdmin(user.GetName())
}

func isSystemReservedNamespace(name string, context *validateContext) bool {

	resourceName := getResourceNameFromAttributes(context.attributes)

	if len(name) == 0 {
		verboseLogIfNecessary("isSystemReservedNamespace false as resource is not namespaced", resourceName)
		return false
	}

	for _, a := range cafeSystemReservedNamespaceNames {
		if a == name {
			verboseLogIfNecessary("isSystemReservedNamespace true as it is one of cafeSystemReservedNamespaceNames", resourceName)
			return true
		}
	}

	return namespaceHasAnnotationCafeSystemReservedNamespace(name, context)
}

func namespaceHasAnnotationCafeSystemReservedNamespace(name string, context *validateContext) bool {
	var err error
	var ok bool
	var metadata metav1.Object

	attributeKind := context.attributes.GetKind()
	group := attributeKind.Group
	version := attributeKind.Version
	kind := attributeKind.Kind

	resourceName := getResourceNameFromAttributes(context.attributes)

	if context.attributes.GetOperation() == admission.Create && isTypeNamespace(group, version, kind) {
		verboseLogIfNecessary("namespaceHasAnnotationCafeSystemReservedNamespace creating new namespace", resourceName)
		metadata, err = meta.Accessor(context.attributes.GetObject())
	} else {
		verboseLogIfNecessary(fmt.Sprintf("namespaceHasAnnotationCafeSystemReservedNamespace examining namespace %s", name), resourceName)
		namespace, err := (*context.namespaceLister).Get(name)
		if err != nil || namespace == nil {
			verboseLogIfNecessary(fmt.Sprintf("namespaceHasAnnotationCafeSystemReservedNamespace failed to examine namespace %s: %v", name, err), resourceName)
			return false
		}
		metadata, err = meta.Accessor(namespace)
	}

	if err != nil {
		verboseLogIfNecessary(fmt.Sprintf("namespaceHasAnnotationCafeSystemReservedNamespace failed to access namespace metadata of %s: %v", resourceName, err), resourceName)
		return false
	}

	value, ok := metadata.GetAnnotations()[AnnotationCafeSystemReservedNamespace]
	if !ok || value != "true" {
		verboseLogIfNecessary(fmt.Sprintf("namespaceHasAnnotationCafeSystemReservedNamespace namespace metadata of %s has no annotation", resourceName), resourceName)
		return false
	}

	verboseLogIfNecessary(fmt.Sprintf("namespaceHasAnnotationCafeSystemReservedNamespace namespace metadata of %s has annotation", resourceName), resourceName)
	return true
}

func (ac *AntiTamper) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *ac
	copied.namespaceLister = ac.namespaceLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(v1.NamespaceLister)
	return &copied
}

func getResourceNameFromAttributes(attributes admission.Attributes) string {
	attributesName := attributes.GetName()
	if len(attributesName) > 0 {
		return attributesName
	}

	obj := attributes.GetObject()
	if obj == nil {
		return ""
	}
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		glog.V(4).Infof("[anti-tamper ac] getResourceNameFromAttributes metaAccessor erred %v", err)
		return ""
	}

	name := metaAccessor.GetName()
	if len(name) > 0 {
		return name
	}

	generateName := metaAccessor.GetGenerateName()
	if len(generateName) > 0 {
		return generateName + "?????"
	}

	return ""
}
