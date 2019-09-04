package inclusterkube

import (
	"flag"
	"fmt"
	"io"
	"net/url"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	PluginName = "AlipayInClusterKubernetes"

	// 存放集群访问信息的 ConfigMap，从这里获取 k8s service host:port
	clusterInfoConfigConfigMapKey = "kube-public/cluster-info"

	// in-cluster kubernetes 环境变量名
	kubernetesInClusterServiceHost = "KUBERNETES_SERVICE_HOST"
	kubernetesInClusterServicePort = "KUBERNETES_SERVICE_PORT"
)

var (
	clusterDomain = flag.String("in-cluster-domain", "", "cluster domain used for search domain")
)

type AlipayInClusterKubernetes struct {
	*admission.Handler

	configMapLister corelisters.ConfigMapLister
	clusterDomain   string
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayInClusterKubernetes(), nil
	})
}

var (
	_ admission.MutationInterface                             = &AlipayInClusterKubernetes{}
	_ admission.InitializationValidator                       = &AlipayInClusterKubernetes{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &AlipayInClusterKubernetes{}
)

func (i *AlipayInClusterKubernetes) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	i.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	i.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (i *AlipayInClusterKubernetes) ValidateInitialization() error {
	if i.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func NewAlipayInClusterKubernetes() *AlipayInClusterKubernetes {
	return &AlipayInClusterKubernetes{Handler: admission.NewHandler(admission.Create), clusterDomain: *clusterDomain}
}

func (i *AlipayInClusterKubernetes) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}
	if !i.WaitForReady(a.GetContext()) {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	pod, ok := a.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("unexpect resource"))
	}

	if err = i.HandleInClusterKubernetesServiceEnv(pod); err != nil {
		return errors.NewInternalError(err)
	}

	i.HandleInClusterDNSSearchDomain(pod, a.GetNamespace())
	return nil
}

func (i *AlipayInClusterKubernetes) HandleInClusterKubernetesServiceEnv(pod *core.Pod) error {
	namespace, name, _ := cache.SplitMetaNamespaceKey(clusterInfoConfigConfigMapKey)

	cm, err := i.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(cm.Data) == 0 {
		return nil
	}
	kubeConfig, err := clientcmd.Load([]byte(cm.Data["kubeconfig"]))
	if err != nil {
		glog.Errorf("unmarshal kubeconfig in ConfigMap %v error: %v", clusterInfoConfigConfigMapKey, err)
		return err
	}

	server, port, err := getServerPortFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}

	for i := range pod.Spec.Containers {
		var found bool

		for _, env := range pod.Spec.Containers[i].Env {
			// FIXME (cao.yin)
			// 这里只是简单处理了Env，实际可能存在EnvFrom的使用，但校验过程比较复杂。
			// 考虑到Kubernetes InCluster配置的注入几乎不可能有Pod指定，因此不需要过度处理。
			if env.Name == kubernetesInClusterServiceHost || env.Name == kubernetesInClusterServicePort {
				found = true
				break
			}
		}

		if !found {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env,
				core.EnvVar{Name: kubernetesInClusterServiceHost, Value: server},
				core.EnvVar{Name: kubernetesInClusterServicePort, Value: port})
		}
	}

	return nil
}

func getServerPortFromKubeConfig(kubeConfig *api.Config) (string, string, error) {
	if n := len(kubeConfig.Clusters); n != 1 {
		return "", "", fmt.Errorf("kubeConfig.Clusters contain %d clusters", n)
	}
	if _, exists := kubeConfig.Clusters[""]; !exists {
		return "", "", fmt.Errorf("kubeConfig.Clusters must contain default cluster")
	}

	u, err := url.Parse(kubeConfig.Clusters[""].Server)
	if err != nil {
		return "", "", fmt.Errorf("url.Parse error: %v", err)
	}
	return u.Hostname(), u.Port(), nil
}

func (i *AlipayInClusterKubernetes) HandleInClusterDNSSearchDomain(pod *core.Pod, ns string) {
	if len(i.clusterDomain) == 0 || len(ns) == 0 {
		return
	}

	switch pod.Spec.DNSPolicy {
	case core.DNSClusterFirst:
		if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.HostNetwork {
			return
		}
	case core.DNSClusterFirstWithHostNet:
	default:
		return
	}

	if pod.Spec.DNSConfig == nil {
		pod.Spec.DNSConfig = &core.PodDNSConfig{}
	}
	glog.V(4).Infof("inject dns search domain for pod %s/%s", ns, pod.Name)
	pod.Spec.DNSConfig.Searches = append(pod.Spec.DNSConfig.Searches,
		fmt.Sprintf("%s.svc.%s", ns, i.clusterDomain),
		fmt.Sprintf("svc.%s", i.clusterDomain),
		i.clusterDomain)
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != core.Resource("pods") {
		return true
	}

	return false
}
