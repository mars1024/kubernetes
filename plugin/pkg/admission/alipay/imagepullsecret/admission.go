/*
Copyright 2014 The Kubernetes Authors.

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

package imagepullsecret

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/credentialprovider"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	// PluginName indicates name of the plugin.
	PluginName = "AlipayImageSecret"
	// DefaultImagePullSecret is the name of the default imagePullSecret name to set on pods.
	DefaultImagePullSecret = "sigma-regcred"
	// DefaultImageRegistryServer is the name of image registry.
	// All images should have prefix DefaultImageRegistryServer.
	DefaultImageRegistryServer = "reg.docker.alibaba-inc.com"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		ImageSecretAdmission := NewPlugin()
		return ImageSecretAdmission, nil
	})
}

var _ = admission.Interface(&imageSecret{})
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&imageSecret{})
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&imageSecret{})

type imageSecret struct {
	*admission.Handler
	client       internalclientset.Interface
	secretLister settingslisters.SecretLister
}

var _ admission.MutationInterface = &imageSecret{}

// NewPlugin returns an admission.Interface implementation which will check or add imagePullSecrets to pod:
// 1. If the pod does not specify DefaultImagePullSecret, it appends the pod's imagePullSecrets with DefaultImagePullSecret.
func NewPlugin() *imageSecret {
	return &imageSecret{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (i *imageSecret) ValidateInitialization() error {
	if i.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	return nil
}

func (i *imageSecret) SetInternalKubeClientSet(client internalclientset.Interface) {
	i.client = client
}

func (i *imageSecret) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	secretInformer := f.Core().InternalVersion().Secrets()
	i.secretLister = secretInformer.Lister()
	i.SetReadyFunc(func() bool { return secretInformer.Informer().HasSynced() })
}

func (i *imageSecret) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	pod := a.GetObject().(*api.Pod)
	glog.V(3).Infof("imageSecret starts to admit %s/%s, operation: %v, pod: %v", pod.Namespace, pod.Name, a.GetOperation(), dumpJson(&pod.ObjectMeta))

	if a.GetOperation() != admission.Create && a.GetOperation() != admission.Update {
		return nil
	}

	if pod.Namespace == "" {
		namespace := a.GetNamespace()
		pod.Namespace = namespace
	}

	// Add DefaultImagePullSecret to pod.
	secretName := DefaultImagePullSecret
	if err = i.appendImagePullSecret(pod, secretName); err != nil {
		glog.Errorf("imageSecret admit err: %v", err)
		return err
	}

	return nil
}

func (i *imageSecret) appendImagePullSecret(pod *api.Pod, secretName string) error {
	secretName, err := i.getOrCreateImagePullSecret(pod)
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

func (i *imageSecret) getOrCreateImagePullSecret(pod *api.Pod) (string, error) {

	secret, err := i.secretLister.Secrets(pod.Namespace).Get(DefaultImagePullSecret)

	// DefaultImagePullSecret is already exists in pod's namespace.
	if err == nil {
		return secret.Name, nil
	}

	if !errors.IsNotFound(err) {
		return "", err
	}

	// Try to create a secret.
	// Get image pull secret from namespace kube-system
	imagePullSecret, err := i.secretLister.Secrets("kube-system").Get(DefaultImagePullSecret)
	if err != nil {
		glog.Errorf("Failed to get secret in namespace kube-system")
		return "", err
	}

	if imagePullSecret == nil {
		return "", fmt.Errorf("not found image pull secret in kube-system")
	}

	authData, ok := imagePullSecret.Data[api.DockerConfigJsonKey]
	if !ok {
		return "", fmt.Errorf("failed to get data")
	}

	dockerCfg := &credentialprovider.DockerConfigJson{}
	err = json.Unmarshal(authData, dockerCfg)
	if err != nil {
		return "", err
	}

	defaultAuth, exists := dockerCfg.Auths[DefaultImageRegistryServer]
	if !exists {
		return "", fmt.Errorf("no auth info found about server: %s", DefaultImageRegistryServer)
	}

	username := defaultAuth.Username
	password := defaultAuth.Password
	email := defaultAuth.Email
	server := DefaultImageRegistryServer

	userSecret, err := generateSecretDockerRegistryCfg(DefaultImagePullSecret, username, password, email, server)
	if err != nil {
		return "", fmt.Errorf("failed generateSecretDockerRegistryCfg: %v", err)
	}
	gotSecret, err := i.client.Core().Secrets(pod.Namespace).Create(userSecret)
	if err != nil {
		return "", fmt.Errorf("failed create secret: %v", err)
	}
	return gotSecret.Name, nil
}

func generateSecretDockerRegistryCfg(seretName, username, password, email, server string) (*api.Secret, error) {
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: seretName,
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
}

func shouldIgnore(a admission.Attributes) bool {
	if a.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}
	obj := a.GetObject()
	if obj == nil {
		return true
	}
	_, ok := obj.(*api.Pod)
	if !ok {
		return true
	}

	return false
}

func dumpJson(v interface{}) string {
	str, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(str)
}
