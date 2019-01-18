package appcert

import (
	"fmt"
	"testing"
	"time"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
)

const (
	appname     = "foo"
	appLocalKey = `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`

	// dev env
	kmiEndpoint     = "http://kmi-d6593.alipay.net/service/getapplocalkey.json"
	pemKMIPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA6W+X/Jjwa8L8LwhXMSvF
slGa7RBTu9oWrLImhhtRjt2VngtDFUHOvHztM6ztBjQQe9XmRYYwOm0FxrJQ7G5s
8IkpwDgnPSfiSP2PR/grHB5gdKuHP9fNtXQyp45V1XcmaswzN9l9EU/Tx5Brd1Fi
BiBtT4dFoPty4NaITFOs7xX58BUl5hLitx2vaU/gVaZO+UwIRJw7+VkRJEUjVzOU
UqbBXSYsmK340808OtnbCFnx1UczI6mP+ump+oVfJVWVPbBTwGdHlyDx3Gher6uw
8BgowSl1nqRi10j9KnUGqT+25M7B+kr4QO865XIoGRrQbYayQXSQyDcjQNzTEJq2
jQIDAQAB
-----END PUBLIC KEY-----
`
	pemSigmaPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAwAPoiMv7HRMeYXV2debCZ0i9pZbEhz0LPh//W4P4XdBr9ygP
MCDHDEGAyoI/Iag+nyGW07FRcQle7mBE+8ktjiUWkbL6tSUbSsbfa8qspuQKx9Wc
oyW1FopBlVEhEC+irAinUsuH0PEeaQfd19RQ1+gBYNNwbK20XokggfFc2dGXjsIp
HZ8C6rEUEs00iU7dOzR2RIFGniVEkqAsyVaMRE2CR1p8XJDbrULDOPoJ2+CbNR1G
pMOS2qae251BhMiO/64KSszPc9oHtFXxE0hnuvWuq4Vx+rZ5r0P5ImBXZQVcNo1Y
CVX/R6EtUQpHn/Xmwl+RwvtPBf2L2ltLZ+MNzQIDAQABAoIBAQCoGUPTrq/yLjCk
pY7FfPWoMhhFBQ6cTqavBpgpaAlhJ/u87kcNnURkyFuV7hySvJXF/kPqpAtmaAvB
qGn7+410Kafueb/eIdQYzK3/0fkAShfeBnYQpgw45WSw8ct+PhWtgg3p/+Cw3MYA
sTBXqLn1qli6iaCcpB2JvYbF+6WL0dqaQAD8MFJpzuerFvCwsAKC9m+SBxQR815h
APEeW2SZvM1+sjDYIzc5LQbfUiwmzh6gg2ZIvnIT0LlasB+sN0PFeGSY50W62d1H
BCg7l2WRtuml6qStqya9NBv2KSgD953AmtOApkCO2XiByWuDa6JpaYxFwIkxhpI1
MR28Nf1JAoGBAOnhvXFE3O6o3GIPnGS/0Avw/lA5sSdoip3ED+vpI3NzJejkCzoj
cL/mRmJv3scP/om1O4X+feN6eKPl3u0pVwFcsiwPF0vrykNmOgd6rKzG1CtpEef3
xY7GDkKmi2rVfiyjenDqApsIEwnEc0UJyy2VvhWhc/4r96sk//h1TPcTAoGBANIs
lTnwQgJvf1vBCPnx/SptKYcfkSwZjwyOOaV6Ym8Y4PYC5/UnpKrBGpgcukvQaktV
IJdSgBILzziZlrJlUXHzkONXQfkjmR0xMrdmebmcLwa/5LEIyXje70MorshTGNfQ
3ECazEc/5Oryu7MLSbnJESatxeUaqzQDnQtfOiOfAoGAfPKAhrbHYSkNM8YrQxfG
SdrhwnJP1kHfbBGGf/35VoA5zIWoCdNNNPgMuiIR3j8JOQB9YERpdNHFCaqQwhrH
xI6FEUyuoXzCfedrMPu0rEk8qERlsIuKG5Brpefbq6OK2MYtb41U/wX9RcaR3lwx
E5VgC6ZJlYxfsCsAJPhluckCgYEAotRzkH25RlXHn/h/0dVPRI1qPQuR10696wZN
VwzoMhZeQ3qg5ugdxUTyK6MmGhKQJ2j+ZP4/xrtrgfhMLk4cuWHwgJFbxX904o75
MemsqMZ+EIae0SFzpbdiOu/L6dunRZzE5zCGzzSLUBNapC48ojlKlmLPDN6KgTPD
ecn/KxUCgYBPXA3JWBBib39tumKjKlTswKp64sjsaB1ZPP9ZcUDRGZBMr4ODe6tE
lBMdFQxQAU5lncKxl3X8/s2RKVmSdw7ALXY3pG0SrOEK2GSiwYj832MUwiHmqdYy
C3p+6k9FpfrqAyRQkGJ94IaDTdtgZW0+G5wuG8cs3S8yk5rd71mrcg==
-----END RSA PRIVATE KEY-----
`
)

