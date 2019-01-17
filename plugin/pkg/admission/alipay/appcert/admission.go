package appcert

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"k8s.io/apiserver/pkg/admission"
	"sigma3/staging/src/k8s.io/apimachinery/pkg/util/json"
)

const PluginName = "AlipayAppCert"

// kmi invoke configurations
const (
	signatureHeader = "X-KMI-SIGNATURE"
	kmiRespCode     = "resultCode"
	kmiRespData     = "result"
	kmiCodeSuccess  = "1"

	kmiInvokeTimeout = 10

	// prod env
	//KMIEndpoint = "http://kmi.alipay.com/service/getapplocalkey.json"
	//pemKMIPublicKey = ``
	//pemSigmaPrivateKey = ``

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

// AlipayAppCert is an implementation of admission.Interface.
type AlipayAppCert struct {
	*admission.Handler
}

func NewAlipayAppCert() *AlipayAppCert {
	return &AlipayAppCert{Handler: admission.NewHandler(admission.Create)}
}

func (c *AlipayAppCert) Admit(a admission.Attributes) (err error) {
	return nil
}

// internal util functions
func fetchAppIdentity(appname string) (appLocalKey string, err error) {
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
		return "", err
	}
	if kmiResp.ResultCode != kmiCodeSuccess {
		return "", errors.New(
			fmt.Sprintf("failed to invoke kmi, result code: %s, error msg: %s", kmiResp.ResultCode, kmiResp.Result),
		)
	} else {
		return kmiResp.Result, nil
	}
}
