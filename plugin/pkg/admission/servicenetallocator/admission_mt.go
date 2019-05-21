/*
Copyright 2019 The Kubernetes Authors.

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

package servicenetallocator

import (
	"fmt"
	"net"
	"io"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core/helper"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/registry/core/service/ipallocator"
	"k8s.io/kubernetes/pkg/registry/core/service/portallocator"

	cafeadmission "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/admission"
	clusterslisters "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/listers_generated/cluster/v1alpha1"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster/v1alpha1"
	informers "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/informers_generated/externalversions"
	internalinformers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	coreclient "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	admissioninitializer "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
)

const (
	PluginName                           = "ServiceNetAllocator"
	AnnotationMCName                     = "aks.cafe.sofastack.io/mc"
	AnnotationIgnoreServiceNetAllocation = "aks.cafe.sofastack.io/ignore-service-net"
)

// ServiceNodePort includes protocol and port number of a service NodePort.
type ServiceNodePort struct {
	// The IP protocol for this port. Supports "TCP" and "UDP".
	Protocol api.Protocol

	// The port on each node on which this service is exposed.
	// Default is to auto-allocate a port if the ServiceType of this Service requires one.
	NodePort int32
}

// serviceNetAllocatorPlugin is an implementation of admission.Interface.
type serviceNetAllocatorPlugin struct {
	*admission.Handler

	allocatorFactory *AllocatorFactory
	lister           clusterslisters.MinionClusterLister
	servicesGetter   corelisters.ServiceLister
	coreClient       coreclient.CoreInterface
}

var _ admission.Interface = &serviceNetAllocatorPlugin{}
var _ = cafeadmission.WantsCafeExtensionKubeInformerFactory(&serviceNetAllocatorPlugin{})
var _ = cafeadmission.WantsStorageFactory(&serviceNetAllocatorPlugin{})
var _ = admissioninitializer.WantsInternalKubeClientSet(&serviceNetAllocatorPlugin{})
var _ = admissioninitializer.WantsInternalKubeInformerFactory(&serviceNetAllocatorPlugin{})

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// NewPlugin creates a new service net allocator admission plugin.
func NewPlugin() *serviceNetAllocatorPlugin {
	return &serviceNetAllocatorPlugin{
		Handler: admission.NewHandler(admission.Create, admission.Update, admission.Delete),
		allocatorFactory: &AllocatorFactory{
			ipAllocators:   make(map[string]ipallocator.Interface),
			portAllocators: make(map[string]portallocator.Interface),
		},
	}
}

func (s *serviceNetAllocatorPlugin) SetCafeExtensionKubeInformerFactory(f informers.SharedInformerFactory) {
	clusterInformer := f.Cluster().V1alpha1().MinionClusters()

	s.lister = clusterInformer.Lister().(multitenancymeta.TenantWise).ShallowCopyWithTenant(multitenancy.AKSAdminTenant).(clusterslisters.MinionClusterLister)

	s.SetReadyFunc(func() bool {
		return clusterInformer.Informer().HasSynced()
	})
	f.Start(make(chan struct{}))
}

func (s *serviceNetAllocatorPlugin) SetInternalKubeClientSet(clientset internalclientset.Interface) {
	s.coreClient = clientset.Core()
}

func (s *serviceNetAllocatorPlugin) SetInternalKubeInformerFactory(f internalinformers.SharedInformerFactory) {
	s.servicesGetter = f.Core().InternalVersion().Services().Lister()
}

func (s *serviceNetAllocatorPlugin) SetStorageFactory(factory storage.StorageFactory) {
	s.allocatorFactory.storageFactory = factory
}

func (s *serviceNetAllocatorPlugin) ValidateInitialization() error {
	if utilfeature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		if s.lister == nil {
			return fmt.Errorf("%s requires a lister", PluginName)
		}
		if s.coreClient == nil {
			return fmt.Errorf("%s requires a client", PluginName)
		}
	}
	return nil
}

// Admit allocates cluster ip and node port for services which need info.
func (s *serviceNetAllocatorPlugin) Admit(a admission.Attributes) error {
	if shouldIgnore(a) {
		return nil
	}

	tenant, err := multitenancyutil.TransformTenantInfoFromUser(a.GetUserInfo())
	if err != nil {
		return err
	}
	s = s.ShallowCopyWithTenant(tenant).(*serviceNetAllocatorPlugin)

	var service *api.Service
	if a.GetOperation() == admission.Create || a.GetOperation() == admission.Update {
		// if we can't convert then fail closed since we've already checked that this is supposed to be a service object.
		// this shouldn't normally happen during admission but could happen if an integrator passes a versioned
		// service object rather than an internal object.
		if _, ok := a.GetObject().(*api.Service); !ok {
			return admission.NewForbidden(a, fmt.Errorf("unexpected type %T", a.GetObject()))
		}
		service = a.GetObject().(*api.Service)
	} else {
		service, err = s.servicesGetter.Services(a.GetNamespace()).Get(a.GetName())
		if errors.IsNotFound(err) {
			return err
		}
		if err != nil {
			return admission.NewForbidden(a, err)
		}
	}

	// extract cluster name from service labels
	var minionCluster *v1alpha1.MinionCluster
	clusterName := getServiceClusterName(service)
	if len(clusterName) > 0 {
		var err error
		minionCluster, err = s.lister.Get(clusterName)
		if err != nil {
			return fmt.Errorf("no such minion cluster %v", clusterName)
		}
	} else {
		clusters, err := s.lister.List(labels.Everything())
		if err != nil {
			return fmt.Errorf("failed to list clusters: %v", err)
		}
		var foundLegacyCluster bool
		for _, cluster := range clusters {
			if service.Annotations[multitenancy.MultiTenancyAnnotationKeyTenantID] != cluster.Labels[v1alpha1.LegacyLabelTenantName] {
				continue
			}
			if service.Annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID] != cluster.Labels[v1alpha1.LegacyLabelWorkspaceName] {
				continue
			}
			if service.Annotations[multitenancy.MultiTenancyAnnotationKeyClusterID] != cluster.Labels[v1alpha1.LegacyLabelClusterName] {
				continue
			}
			foundLegacyCluster = true
			minionCluster = cluster
		}
		var foundMinionCluster bool
		for _, cluster := range clusters {
			if service.Labels[v1alpha1.LabelTenantName] != cluster.Labels[v1alpha1.LabelTenantName] {
				continue
			}
			if service.Labels[v1alpha1.LabelWorkspaceName] != cluster.Labels[v1alpha1.LabelWorkspaceName] {
				continue
			}
			if service.Labels[v1alpha1.LabelClusterName] != cluster.Labels[v1alpha1.LabelClusterName] {
				continue
			}
			foundMinionCluster = true
			minionCluster = cluster
		}

		if !foundLegacyCluster && !foundMinionCluster {
			return fmt.Errorf("no such minion cluster %v/%v/%v",
				service.Annotations[multitenancy.MultiTenancyAnnotationKeyTenantID],
				service.Annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID],
				service.Annotations[multitenancy.MultiTenancyAnnotationKeyClusterID],
			)
		}
	}

	switch a.GetOperation() {
	case admission.Create:
		return s.handleServiceCreate(service, minionCluster, a.IsDryRun())
	case admission.Update:
		// require an existing service
		oldService, ok := a.GetOldObject().(*api.Service)
		if !ok {
			return admission.NewForbidden(a, fmt.Errorf("unexpected type %T", a.GetOldObject()))
		}
		return s.handleServiceUpdate(service, oldService, minionCluster, a.IsDryRun())
	case admission.Delete:
		return s.handleServiceDelete(service, minionCluster, a.IsDryRun())
	default:
		return nil
	}

	return nil
}

func (s *serviceNetAllocatorPlugin) ShallowCopyWithTenant(info multitenancy.TenantInfo) interface{} {
	copied := *s
	copied.coreClient = s.coreClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(info).(coreclient.CoreInterface)
	copied.servicesGetter = s.servicesGetter.(multitenancymeta.TenantWise).ShallowCopyWithTenant(info).(corelisters.ServiceLister)
	return &copied
}

func shouldIgnore(a admission.Attributes) bool {
	if a.GetResource().GroupResource() != api.Resource("services") {
		return true
	}
	if len(a.GetSubresource()) != 0 {
		return true
	}

	operation := a.GetOperation()
	if operation != admission.Create && operation != admission.Update && operation != admission.Delete {
		return true
	}
	if obj := a.GetObject(); obj != nil {
		if svc, ok := obj.(*api.Service); ok {
			if svc.Annotations[AnnotationIgnoreServiceNetAllocation] == "true" {
				return true
			}
		}
	}

	return false
}

func (s *serviceNetAllocatorPlugin) handleServiceCreate(service *api.Service, cluster *v1alpha1.MinionCluster, dryRun bool) error {
	serviceIPs, err := s.allocatorFactory.IPAllocatorForCluster(cluster, s.coreClient, s.coreClient)
	if err != nil {
		return err
	}

	releaseServiceIP := false
	defer func() {
		if releaseServiceIP {
			if helper.IsServiceIPSet(service) {
				serviceIPs.Release(net.ParseIP(service.Spec.ClusterIP))
			}
		}
	}()

	if !dryRun {
		if service.Spec.Type != api.ServiceTypeExternalName {
			if releaseServiceIP, err = s.allocateClusterIP(service, cluster); err != nil {
				return err
			}
		}
	}

	serviceNodePorts, err := s.allocatorFactory.NodePortAllocatorForCluster(cluster, s.coreClient, s.coreClient)
	if err != nil {
		return err
	}

	nodePortOp := portallocator.StartOperation(serviceNodePorts, dryRun)
	defer nodePortOp.Finish()

	if service.Spec.Type == api.ServiceTypeNodePort || service.Spec.Type == api.ServiceTypeLoadBalancer {
		if err := initNodePorts(service, nodePortOp); err != nil {
			return err
		}
	}

	if err == nil {
		el := nodePortOp.Commit()
		if el != nil {
			// these should be caught by an eventual reconciliation / restart
			utilruntime.HandleError(fmt.Errorf("error(s) committing service node-ports changes: %v", el))
		}

		releaseServiceIP = false
	}

	return nil
}

func (s *serviceNetAllocatorPlugin) handleServiceUpdate(service, oldService *api.Service, cluster *v1alpha1.MinionCluster, dryRun bool) error {
	serviceIPs, err := s.allocatorFactory.IPAllocatorForCluster(cluster, s.coreClient, s.coreClient)
	if err != nil {
		return err
	}

	releaseServiceIP := false
	defer func() {
		if releaseServiceIP {
			if helper.IsServiceIPSet(service) {
				serviceIPs.Release(net.ParseIP(service.Spec.ClusterIP))
			}
		}
	}()

	serviceNodePorts, err := s.allocatorFactory.NodePortAllocatorForCluster(cluster, s.coreClient, s.coreClient)
	if err != nil {
		return err
	}
	nodePortOp := portallocator.StartOperation(serviceNodePorts, dryRun)
	defer nodePortOp.Finish()

	if !dryRun {
		// Update service from ExternalName to non-ExternalName, should initialize ClusterIP.
		if oldService.Spec.Type == api.ServiceTypeExternalName && service.Spec.Type != api.ServiceTypeExternalName {
			if releaseServiceIP, err = s.allocateClusterIP(service, cluster); err != nil {
				return err
			}
		}
		// Update service from non-ExternalName to ExternalName, should release ClusterIP if exists.
		if oldService.Spec.Type != api.ServiceTypeExternalName && service.Spec.Type == api.ServiceTypeExternalName {
			if helper.IsServiceIPSet(oldService) {
				serviceIPs.Release(net.ParseIP(oldService.Spec.ClusterIP))
			}
		}
	}
	// Update service from NodePort or LoadBalancer to ExternalName or ClusterIP, should release NodePort if exists.
	if (oldService.Spec.Type == api.ServiceTypeNodePort || oldService.Spec.Type == api.ServiceTypeLoadBalancer) &&
		(service.Spec.Type == api.ServiceTypeExternalName || service.Spec.Type == api.ServiceTypeClusterIP) {
		releaseNodePorts(oldService, nodePortOp)
	}
	// Update service from any type to NodePort or LoadBalancer, should update NodePort.
	if service.Spec.Type == api.ServiceTypeNodePort || service.Spec.Type == api.ServiceTypeLoadBalancer {
		if err := updateNodePorts(oldService, service, nodePortOp); err != nil {
			return err
		}
	}

	if err == nil {
		el := nodePortOp.Commit()
		if el != nil {
			// problems should be fixed by an eventual reconciliation / restart
			utilruntime.HandleError(fmt.Errorf("error(s) committing NodePorts changes: %v", el))
		}

		releaseServiceIP = false
	}

	return nil
}

func (s *serviceNetAllocatorPlugin) handleServiceDelete(service *api.Service, cluster *v1alpha1.MinionCluster, dryRun bool) error {
	if dryRun {
		return nil
	}
	if helper.IsServiceIPSet(service) {
		serviceIPs, err := s.allocatorFactory.IPAllocatorForCluster(cluster, s.coreClient, s.coreClient)
		if err != nil {
			return err
		}
		serviceIPs.Release(net.ParseIP(service.Spec.ClusterIP))
	}

	for _, nodePort := range collectServiceNodePorts(service) {
		serviceNodePorts, err := s.allocatorFactory.NodePortAllocatorForCluster(cluster, s.coreClient, s.coreClient)
		if err != nil {
			return err
		}
		err = serviceNodePorts.Release(nodePort)
		if err != nil {
			// these should be caught by an eventual reconciliation / restart
			utilruntime.HandleError(fmt.Errorf("Error releasing service %s node port %d: %v", service.Name, nodePort, err))
		}
	}
	return nil
}

func (s *serviceNetAllocatorPlugin) allocateClusterIP(service *api.Service, cluster *v1alpha1.MinionCluster) (bool, error) {
	serviceIPs, err := s.allocatorFactory.IPAllocatorForCluster(cluster, s.coreClient, s.coreClient)
	if err != nil {
		return false, err
	}
	switch {
	case service.Spec.ClusterIP == "":
		// Allocate next available.
		ip, err := serviceIPs.AllocateNext()
		if err != nil {
			// TODO: what error should be returned here?  It's not a
			// field-level validation failure (the field is valid), and it's
			// not really an internal error.
			return false, errors.NewInternalError(fmt.Errorf("failed to allocate a serviceIP: %v", err))
		}
		service.Spec.ClusterIP = ip.String()
		return true, nil
	case service.Spec.ClusterIP != api.ClusterIPNone && service.Spec.ClusterIP != "":
		// Try to respect the requested IP.
		if err := serviceIPs.Allocate(net.ParseIP(service.Spec.ClusterIP)); err != nil {
			// TODO: when validation becomes versioned, this gets more complicated.
			el := field.ErrorList{field.Invalid(field.NewPath("spec", "clusterIP"), service.Spec.ClusterIP, err.Error())}
			return false, errors.NewInvalid(api.Kind("Service"), service.Name, el)
		}
		return true, nil
	}

	return false, nil
}

// Loop through the service ports list, find one with the same port number and
// NodePort specified, return this NodePort otherwise return 0.
func findRequestedNodePort(port int, servicePorts []api.ServicePort) int {
	for i := range servicePorts {
		servicePort := servicePorts[i]
		if port == int(servicePort.Port) && servicePort.NodePort != 0 {
			return int(servicePort.NodePort)
		}
	}
	return 0
}

func initNodePorts(service *api.Service, nodePortOp *portallocator.PortAllocationOperation) error {
	svcPortToNodePort := map[int]int{}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		allocatedNodePort := svcPortToNodePort[int(servicePort.Port)]
		if allocatedNodePort == 0 {
			// This will only scan forward in the service.Spec.Ports list because any matches
			// before the current port would have been found in svcPortToNodePort. This is really
			// looking for any user provided values.
			np := findRequestedNodePort(int(servicePort.Port), service.Spec.Ports)
			if np != 0 {
				err := nodePortOp.Allocate(np)
				if err != nil {
					// TODO: when validation becomes versioned, this gets more complicated.
					el := field.ErrorList{field.Invalid(field.NewPath("spec", "ports").Index(i).Child("nodePort"), np, err.Error())}
					return errors.NewInvalid(api.Kind("Service"), service.Name, el)
				}
				servicePort.NodePort = int32(np)
				svcPortToNodePort[int(servicePort.Port)] = np
			} else {
				nodePort, err := nodePortOp.AllocateNext()
				if err != nil {
					// TODO: what error should be returned here?  It's not a
					// field-level validation failure (the field is valid), and it's
					// not really an internal error.
					return errors.NewInternalError(fmt.Errorf("failed to allocate a nodePort: %v", err))
				}
				servicePort.NodePort = int32(nodePort)
				svcPortToNodePort[int(servicePort.Port)] = nodePort
			}
		} else if int(servicePort.NodePort) != allocatedNodePort {
			// Note: the current implementation is better, because it saves a NodePort.
			if servicePort.NodePort == 0 {
				servicePort.NodePort = int32(allocatedNodePort)
			} else {
				err := nodePortOp.Allocate(int(servicePort.NodePort))
				if err != nil {
					// TODO: when validation becomes versioned, this gets more complicated.
					el := field.ErrorList{field.Invalid(field.NewPath("spec", "ports").Index(i).Child("nodePort"), servicePort.NodePort, err.Error())}
					return errors.NewInvalid(api.Kind("Service"), service.Name, el)
				}
			}
		}
	}

	return nil
}

// This is O(N), but we expect haystack to be small;
// so small that we expect a linear search to be faster
func containsNumber(haystack []int, needle int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// This is O(N), but we expect serviceNodePorts to be small;
// so small that we expect a linear search to be faster
func containsNodePort(serviceNodePorts []ServiceNodePort, serviceNodePort ServiceNodePort) bool {
	for _, snp := range serviceNodePorts {
		if snp == serviceNodePort {
			return true
		}
	}
	return false
}

func updateNodePorts(oldService, newService *api.Service, nodePortOp *portallocator.PortAllocationOperation) error {
	oldNodePortsNumbers := collectServiceNodePorts(oldService)
	newNodePorts := []ServiceNodePort{}
	portAllocated := map[int]bool{}

	for i := range newService.Spec.Ports {
		servicePort := &newService.Spec.Ports[i]
		nodePort := ServiceNodePort{Protocol: servicePort.Protocol, NodePort: servicePort.NodePort}
		if nodePort.NodePort != 0 {
			if !containsNumber(oldNodePortsNumbers, int(nodePort.NodePort)) && !portAllocated[int(nodePort.NodePort)] {
				err := nodePortOp.Allocate(int(nodePort.NodePort))
				if err != nil {
					el := field.ErrorList{field.Invalid(field.NewPath("spec", "ports").Index(i).Child("nodePort"), nodePort.NodePort, err.Error())}
					return errors.NewInvalid(api.Kind("Service"), newService.Name, el)
				}
				portAllocated[int(nodePort.NodePort)] = true
			}
		} else {
			nodePortNumber, err := nodePortOp.AllocateNext()
			if err != nil {
				// TODO: what error should be returned here?  It's not a
				// field-level validation failure (the field is valid), and it's
				// not really an internal error.
				return errors.NewInternalError(fmt.Errorf("failed to allocate a nodePort: %v", err))
			}
			servicePort.NodePort = int32(nodePortNumber)
			nodePort.NodePort = servicePort.NodePort
		}
		if containsNodePort(newNodePorts, nodePort) {
			return fmt.Errorf("duplicate nodePort: %v", nodePort)
		}
		newNodePorts = append(newNodePorts, nodePort)
	}

	newNodePortsNumbers := collectServiceNodePorts(newService)

	// The comparison loops are O(N^2), but we don't expect N to be huge
	// (there's a hard-limit at 2^16, because they're ports; and even 4 ports would be a lot)
	for _, oldNodePortNumber := range oldNodePortsNumbers {
		if containsNumber(newNodePortsNumbers, oldNodePortNumber) {
			continue
		}
		nodePortOp.ReleaseDeferred(int(oldNodePortNumber))
	}

	return nil
}

func releaseNodePorts(service *api.Service, nodePortOp *portallocator.PortAllocationOperation) {
	nodePorts := collectServiceNodePorts(service)

	for _, nodePort := range nodePorts {
		nodePortOp.ReleaseDeferred(nodePort)
	}
}

func collectServiceNodePorts(service *api.Service) []int {
	var servicePorts []int
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		if servicePort.NodePort != 0 {
			servicePorts = append(servicePorts, int(servicePort.NodePort))
		}
	}
	return servicePorts
}

func getServiceClusterName(service *api.Service) string {
	clusterName, ok := service.Annotations[AnnotationMCName]
	if !ok {
		return ""
	}
	return clusterName
}
