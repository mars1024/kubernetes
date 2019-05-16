package apiserver

import (
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/kube-aggregator/pkg/apiserver"

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/kube-aggregator/pkg/controllers/autoregister"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/master/controller/crdregistration"
	multitenancycrdregistration "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/master/controller/crdregistration"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/apiregistration/internalversion"
	"os"
	"strings"
)

const (
	AggregationEnableBasic         = "CAFE_BASIC_ENABLED"
	AggregationEnableCafeExtension = "CAFE_EXTENSION_ENABLED"
	AggregationEnableCafeCluster   = "CAFE_CLUSTER_ENABLED"
	AggregationEnableCafeMetrics   = "CAFE_METRICS_ENABLED"
)

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func CompleteConfig(cfg *apiserver.Config) CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.EnableDiscovery = false
	c.GenericConfig.Version = &version.Info{
		Major: "0",
		Minor: "1",
	}

	return CompletedConfig{&c}
}

func CreateAggregatorServer(aggregatorConfig *apiserver.Config, delegateAPIServer genericapiserver.DelegationTarget, apiExtensionInformers apiextensionsinformers.SharedInformerFactory) (*APIAggregator, error) {
	aggregatorServer, err := CompleteConfig(aggregatorConfig).NewWithDelegate(delegateAPIServer)
	if err != nil {
		return nil, err
	}

	// create controllers for auto-registration
	apiRegistrationClient, err := apiregistrationclient.NewForConfig(aggregatorConfig.GenericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}
	autoRegistrationController := autoregister.NewAutoRegisterController(aggregatorServer.APIRegistrationInformers.Apiregistration().InternalVersion().APIServices(), apiRegistrationClient, aggregatorServer)
	apiServicesToRegister(delegateAPIServer, autoRegistrationController)
	crdRegistrationController := crdregistration.NewAutoRegistrationController(
		apiExtensionInformers.Apiextensions().InternalVersion().CustomResourceDefinitions(),
		autoRegistrationController)

	aggregatorServer.GenericAPIServer.AddPostStartHook("kube-apiserver-autoregistration", func(context genericapiserver.PostStartHookContext) error {
		crdRegistrationControllerWithMultiTenancy := multitenancycrdregistration.NewAutoRegistrationController(
			apiExtensionInformers.Apiextensions().InternalVersion().CustomResourceDefinitions(),
			autoRegistrationController,
		)
		go crdRegistrationControllerWithMultiTenancy.Run(5, context.StopCh)
		go func() {
			// let the CRD controller process the initial set of CRDs before starting the autoregistration controller.
			// this prevents the autoregistration controller's initial sync from deleting APIServices for CRDs that still exist.
			// we only need to do this if CRDs are enabled on this server.  We can't use discovery because we are the source for discovery.
			if aggregatorConfig.GenericConfig.MergedResourceConfig.AnyVersionForGroupEnabled("apiextensions.k8s.io") {
				crdRegistrationController.WaitForInitialSync()
			}
			autoRegistrationController.Run(5, context.StopCh)
		}()
		return nil
	})

	return aggregatorServer, nil
}

func apiServicesToRegister(delegateAPIServer genericapiserver.DelegationTarget, registration autoregister.AutoAPIServiceRegistration) []*apiregistration.APIService {
	apiServices := []*apiregistration.APIService{}

	for _, curr := range delegateAPIServer.ListedPaths() {
		if curr == "/api/v1" {
			apiService := makeAPIService(schema.GroupVersion{Group: "", Version: "v1"})
			registration.AddAPIServiceToSyncOnStart(apiService)
			apiServices = append(apiServices, apiService)
			continue
		}

		if !strings.HasPrefix(curr, "/apis/") {
			continue
		}
		// this comes back in a list that looks like /apis/rbac.authorization.k8s.io/v1alpha1
		tokens := strings.Split(curr, "/")
		if len(tokens) != 4 {
			continue
		}

		apiService := makeAPIService(schema.GroupVersion{Group: tokens[2], Version: tokens[3]})
		if apiService == nil {
			continue
		}
		registration.AddAPIServiceToSyncOnStart(apiService)
		apiServices = append(apiServices, apiService)
	}
	if len(os.Getenv(AggregationEnableCafeExtension)) > 0 {
		// Forward compatibility for cloud alipay sites
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1alpha1.apps.cafe.cloud.alipay.com",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "apps.cafe.cloud.alipay.com",
				Version:              "v1alpha1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1alpha1.cluster.cafe.cloud.alipay.com",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "cluster.cafe.cloud.alipay.com",
				Version:              "v1alpha1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1.serverless.cafe.cloud.alipay.com",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "serverless.cafe.cloud.alipay.com",
				Version:              "v1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
	}
	if len(os.Getenv(AggregationEnableCafeCluster)) > 0 {
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1alpha1.cluster.aks.cafe.sofastack.io",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "cluster.aks.cafe.sofastack.io",
				Version:              "v1alpha1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
	}
	if len(os.Getenv(AggregationEnableBasic)) > 0 {
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1.cafe.sofastack.io",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "cafe.sofastack.io",
				Version:              "v1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
	}
	if len(os.Getenv(AggregationEnableCafeMetrics)) > 0 {
		registration.AddAPIServiceToSyncOnStart(&apiregistration.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "v1beta1.metrics.k8s.io",
			},
			Spec: apiregistration.APIServiceSpec{
				Group:                "metrics.k8s.io",
				Version:              "v1beta1",
				GroupPriorityMinimum: 1000,
				VersionPriority:      100,
			},
		})
	}

	return apiServices
}

