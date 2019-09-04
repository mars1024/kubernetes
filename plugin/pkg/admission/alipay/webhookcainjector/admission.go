package webhookcainjector

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/apis/admissionregistration"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	PluginName = "WebhookCAInjector"

	// 存放集群访问信息的 ConfigMap，从这里获取 k8s service host:port
	clusterInfoConfigConfigMapKey = "kube-public/cluster-info"

	// in-cluster kubernetes 环境变量名
	kubernetesInClusterServiceHost = "KUBERNETES_SERVICE_HOST"
	kubernetesInClusterServicePort = "KUBERNETES_SERVICE_PORT"
)

type WebhookCAInjector struct {
	*admission.Handler

	configMapLister corelisters.ConfigMapLister
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewWebhookCAInjector(), nil
	})
}

var (
	_ admission.MutationInterface                             = &WebhookCAInjector{}
	_ admission.InitializationValidator                       = &WebhookCAInjector{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &WebhookCAInjector{}
)

func (i *WebhookCAInjector) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	i.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	i.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (i *WebhookCAInjector) ValidateInitialization() error {
	if i.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func NewWebhookCAInjector() *WebhookCAInjector {
	return &WebhookCAInjector{Handler: admission.NewHandler(admission.Create, admission.Update)}
}

func (i *WebhookCAInjector) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}
	if !i.WaitForReady(a.GetContext()) {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	switch a.GetObject().(type) {
	case *admissionregistration.MutatingWebhookConfiguration:
		cfg := a.GetObject().(*admissionregistration.MutatingWebhookConfiguration)
		return i.injectCA(cfg.Webhooks)
	case *admissionregistration.ValidatingWebhookConfiguration:
		cfg := a.GetObject().(*admissionregistration.ValidatingWebhookConfiguration)
		return i.injectCA(cfg.Webhooks)
	default:
		return fmt.Errorf("object is not MutatingWebhookConfiguration nor ValidatingWebhookConfiguration")
	}
}

func (i *WebhookCAInjector) injectCA(whs []admissionregistration.Webhook) error {
	ca, err := i.getCAFromKubeConfig()
	if nil != err {
		return err
	}

	if 0 == len(ca) {
		glog.Info("can not get CA info from cm kube-public/cluster-info")
		return nil
	}

	for i := range whs {
		if 0 == len(whs[i].ClientConfig.CABundle) {
			glog.Infof("Webhook has no CA, inject ca")
			whs[i].ClientConfig.CABundle = ca
		}
	}

	return nil
}

func (i *WebhookCAInjector) getCAFromKubeConfig() ([]byte, error) {
	namespace, name, _ := cache.SplitMetaNamespaceKey(clusterInfoConfigConfigMapKey)

	cm, err := i.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(cm.Data) == 0 {
		return nil, nil
	}
	kubeConfig, err := clientcmd.Load([]byte(cm.Data["kubeconfig"]))
	if err != nil {
		glog.Errorf("unmarshal kubeconfig in ConfigMap %v error: %v", clusterInfoConfigConfigMapKey, err)
		return nil, err
	}

	if n := len(kubeConfig.Clusters); n != 1 {
		return nil, fmt.Errorf("kubeConfig.Clusters contain %d clusters", n)
	}
	if _, exists := kubeConfig.Clusters[""]; !exists {
		return nil, fmt.Errorf("kubeConfig.Clusters must contain default cluster")
	}

	ca := kubeConfig.Clusters[""].CertificateAuthorityData

	if 0 == len(ca) {
		return nil, fmt.Errorf("kubeConfig.Clusters[\"\"].CertificateAuthorityData must contain CA Data")
	}

	return ca, nil
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than validatingwebhookconfigurations or mutatingwebhookconfigurations.
	if len(attributes.GetSubresource()) != 0 {
		return true
	}

	if attributes.GetResource().GroupResource() != admissionregistration.Resource("validatingwebhookconfigurations") &&
		attributes.GetResource().GroupResource() != admissionregistration.Resource("mutatingwebhookconfigurations") {
		return true
	}

	return false
}
