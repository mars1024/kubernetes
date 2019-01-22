package alipodlifecyclehook

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/credentialprovider"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	PluginName = "AliPodLifeTimeHook"
	ConfigName = "sigma-alipodlifecyclehook-config"

	AnnotationDisableLifeCycleHook = "pod.beta1.sigma.ali/disable-lifecycle-hook"
	LabelSecretRegistryUsage       = "ali-registry-user-account"
)

var (
	secretRegistryUserAccountSelector, _ = labels.NewRequirement("usage", selection.In, []string{LabelSecretRegistryUsage})
)

// aliPodLifeCycleHook is an implementation of admission.Interface.
// It adds pod lifecycle hooks for ali common containers.
type aliPodLifeCycleHook struct {
	*admission.Handler
	client          internalclientset.Interface
	secretLister    settingslisters.SecretLister
	configMapLister settingslisters.ConfigMapLister
}

var _ admission.MutationInterface = &aliPodLifeCycleHook{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&aliPodLifeCycleHook{})
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&aliPodLifeCycleHook{})

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// NewPlugin creates a new aliPodLifeCycleHook plugin.
func NewPlugin() *aliPodLifeCycleHook {
	return &aliPodLifeCycleHook{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (plugin *aliPodLifeCycleHook) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	return nil
}

func (c *aliPodLifeCycleHook) SetInternalKubeClientSet(client internalclientset.Interface) {
	c.client = client
}

func (a *aliPodLifeCycleHook) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	secretInformer := f.Core().InternalVersion().Secrets()
	configMapInformer := f.Core().InternalVersion().ConfigMaps()
	a.secretLister = secretInformer.Lister()
	a.configMapLister = configMapInformer.Lister()
	a.SetReadyFunc(func() bool { return secretInformer.Informer().HasSynced() && configMapInformer.Informer().HasSynced() })
}

type configMap map[string]string

func (c *aliPodLifeCycleHook) getConfigMap() (configMap, error) {
	cm, err := c.configMapLister.ConfigMaps("kube-system").Get(ConfigName)
	if err != nil {
		return nil, err
	}
	return cm.Data, nil
}

func (c configMap) getConfig(key, defaultValue string) string {
	if c == nil {
		return defaultValue
	}
	if d, ok := c[key]; ok {
		return d
	}
	return defaultValue
}

// Admit injects a pod with the specific fields for each pod preset it matches.
func (c *aliPodLifeCycleHook) Admit(a admission.Attributes) error {
	// Ignore all calls to subresources or resources other than pods.
	if len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() != api.Resource("pods") {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return errors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted by aliPodLifeCycleHook")
	}
	glog.V(3).Infof("aliPodLifeCycleHook start to admit %s/%s, operation: %v, pod: %v", pod.Namespace, pod.Name, a.GetOperation(), dumpJson(&pod.ObjectMeta))

	if a.GetOperation() != admission.Create && a.GetOperation() != admission.Update {
		return nil
	}

	if pod.Labels == nil || pod.DeletionTimestamp != nil {
		return nil
	}

	cm, err := c.getConfigMap()
	if err != nil {
		glog.Errorf("aliPodLifeCycleHook get config failed: %v", err)
	}

	isDockerVMMode := pod.Labels[sigmak8sapi.LabelPodContainerModel] == "dockervm"
	disableLifeCycle := pod.Annotations[AnnotationDisableLifeCycleHook] == "true"

	var aoneImage string
	newContainers := make([]api.Container, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, cm.getConfig("aone-image-name-contains", "docker.alibaba-inc.com/aone/")) {
			aoneImage = container.Image
		}

		//for _, e := range container.Env {
		//	if e.Name == "ali_start_app" && e.Value == "no" {
		//		disableLifeCycle = true
		//		break
		//	}
		//}

		if isDockerVMMode {
			_ = c.updateLifeCycleAndProbe(cm, pod, &container, disableLifeCycle)
		}

		newContainers = append(newContainers, container)
	}
	pod.Spec.Containers = newContainers

	if aoneImage != "" {
		if err = c.appendImagePullSecret(cm, pod, aoneImage); err != nil {
			glog.Errorf("aliPodLifeCycleHook admit err: %v", err)
		}
	}

	return nil
}

