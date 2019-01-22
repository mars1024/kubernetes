package apiserver

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[kube-apiserver][admission][alipay_image_secret]", func() {
	image := "reg.docker.alibaba-inc.com/sigma-x/nginx-secret:1.15.3"
	defaultSecretName := "sigma-regcred"
	f := framework.NewDefaultFramework("sigma-apiserver")

	It("[ant][smoke]test for defaultImagePullSecret injection", func() {
		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create pod")
		podCfg.Spec.Containers[0].Image = image
		pod := podCfg.DeepCopy()
		pod.Namespace = f.Namespace.Name
		pod.Name = pod.Name + "-image-pull-secret"
		pod.Spec.Containers[0].ImagePullPolicy = v1.PullAlways
		createdPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		defer util.DeletePod(f.ClientSet, createdPod)
		Expect(err).NotTo(HaveOccurred(), "failed get create pod")

		By("check pod imagePullSecret")
		pass := false
		imagePullSecrets := createdPod.Spec.ImagePullSecrets
		for _, imagePullSecret := range imagePullSecrets {
			if imagePullSecret.Name == defaultSecretName {
				pass = true
			}
		}
		Expect(pass).To(BeTrue(), "imagePullSecret injection failed")

		By("check container is running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, createdPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		By("check secret")
		secret, err := f.ClientSet.CoreV1().Secrets(f.Namespace.Name).Get(defaultSecretName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "failed get secret from current namespace")
		Expect(secret).NotTo(BeNil(), "secret is nil in current namespace")
		authData, ok := secret.Data[api.DockerConfigJsonKey]
		Expect(ok).To(BeTrue(), "failed to get secret data")

		defaultImagePullSecret, err := f.ClientSet.CoreV1().Secrets("kube-system").Get(defaultSecretName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "failed get secret from kube-system")
		Expect(defaultImagePullSecret).NotTo(BeNil(), "secret is nil in kube-system")
		defaultAuthData, ok := defaultImagePullSecret.Data[api.DockerConfigJsonKey]
		Expect(ok).To(BeTrue(), "failed to get secret data")

		Expect(authData).Should(Equal(defaultAuthData), "secret is different from default secret")

	})
})