func makeAPIService(gv schema.GroupVersion) *apiregistration.APIService {
	apiServicePriority, ok := apiVersionPriorities[gv]
	if !ok {
		// if we aren't found, then we shouldn't register ourselves because it could result in a CRD group version
		// being permanently stuck in the APIServices list.
		glog.Infof("Skipping APIService creation for %v", gv)
		return nil
	}
	return &apiregistration.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: gv.Version + "." + gv.Group},
		Spec: apiregistration.APIServiceSpec{
			Group:                gv.Group,
			Version:              gv.Version,
			GroupPriorityMinimum: apiServicePriority.group,
			VersionPriority:      apiServicePriority.version,
		},
	}
}

// The proper way to resolve this letting the aggregator know the desired group and version-within-group order of the underlying servers
// is to refactor the genericapiserver.DelegationTarget to include a list of priorities based on which APIs were installed.
// This requires the APIGroupInfo struct to evolve and include the concept of priorities and to avoid mistakes, the core storage map there needs to be updated.
// That ripples out every bit as far as you'd expect, so for 1.7 we'll include the list here instead of being built up during storage.
var apiVersionPriorities = map[schema.GroupVersion]priority{
	{Group: "", Version: "v1"}: {group: 18000, version: 1},
	// extensions is above the rest for CLI compatibility, though the level of unqualified resource compatibility we
	// can reasonably expect seems questionable.
	{Group: "extensions", Version: "v1beta1"}: {group: 17900, version: 1},
	// to my knowledge, nothing below here collides
	{Group: "apps", Version: "v1beta1"}:                          {group: 17800, version: 1},
	{Group: "apps", Version: "v1beta2"}:                          {group: 17800, version: 9},
	{Group: "apps", Version: "v1"}:                               {group: 17800, version: 15},
	{Group: "events.k8s.io", Version: "v1beta1"}:                 {group: 17750, version: 5},
	{Group: "authentication.k8s.io", Version: "v1"}:              {group: 17700, version: 15},
	{Group: "authentication.k8s.io", Version: "v1beta1"}:         {group: 17700, version: 9},
	{Group: "authorization.k8s.io", Version: "v1"}:               {group: 17600, version: 15},
	{Group: "authorization.k8s.io", Version: "v1beta1"}:          {group: 17600, version: 9},
	{Group: "autoscaling", Version: "v1"}:                        {group: 17500, version: 15},
	{Group: "autoscaling", Version: "v2beta1"}:                   {group: 17500, version: 9},
	{Group: "autoscaling", Version: "v2beta2"}:                   {group: 17500, version: 1},
	{Group: "batch", Version: "v1"}:                              {group: 17400, version: 15},
	{Group: "batch", Version: "v1beta1"}:                         {group: 17400, version: 9},
	{Group: "batch", Version: "v2alpha1"}:                        {group: 17400, version: 9},
	{Group: "certificates.k8s.io", Version: "v1beta1"}:           {group: 17300, version: 9},
	{Group: "networking.k8s.io", Version: "v1"}:                  {group: 17200, version: 15},
	{Group: "policy", Version: "v1beta1"}:                        {group: 17100, version: 9},
	{Group: "rbac.authorization.k8s.io", Version: "v1"}:          {group: 17000, version: 15},
	{Group: "rbac.authorization.k8s.io", Version: "v1beta1"}:     {group: 17000, version: 12},
	{Group: "rbac.authorization.k8s.io", Version: "v1alpha1"}:    {group: 17000, version: 9},
	{Group: "settings.k8s.io", Version: "v1alpha1"}:              {group: 16900, version: 9},
	{Group: "storage.k8s.io", Version: "v1"}:                     {group: 16800, version: 15},
	{Group: "storage.k8s.io", Version: "v1beta1"}:                {group: 16800, version: 9},
	{Group: "storage.k8s.io", Version: "v1alpha1"}:               {group: 16800, version: 1},
	{Group: "apiextensions.k8s.io", Version: "v1beta1"}:          {group: 16700, version: 9},
	{Group: "admissionregistration.k8s.io", Version: "v1"}:       {group: 16700, version: 15},
	{Group: "admissionregistration.k8s.io", Version: "v1beta1"}:  {group: 16700, version: 12},
	{Group: "admissionregistration.k8s.io", Version: "v1alpha1"}: {group: 16700, version: 9},
	{Group: "scheduling.k8s.io", Version: "v1beta1"}:             {group: 16600, version: 12},
	{Group: "scheduling.k8s.io", Version: "v1alpha1"}:            {group: 16600, version: 9},
	{Group: "coordination.k8s.io", Version: "v1beta1"}:           {group: 16500, version: 9},
	// Append a new group to the end of the list if unsure.
	// You can use min(existing group)-100 as the initial value for a group.
	// Version can be set to 9 (to have space around) for a new group.
}

// priority defines group priority that is used in discovery. This controls
// group position in the kubectl output.
type priority struct {
	// group indicates the order of the group relative to other groups.
	group int32
	// version indicates the relative order of the version inside of its group.
	version int32
}