func (c *aliPodLifeCycleHook) updateLifeCycleAndProbe(cm configMap, pod *api.Pod, container *api.Container, disableLifeCycle bool) error {
	if disableLifeCycle {
		container.Lifecycle = nil
		container.ReadinessProbe = nil
		return nil
	}

	preStopHandler := api.Handler{
		Exec: &api.ExecAction{
			Command: []string{
				"/bin/sh",
				"-c",
				cm.getConfig("preStop-command", "sudo -u admin /home/admin/stop.sh>/var/log/sigma/stop.log 2>&1"),
			},
		},
	}
	postStartHandler := api.Handler{
		Exec: &api.ExecAction{
			Command: []string{
				"/bin/sh",
				"-c",
				cm.getConfig("postStart-command",
					"for i in $(seq 1 60); do [ -x /home/admin/.start ] && break ; sleep 5 ; done; sudo -u admin /home/admin/.start>/var/log/sigma/start.log 2>&1 && sudo -u admin /home/admin/health.sh>>/var/log/sigma/start.log 2>&1"),
			},
		},
	}
	container.Lifecycle = &api.Lifecycle{
		PreStop:   &preStopHandler,
		PostStart: &postStartHandler,
	}

	if probeDisable := cm.getConfig("probe-disable", "false") == "true"; !probeDisable {
		probeHealth := api.Handler{
			Exec: &api.ExecAction{
				Command: []string{
					"/bin/sh",
					"-c",
					cm.getConfig("probe-command", "sudo -u admin /home/admin/health.sh>/var/log/sigma/health.log 2>&1"),
				},
			},
		}
		probeTimeout, err := strconv.Atoi(cm.getConfig("probe-timeout-seconds", "20"))
		if err != nil {
			probeTimeout = 20
		}
		probePeriod, err := strconv.Atoi(cm.getConfig("probe-period-seconds", "60"))
		if err != nil {
			probePeriod = 60
		}
		specifiedProbePeriodStr := cm.getConfig("probe-period-seconds-specified", "{}")
		specifiedProbePeriodMap := map[string]int{}
		if err := json.Unmarshal([]byte(specifiedProbePeriodStr), &specifiedProbePeriodMap); err == nil {
			if specifiedPeriod, ok := specifiedProbePeriodMap[pod.Labels[sigmak8sapi.LabelAppName]]; ok {
				probePeriod = specifiedPeriod
			}
		}
		container.ReadinessProbe = &api.Probe{
			Handler:             probeHealth,
			InitialDelaySeconds: 20,
			TimeoutSeconds:      int32(probeTimeout),
			PeriodSeconds:       int32(probePeriod),
		}
	} else {
		container.ReadinessProbe = nil
	}

	return nil
}

func (c *aliPodLifeCycleHook) appendImagePullSecret(cm configMap, pod *api.Pod, imageName string) error {
	secretName, err := c.getOrCreateImagePullSecret(pod, "aone", strings.Split(imageName, "/")[0])
	if err != nil {
		return err
	}

	for _, secret := range pod.Spec.ImagePullSecrets {
		if secret.Name == secretName {
			return nil
		}
	}
	pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, api.LocalObjectReference{Name: secretName})
	return nil
}

func (c *aliPodLifeCycleHook) getOrCreateImagePullSecret(pod *api.Pod, username, registryServer string) (string, error) {
	reqUsername, err1 := labels.NewRequirement("username", selection.In, []string{username})
	reqServer, err2 := labels.NewRequirement("server", selection.In, []string{registryServer})
	if err1 != nil {
		return "", err1
	} else if err2 != nil {
		return "", err2
	}

	secrets, err := c.secretLister.Secrets(pod.Namespace).List(labels.NewSelector().Add(*reqUsername, *reqServer, *secretRegistryUserAccountSelector))
	if err != nil {
		return "", err
	}

	// 已经有这个仓库的secret
	if len(secrets) > 0 {
		return secrets[0].Name, nil
	}

	// 没有secret，需要创建
	userSecrets, err := c.secretLister.Secrets("kube-system").List(labels.NewSelector().Add(*reqUsername, *secretRegistryUserAccountSelector))
	if err != nil {
		return "", err
	}
	if len(userSecrets) <= 0 {
		return "", fmt.Errorf("not found registry-user-%s in kube-system", username)
	}
	password := ""
	if pw, ok := userSecrets[0].Data["password"]; ok {
		password = string(pw)
	} else {
		return "", fmt.Errorf("not found data password in kube-system/%s", userSecrets[0].Name)
	}

	newSecret, err := generateSecretDockerRegistryCfg(fmt.Sprintf("registry-secret-%s-", username), username, password, registryServer, "sigma.ali")
	if err != nil {
		return "", fmt.Errorf("failed generateSecretDockerRegistryCfg: %v", err)
	}
	gotSecret, err := c.client.Core().Secrets(pod.Namespace).Create(newSecret)
	if err != nil {
		return "", fmt.Errorf("failed create secret: %v", err)
	}
	return gotSecret.Name, nil
}

func generateSecretDockerRegistryCfg(generateName, username, password, server, email string) (*api.Secret, error) {
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
			Labels: map[string]string{
				"usage":    LabelSecretRegistryUsage,
				"username": username,
				"server":   server,
			},
		},
		Type: api.SecretTypeDockerConfigJson,
		Data: map[string][]byte{},
	}

	dockercfgJsonContent, err := handleDockerCfgJsonContent(username, password, email, server)
	if err != nil {
		return nil, err
	}
	secret.Data[api.DockerConfigJsonKey] = []byte(dockercfgJsonContent)
	return secret, nil
}

// handleDockerCfgJsonContent serializes a ~/.docker/config.json file
func handleDockerCfgJsonContent(username, password, email, server string) (string, error) {
	dockercfgAuth := credentialprovider.DockerConfigEntry{
		Username: username,
		Password: password,
		Email:    email,
	}

	dockerCfgJson := credentialprovider.DockerConfigJson{
		Auths: map[string]credentialprovider.DockerConfigEntry{server: dockercfgAuth},
	}

	jsonStr, err := json.Marshal(dockerCfgJson)
	if err != nil {
		return "", err
	}

	return string(jsonStr), nil
	//return base64.StdEncoding.EncodeToString(jsonStr), nil
}

func dumpJson(v interface{}) string {
	str, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(str)
}
