/*
 * 功能描述
 * 创建应用 Pod 时，从 KMI 获取应用身份凭证文件 (app_local_key.json)，并存放到 Pod 所属的 Namespace 的 Secret 中
 * Secret 的 Name 为应用名 (`appname`)，Secret 内键为 `AppIdentitySecretKey` 的值，存放 app_local_key.json 的内容
 *
 * 插件配置
 * 插件运行前，需要在 Namespace 为 `AppCertsSecretNamespace` 的 Secret 中，添加 Name 为 `PluginConfSecretName` 的配置
 * 配置示例：
 * map[string][]byte {
 *     "kmi-endpoint": []byte("http://kmi-d6593.alipay.net/service/getapplocalkey.json"),
 *     "kmi-public-key": []byte(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA6W+X/Jjwa8L8LwhXMSvF
slGa7RBTu9oWrLImhhtRjt2VngtDFUHOvHztM6ztBjQQe9XmRYYwOm0FxrJQ7G5s
8IkpwDgnPSfiSP2PR/grHB5gdKuHP9fNtXQyp45V1XcmaswzN9l9EU/Tx5Brd1Fi
BiBtT4dFoPty4NaITFOs7xX58BUl5hLitx2vaU/gVaZO+UwIRJw7+VkRJEUjVzOU
UqbBXSYsmK340808OtnbCFnx1UczI6mP+ump+oVfJVWVPbBTwGdHlyDx3Gher6uw
8BgowSl1nqRi10j9KnUGqT+25M7B+kr4QO865XIoGRrQbYayQXSQyDcjQNzTEJq2
jQIDAQAB
-----END PUBLIC KEY-----
`),
 *     "sigma-private-key": []byte(`-----BEGIN RSA PRIVATE KEY-----
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
`),
 * }
*/
package appcert

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

// plugin conf
const (
	PluginName               = "AlipayAppCert"
	AppCertsSecretNamespace  = "app-certs"
	PluginConfSecretName     = "basic-conf"
	KMIEndpointSecretKey     = "kmi-endpoint"
	KMIPublicKeySecretKey    = "kmi-public-key"
	SigmaPrivateKeySecretKey = "sigma-private-key"
)

// app_local_key.json secret conf
const (
	AppIdentitySecretNameTemp = "%s-local-key"
	AppIdentitySecretKey      = "app-local-key"
)

// kmi invoke configurations
const (
	signatureHeader  = "X-KMI-SIGNATURE"
	kmiCodeNoCert    = "0"
	kmiCodeSuccess   = "1"
	kmiInvokeTimeout = 10
)

type kmiInvokeResp struct {
	ResultCode string `json:"resultCode"`
	Result     string `json:"result"`
	KmiCa      string `json:"kmiCa"`
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayAppCert(), nil
	})
}

// alipayAppCert is an implementation of admission.Interface.
type alipayAppCert struct {
	*admission.Handler
	client       internalclientset.Interface
	secretLister settingslisters.SecretLister
}

var (
	_ = admission.Interface(&alipayAppCert{})
	_ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&alipayAppCert{})
	_ = kubeapiserveradmission.WantsInternalKubeClientSet(&alipayAppCert{})
)

// NewAlipayAppCert creates a new admission plugin
func NewAlipayAppCert() *alipayAppCert {
	return &alipayAppCert{Handler: admission.NewHandler(admission.Create)}
}

// ValidateInitialization checks whether the plugin was correctly initialized.
func (plugin *alipayAppCert) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	return nil
}

func (plugin *alipayAppCert) SetInternalKubeClientSet(client internalclientset.Interface) {
	plugin.client = client
}

func (plugin *alipayAppCert) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	secretInformer := f.Core().InternalVersion().Secrets()
	plugin.secretLister = secretInformer.Lister()
	plugin.SetReadyFunc(secretInformer.Informer().HasSynced)
}

