package test

import (
	"k8s.io/client-go/discovery"
	admissionregistrationv1alpha1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authenticationv1beta1 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	authorizationv1beta1 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	autoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	batchv1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	batchv2alpha1 "k8s.io/client-go/kubernetes/typed/batch/v2alpha1"
	certificatesv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	coordinationv1beta1 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	eventsv1beta1 "k8s.io/client-go/kubernetes/typed/events/v1beta1"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	policyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbacv1alpha1 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"
	settingsv1alpha1 "k8s.io/client-go/kubernetes/typed/settings/v1alpha1"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	storagev1alpha1 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"
	storagev1beta1 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"
)

type FakeClient struct {
}

func (FakeClient) SchedulingV1beta1() schedulingv1beta1.SchedulingV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Scheduling() schedulingv1beta1.SchedulingV1beta1Interface {
	panic("implement me")
}

func (FakeClient) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface {
	panic("implement me")
}

func (FakeClient) AutoscalingV2beta2() autoscalingv2beta2.AutoscalingV2beta2Interface {
	panic("implement me")
}

func (FakeClient) CoordinationV1beta1() coordinationv1beta1.CoordinationV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Coordination() coordinationv1beta1.CoordinationV1beta1Interface {
	panic("implement me")
}
func (FakeClient) Discovery() discovery.DiscoveryInterface {
	panic("implement me")
}

func (FakeClient) AdmissionregistrationV1alpha1() admissionregistrationv1alpha1.AdmissionregistrationV1alpha1Interface {
	panic("implement me")
}

func (FakeClient) AdmissionregistrationV1beta1() admissionregistrationv1beta1.AdmissionregistrationV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Admissionregistration() admissionregistrationv1beta1.AdmissionregistrationV1beta1Interface {
	panic("implement me")
}

func (FakeClient) AppsV1beta1() appsv1beta1.AppsV1beta1Interface {
	panic("implement me")
}

func (FakeClient) AppsV1beta2() appsv1beta2.AppsV1beta2Interface {
	panic("implement me")
}

func (FakeClient) AppsV1() appsv1.AppsV1Interface {
	panic("implement me")
}

func (FakeClient) Apps() appsv1.AppsV1Interface {
	panic("implement me")
}

func (FakeClient) AuthenticationV1() authenticationv1.AuthenticationV1Interface {
	panic("implement me")
}

func (FakeClient) Authentication() authenticationv1.AuthenticationV1Interface {
	panic("implement me")
}

func (FakeClient) AuthenticationV1beta1() authenticationv1beta1.AuthenticationV1beta1Interface {
	panic("implement me")
}

func (FakeClient) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	panic("implement me")
}

func (FakeClient) Authorization() authorizationv1.AuthorizationV1Interface {
	panic("implement me")
}

func (FakeClient) AuthorizationV1beta1() authorizationv1beta1.AuthorizationV1beta1Interface {
	panic("implement me")
}

func (FakeClient) AutoscalingV1() autoscalingv1.AutoscalingV1Interface {
	panic("implement me")
}

func (FakeClient) Autoscaling() autoscalingv1.AutoscalingV1Interface {
	panic("implement me")
}

func (FakeClient) AutoscalingV2beta1() autoscalingv2beta1.AutoscalingV2beta1Interface {
	panic("implement me")
}

func (FakeClient) BatchV1() batchv1.BatchV1Interface {
	panic("implement me")
}

func (FakeClient) Batch() batchv1.BatchV1Interface {
	panic("implement me")
}

func (FakeClient) BatchV1beta1() batchv1beta1.BatchV1beta1Interface {
	panic("implement me")
}

func (FakeClient) BatchV2alpha1() batchv2alpha1.BatchV2alpha1Interface {
	panic("implement me")
}

func (FakeClient) CertificatesV1beta1() certificatesv1beta1.CertificatesV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Certificates() certificatesv1beta1.CertificatesV1beta1Interface {
	panic("implement me")
}

func (FakeClient) CoreV1() corev1.CoreV1Interface {
	panic("implement me")
}

func (FakeClient) Core() corev1.CoreV1Interface {
	panic("implement me")
}

func (FakeClient) EventsV1beta1() eventsv1beta1.EventsV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Events() eventsv1beta1.EventsV1beta1Interface {
	panic("implement me")
}

func (FakeClient) ExtensionsV1beta1() extensionsv1beta1.ExtensionsV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Extensions() extensionsv1beta1.ExtensionsV1beta1Interface {
	panic("implement me")
}

func (FakeClient) NetworkingV1() networkingv1.NetworkingV1Interface {
	panic("implement me")
}

func (FakeClient) Networking() networkingv1.NetworkingV1Interface {
	panic("implement me")
}

func (FakeClient) PolicyV1beta1() policyv1beta1.PolicyV1beta1Interface {
	panic("implement me")
}

func (FakeClient) Policy() policyv1beta1.PolicyV1beta1Interface {
	panic("implement me")
}

func (FakeClient) RbacV1() rbacv1.RbacV1Interface {
	panic("implement me")
}

func (FakeClient) Rbac() rbacv1.RbacV1Interface {
	panic("implement me")
}

func (FakeClient) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	panic("implement me")
}

func (FakeClient) RbacV1alpha1() rbacv1alpha1.RbacV1alpha1Interface {
	panic("implement me")
}

func (FakeClient) SettingsV1alpha1() settingsv1alpha1.SettingsV1alpha1Interface {
	panic("implement me")
}

func (FakeClient) Settings() settingsv1alpha1.SettingsV1alpha1Interface {
	panic("implement me")
}

func (FakeClient) StorageV1beta1() storagev1beta1.StorageV1beta1Interface {
	panic("implement me")
}

func (FakeClient) StorageV1() storagev1.StorageV1Interface {
	panic("implement me")
}

func (FakeClient) Storage() storagev1.StorageV1Interface {
	panic("implement me")
}

func (FakeClient) StorageV1alpha1() storagev1alpha1.StorageV1alpha1Interface {
	panic("implement me")
}
