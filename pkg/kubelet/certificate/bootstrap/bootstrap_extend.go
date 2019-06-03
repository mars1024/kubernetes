package bootstrap

import (
	"crypto/sha512"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang/glog"

	certificates "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	certificatesclient "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	restclient "k8s.io/client-go/rest"
	bootstrapapi "k8s.io/client-go/tools/bootstrap/token/api"
	bootstraputil "k8s.io/client-go/tools/bootstrap/token/util"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate/csr"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

const (
	tenantIDLabel    = "cafe.sofastack.io/tenant"
	workspaceIDLabel = "cafe.sofastack.io/workspace"
	clusterIDLabel   = "cafe.sofastack.io/cluster"
)

var (
	TenantID    string
	WorkspaceID string
	ClusterID   string
)

func getTenantInfo(bootstrapToken string, config *restclient.Config) (multitenancy.TenantInfo, error) {
	glog.V(1).Infof("trying to get tenant info from bootstrap token")
	tokenID, _, err := parseToken(bootstrapToken)
	if err != nil {
		return nil, err
	}

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	secretName := bootstrapapi.BootstrapTokenSecretPrefix + tokenID
	secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceSystem).Get(secretName, metav1.GetOptions{})
	// ignore the error here to not break anything
	if err != nil {
		glog.Warningf("failed to get secret: %v", err)
		glog.Warningf("If for multitenancy, please allow to get the secret. If not, please ignore.")
		return nil, nil
	}

	tenantID := secret.Labels[tenantIDLabel]
	if len(tenantID) == 0 {
		tenantID = secret.Annotations[multitenancy.MultiTenancyAnnotationKeyTenantID]
	}
	workspaceID := secret.Labels[workspaceIDLabel]
	if len(workspaceID) == 0 {
		workspaceID = secret.Annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID]
	}
	clusterID := secret.Labels[clusterIDLabel]
	if len(clusterID) == 0 {
		clusterID = secret.Annotations[multitenancy.MultiTenancyAnnotationKeyClusterID]
	}

	// mark sure tenant info is completed
	if len(tenantID) > 0 && len(workspaceID) > 0 && len(clusterID) > 0 {
		exportTenantInfo(tenantID, workspaceID, clusterID)
		return multitenancy.NewTenantInfo(tenantID, workspaceID, clusterID), nil
	}

	glog.V(1).Infof("found incomplete tenant info or nil tenant info. TenantID: %q, ClusterID: %q, WorkspaceID: %q", tenantID, clusterID, workspaceID)
	return nil, nil
}

func exportTenantInfo(tenantID, workspaceID, clusterID string) {
	glog.V(4).Infof("export tenant info for future kubelet certficate rotation")
	TenantID = tenantID
	WorkspaceID = workspaceID
	ClusterID = clusterID
}

// parseToken tries and parse a valid token from a string.
// A token ID and token secret are returned in case of success, an error otherwise.
func parseToken(s string) (string, string, error) {
	split := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(s)
	if len(split) != 3 {
		return "", "", fmt.Errorf("token [%q] was not of form [%q]", s, bootstrapapi.BootstrapTokenPattern)
	}
	return split[1], split[2], nil
}

// RequestNodeCertificate will create a certificate signing request for a node
// (Organization and CommonName for the CSR will be set as expected for node
// certificates) and send it to API server, then it will watch the object's
// status, once approved by API server, it will return the API server's issued
// certificate (pem-encoded). If there is any errors, or the watch timeouts, it
// will return an error. This is intended for use on nodes (kubelet and
// kubeadm).
// Note: original forked from RequestNodeCertificate
func requestNodeCertificateExtended(config *restclient.Config, csrclient certificatesclient.CertificateSigningRequestInterface, privateKeyData []byte, nodeName types.NodeName) (certData []byte, err error) {
	tenantInfo, err := getTenantInfo(config.BearerToken, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant info for certificate request: %v", err)
	}

	subject := &pkix.Name{
		Organization: []string{"system:nodes"},
		CommonName:   "system:node:" + string(nodeName),
	}
	// append tenant info
	if tenantInfo != nil {
		glog.V(1).Infof("appending tenant info %v to CSR", tenantInfo)
		subject.Organization = append(subject.Organization,
			fmt.Sprintf("%s%s", multitenancy.X509CertificateClusterIDPrefix, tenantInfo.GetClusterID()),
			fmt.Sprintf("%s%s", multitenancy.X509CertificateWorkspaceIDPrefix, tenantInfo.GetWorkspaceID()),
			fmt.Sprintf("%s%s", multitenancy.X509CertificateTenantIDPrefix, tenantInfo.GetTenantID()))
	}

	privateKey, err := certutil.ParsePrivateKeyPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("invalid private key for certificate request: %v", err)
	}
	csrData, err := certutil.MakeCSR(privateKey, subject, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to generate certificate request: %v", err)
	}

	usages := []certificates.KeyUsage{
		certificates.UsageDigitalSignature,
		certificates.UsageKeyEncipherment,
		certificates.UsageClientAuth,
	}
	name := digestedName(privateKeyData, subject, usages)
	req, err := csr.RequestCertificate(csrclient, csrData, name, usages, privateKey)
	if err != nil {
		return nil, err
	}
	return csr.WaitForCertificate(csrclient, req, 3600*time.Second)
}

// This digest should include all the relevant pieces of the CSR we care about.
// We can't direcly hash the serialized CSR because of random padding that we
// regenerate every loop and we include usages which are not contained in the
// CSR. This needs to be kept up to date as we add new fields to the node
// certificates and with ensureCompatible.
func digestedName(privateKeyData []byte, subject *pkix.Name, usages []certificates.KeyUsage) string {
	hash := sha512.New512_256()

	// Here we make sure two different inputs can't write the same stream
	// to the hash. This delimiter is not in the base64.URLEncoding
	// alphabet so there is no way to have spill over collisions. Without
	// it 'CN:foo,ORG:bar' hashes to the same value as 'CN:foob,ORG:ar'
	const delimiter = '|'
	encode := base64.RawURLEncoding.EncodeToString

	write := func(data []byte) {
		hash.Write([]byte(encode(data)))
		hash.Write([]byte{delimiter})
	}

	write(privateKeyData)
	write([]byte(subject.CommonName))
	for _, v := range subject.Organization {
		write([]byte(v))
	}
	for _, v := range usages {
		write([]byte(v))
	}

	return "node-csr-" + encode(hash.Sum(nil))
}