func (plugin *alipayAppCert) Admit(a admission.Attributes) (err error) {
	// this admission plugin only work on application's pod resource
	if shouldIgnore(a) {
		return nil
	}

	// after shouldIgnore, the resource must be `pods`
	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	// check if mosn is enabled
	if pod.Annotations[alipaysigmak8sapi.MOSNSidecarInject] != string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled) {
		// no need to fetch app_local_key.json if mosn is not enabled
		return nil
	}

	// fetch appname from pod labels
	appname := pod.Labels[sigmak8sapi.LabelAppName]
	if appname == "" {
		glog.Error("failed to fetch appname from labels")
		return admission.NewForbidden(a, fmt.Errorf("failed to fetch appname from labels"))
	}

	// make sure pod.Namespace is not empty
	if pod.Namespace == "" {
		namespace := a.GetNamespace()
		pod.Namespace = namespace
	}

	// check secret exist
	exist, err := plugin.checkAppCertSecretExist(appname, pod)
	if err != nil {
		glog.Errorf("failed to check secret exist, err msg: %v", err)
		return admission.NewForbidden(a, fmt.Errorf("failed to check secret exist, err msg: %v", err))
	}
	if exist {
		// app_local_key.json has been set to secret
		return nil
	}

	// fetch plugin conf from secret
	pluginConfSecret, err := plugin.secretLister.Secrets(AppCertsSecretNamespace).Get(PluginConfSecretName)
	if err != nil {
		glog.Errorf("failed to fetch plugin configuration from secret, err msg: %v", err)
		return admission.NewForbidden(a, fmt.Errorf("failed to fetch plugin configuration from secret, err msg: %v", err))
	}

	kmiEndpoint, ok := pluginConfSecret.Data[KMIEndpointSecretKey]
	if !ok || len(kmiEndpoint) <= 0 {
		glog.Errorf("failed to decode plugin conf, key[%s]", KMIEndpointSecretKey)
		return nil
	}

	pemKMIPublicKey, ok := pluginConfSecret.Data[KMIPublicKeySecretKey]
	if !ok || len(pemKMIPublicKey) <= 0 {
		glog.Errorf("failed to decode plugin conf, key[%s]", KMIPublicKeySecretKey)
		return nil
	}

	pemSigmaPrivateKey, ok := pluginConfSecret.Data[SigmaPrivateKeySecretKey]
	if !ok || len(pemSigmaPrivateKey) <= 0 {
		glog.Errorf("failed to decode plugin conf, key[%s]", SigmaPrivateKeySecretKey)
		return nil
	}

	// fetch the app_local_key.json
	appLocalKey, err := fetchAppIdentity(appname, string(kmiEndpoint), string(pemKMIPublicKey), string(pemSigmaPrivateKey))
	if err != nil {
		glog.Errorf("failed to fetch app_local_key.json from kmi, err msg: %v", err)
		return admission.NewForbidden(a, fmt.Errorf("failed to fetch app_local_key.json from kmi, err msg: %v", err))
	}
	if appLocalKey == "" {
		return nil
	}

	// save app_local_key.json to secret
	secretName := fmt.Sprintf(AppIdentitySecretNameTemp, appname)
	gotSecret, err := plugin.createAppCertSecret(secretName, pod.Namespace, appLocalKey)
	if err != nil {
		glog.Errorf("failed to save secret, err msg: %v", err)
		return admission.NewForbidden(a, fmt.Errorf("failed to save secret, err msg: %v", err))
	} else {
		glog.Infof("save app_loca_key to secret, secret name: %s", gotSecret.Name)
	}

	return nil
}

func (plugin *alipayAppCert) checkAppCertSecretExist(appname string, pod *api.Pod) (bool, error) {
	secretName := fmt.Sprintf(AppIdentitySecretNameTemp, appname)
	_, err := plugin.secretLister.Secrets(pod.Namespace).Get(secretName)

	// AppCertSecret is already exists in pod's namespace.
	if err == nil {
		return true, nil
	}

	if errors.IsNotFound(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (plugin *alipayAppCert) createAppCertSecret(secretName string, secretNamespace string, appLocalKey string) (*api.Secret, error) {
	appCertSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: api.SecretTypeOpaque,
		Data: map[string][]byte{},
	}
	appCertSecret.Data[AppIdentitySecretKey] = []byte(appLocalKey)

	gotSecret, err := plugin.client.Core().Secrets(secretNamespace).Create(appCertSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret, err msg: %v", err)
	}
	return gotSecret, nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != api.Resource("pods") {
		return true
	}
	if a.GetSubresource() != "" {
		// only run the checks below on pods proper and not subresources
		return true
	}

	_, ok := a.GetObject().(*api.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	return false
}

// fetch app_local_key.json from KMI
func fetchAppIdentity(appname string, kmiEndpoint string, pemKMIPublicKey string, pemSigmaPrivateKey string) (appLocalKey string, err error) {
	// init private key & public key
	block, _ := pem.Decode([]byte(pemKMIPublicKey))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	block, _ = pem.Decode([]byte(pemSigmaPrivateKey))
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// build request body
	plainReqBody, _ := json.Marshal(map[string]string{
		"appName":   appname,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	})
	random := rand.Reader
	cipherReqBody, err := rsa.EncryptPKCS1v15(random, pub.(*rsa.PublicKey), plainReqBody)
	if err != nil {
		return "", err
	}
	reqBody := []byte(base64.StdEncoding.EncodeToString(cipherReqBody))

	// build signature header
	hashed := sha256.Sum256(plainReqBody)
	signBytes, err := rsa.SignPKCS1v15(random, priv, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	sign := base64.StdEncoding.EncodeToString(signBytes)

	// launch request
	client := &http.Client{
		Timeout: kmiInvokeTimeout * time.Second,
	}
	req, err := http.NewRequest("POST", kmiEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set(signatureHeader, sign)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// parse the resp
	var kmiResp kmiInvokeResp
	respBody, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(respBody, &kmiResp); err != nil {
		return "", fmt.Errorf("failed to parse kmi response: %v", err)
	}
	if kmiResp.ResultCode == kmiCodeSuccess {
		// fetch app_local_key.json success
		return kmiResp.Result, nil
	} else if kmiResp.ResultCode == kmiCodeNoCert {
		// no app_local_key.json for indicate app, return ""
		return "", nil
	} else {
		return "", fmt.Errorf("failed to invoke kmi, result code: %s, error msg: %s", kmiResp.ResultCode, kmiResp.Result)
	}
}