func TestRegister(t *testing.T) {
	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()
	if len(registered) == 1 && registered[0] == PluginName {
		return
	} else {
		t.Errorf("Register failed")
	}
}

func TestAdmitOtherResources(t *testing.T) {
	pod := newPod(appname)

	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
	}{
		{
			name:     "non-pod resource",
			kind:     "Foo",
			resource: "foos",
			object:   pod,
		},
		{
			name:        "pod subresource",
			kind:        "Pod",
			resource:    "pods",
			subresource: "eviction",
			object:      pod,
		},
		{
			name:     "non-pod object",
			kind:     "Pod",
			resource: "pods",
			object:   &api.Service{},
		},
	}

	for _, tc := range tests {
		handler := NewAlipayAppCert()

		err := handler.Admit(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), pod.Namespace, pod.Name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
	}
}

func TestCheckAppCertSecretExist(t *testing.T) {
	client := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(client, 1*time.Second)
	plugin := NewTestAdmission(t, client, informerFactory)
	pod := newPod(appname)

	exists, err := plugin.checkAppCertSecretExist(appname, pod)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Errorf("check secret exist failed, expected: false, got: true")
	}

	secretName := fmt.Sprintf(AppIdentitySecretNameTemp, appname)
	secret, err := plugin.createAppCertSecret(secretName, appname, appLocalKey)
	if err != nil {
		t.Error(err)
	}

	informerFactory.Core().InternalVersion().Secrets().Informer().GetStore().Add(secret)
	exists, err = plugin.checkAppCertSecretExist(appname, pod)
	if err != nil {
		t.Error(err)
	}
	if !exists {
		t.Errorf("check secret exist failed, expected: true, got: false")
	}

	informerFactory.Core().InternalVersion().Secrets().Informer().GetStore().Delete(secret)
	exists, err = plugin.checkAppCertSecretExist(appname, pod)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Errorf("check secret exist failed, expected: false, got: true")
	}
}

func TestCreateAppCertSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(client, 1*time.Second)
	plugin := NewTestAdmission(t, client, informerFactory)

	secretName := fmt.Sprintf(AppIdentitySecretNameTemp, appname)
	gotSecret, err := plugin.createAppCertSecret(secretName, appname, appLocalKey)
	if err != nil {
		t.Fatalf("create secret error: %v", err)
	}

	if gotSecret.Namespace != appname {
		t.Fatalf("secret namespace expected: %v\n got: %v", appname, gotSecret.Namespace)
	}

	if gotSecret.ObjectMeta.Name != secretName {
		t.Fatalf("secret name expected: %v\n got: %v", secretName, gotSecret.ObjectMeta.Name)
	}

	if string(gotSecret.Data[AppIdentitySecretKey]) != appLocalKey {
		t.Fatalf("secret data expected: %v\n got: %v", appLocalKey, string(gotSecret.Data[AppIdentitySecretKey]))
	}
}

func TestFetchAppIdentity(t *testing.T) {
	ret, err := fetchAppIdentity("foo", kmiEndpoint, pemKMIPublicKey, pemSigmaPrivateKey)
	if err != nil {
		t.Error(err)
	}
	if ret == "" {
		t.Errorf("failed to fetch `app_local_key.json` for app: foo")
	}
}

func NewTestAdmission(t *testing.T, client internalclientset.Interface, f informers.SharedInformerFactory) *alipayAppCert {
	p := NewAlipayAppCert()

	if p.ValidateInitialization() == nil {
		t.Fatalf("plugin ValidateInitialization should return error")
	}

	p.SetInternalKubeClientSet(client)
	p.SetInternalKubeInformerFactory(f)

	if p.ValidateInitialization() != nil {
		t.Fatalf("plugin ValidateInitialization should not return error")
	}
	return p
}

func newPod(appname string) *api.Pod {
	return &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-appcert-pod",
			Namespace: appname,
			Labels: map[string]string{
				sigmak8sapi.LabelAppName: appname,
			},
			Annotations: map[string]string{},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:  "javaweb",
					Image: "pause:2.0",
				},
				{
					Name:  "sidecar",
					Image: "pause:2.0",
				},
			},
		},
	}
}
