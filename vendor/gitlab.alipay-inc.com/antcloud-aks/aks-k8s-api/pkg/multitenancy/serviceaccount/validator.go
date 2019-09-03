package serviceaccount

import (
	"bytes"
	"errors"
	"fmt"

	"k8s.io/api/core/v1"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TenantWiseLegacyClaims(serviceAccount v1.ServiceAccount, secret v1.Secret) (*jwt.Claims, interface{}) {
	return &jwt.Claims{
			Subject: apiserverserviceaccount.MakeUsername(serviceAccount.Namespace, serviceAccount.Name),
		}, &tenantWiseLegacyPrivateClaims{
			Namespace:          serviceAccount.Namespace,
			ServiceAccountName: serviceAccount.Name,
			ServiceAccountUID:  string(serviceAccount.UID),
			SecretName:         secret.Name,
			TenantID:           serviceAccount.Annotations[multitenancy.MultiTenancyAnnotationKeyTenantID],
			WorkspaceID:        serviceAccount.Annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID],
			ClusterID:          serviceAccount.Annotations[multitenancy.MultiTenancyAnnotationKeyClusterID],
		}
}

type tenantWiseLegacyPrivateClaims struct {
	ServiceAccountName string `json:"kubernetes.io/serviceaccount/service-account.name"`
	ServiceAccountUID  string `json:"kubernetes.io/serviceaccount/service-account.uid"`
	SecretName         string `json:"kubernetes.io/serviceaccount/secret.name"`
	Namespace          string `json:"kubernetes.io/serviceaccount/namespace"`
	TenantID           string `json:"kubernetes.io/serviceaccount/tenant-id"`
	WorkspaceID        string `json:"kubernetes.io/serviceaccount/workspace-id"`
	ClusterID          string `json:"kubernetes.io/serviceaccount/cluster-id"`
}

func NewTenantWiseLegacyValidator(lookup bool, clientset kubernetes.Interface) *tenantWiseLegacyValidator {
	return &tenantWiseLegacyValidator{
		lookup: lookup,
		clientset: clientset,
	}
}

type tenantWiseLegacyValidator struct {
	lookup bool
	clientset kubernetes.Interface
}

func (v *tenantWiseLegacyValidator) Validate(tokenData string, public *jwt.Claims, privateObj interface{}) (multitenancy.TenantInfo, string, string, string, error) {
	private, ok := privateObj.(*tenantWiseLegacyPrivateClaims)
	if !ok {
		glog.Errorf("jwt validator expected private claim of type *legacyPrivateClaims but got: %T", privateObj)
		return nil, "", "", "", errors.New("Token could not be validated.")
	}

	// Make sure the claims we need exist
	if len(public.Subject) == 0 {
		return nil, "", "", "", errors.New("sub claim is missing")
	}
	namespace := private.Namespace
	if len(namespace) == 0 {
		return nil, "", "", "", errors.New("namespace claim is missing")
	}
	secretName := private.SecretName
	if len(secretName) == 0 {
		return nil, "", "", "", errors.New("secretName claim is missing")
	}
	serviceAccountName := private.ServiceAccountName
	if len(serviceAccountName) == 0 {
		return nil, "", "", "", errors.New("serviceAccountName claim is missing")
	}
	serviceAccountUID := private.ServiceAccountUID
	if len(serviceAccountUID) == 0 {
		return nil, "", "", "", errors.New("serviceAccountUID claim is missing")
	}

	subjectNamespace, subjectName, err := apiserverserviceaccount.SplitUsername(public.Subject)
	if err != nil || subjectNamespace != namespace || subjectName != serviceAccountName {
		return nil, "", "", "", errors.New("sub claim is invalid")
	}

	if v.lookup {
		// Make sure token hasn't been invalidated by deletion of the secret
		secret, err := v.clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
		if err != nil {
			glog.V(4).Infof("Could not retrieve token %s/%s for service account %s/%s: %v", namespace, secretName, namespace, serviceAccountName, err)
			return nil, "", "", "", errors.New("Token has been invalidated")
		}
		if secret.DeletionTimestamp != nil {
			glog.V(4).Infof("Token is deleted and awaiting removal: %s/%s for service account %s/%s", namespace, secretName, namespace, serviceAccountName)
			return nil, "", "", "", errors.New("Token has been invalidated")
		}
		if bytes.Compare(secret.Data[v1.ServiceAccountTokenKey], []byte(tokenData)) != 0 {
			glog.V(4).Infof("Token contents no longer matches %s/%s for service account %s/%s", namespace, secretName, namespace, serviceAccountName)
			return nil, "", "", "", errors.New("Token does not match server's copy")
		}

		// Make sure service account still exists (name and UID)
		serviceAccount, err := v.clientset.CoreV1().ServiceAccounts(namespace).Get(serviceAccountName, metav1.GetOptions{})
		if err != nil {
			glog.V(4).Infof("Could not retrieve service account %s/%s: %v", namespace, serviceAccountName, err)
			return nil, "", "", "", err
		}
		if serviceAccount.DeletionTimestamp != nil {
			glog.V(4).Infof("Service account has been deleted %s/%s", namespace, serviceAccountName)
			return nil, "", "", "", fmt.Errorf("ServiceAccount %s/%s has been deleted", namespace, serviceAccountName)
		}
		if string(serviceAccount.UID) != serviceAccountUID {
			glog.V(4).Infof("Service account UID no longer matches %s/%s: %q != %q", namespace, serviceAccountName, string(serviceAccount.UID), serviceAccountUID)
			return nil, "", "", "", fmt.Errorf("ServiceAccount UID (%s) does not match claim (%s)", serviceAccount.UID, serviceAccountUID)
		}
	}

	return multitenancy.NewTenantInfo(private.TenantID, private.WorkspaceID, private.ClusterID), private.Namespace, private.ServiceAccountName, private.ServiceAccountUID, nil
}

func (v *tenantWiseLegacyValidator) NewPrivateClaims() interface{} {
	return &tenantWiseLegacyPrivateClaims{}
}
