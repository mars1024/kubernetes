/*
Copyright 2018 The Alipay.com Inc Authors.

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

package clusterresourcequota

import (
	"errors"
	"io"
	"sync"
	"sort"
	"time"

	apiserveradmission "k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/admission"
	kcorev1Lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	listers "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	cafeinformers "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/informers_generated/externalversions"
	cafeadmission "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/plugin/admission"
	clustertypedclient "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota/generic"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota/evaluator/core"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/controller/clusterquotamapping"
)

const (
	PluginName             = "ClusterResourceQuota"
	timeToWaitForCacheSync = 10 * time.Second
	numEvaluatorThreads    = 10

	AnnotationClusterName = "aks.cafe.sofastack.io/mc"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewClusterResourceQuota()
	})
}

var _ apiserveradmission.WantsExternalKubeInformerFactory = &clusterResourceQuotaAdmission{}
var _ cafeadmission.WantsCafeClusterOperatorKubeInformerFactory = &clusterResourceQuotaAdmission{}
var _ cafeadmission.WantsRESTClientConfig = &clusterResourceQuotaAdmission{}
var _ admission.ValidationInterface = &clusterResourceQuotaAdmission{}

// clusterResourceQuotaAdmission implements an admission controller that can enforce clusterResourceQuota constraints
type clusterResourceQuotaAdmission struct {
	*admission.Handler

	clusterResourceQuotaLister listers.ClusterResourceQuotaLister
	namespaceLister            kcorev1Lister.NamespaceLister
	clusterResourceQuotaSynced func() bool
	namespaceSynced            func() bool
	clusterResourceQuotaClient clustertypedclient.ClusterResourceQuotasGetter
	clusterQuotaMapper         clusterquotamapping.ClusterQuotaMapper

	lockFactory LockFactory

	// these are used to create the evaluator
	registry quota.Registry

	init      sync.Once
	evaluator Evaluator
}

func (q *clusterResourceQuotaAdmission) SetExternalKubeInformerFactory(f informers.SharedInformerFactory) {
	q.namespaceLister = f.Core().V1().Namespaces().Lister()
	q.namespaceSynced = f.Core().V1().Namespaces().Informer().HasSynced
}

func (q *clusterResourceQuotaAdmission) SetCafeClusterOperatorKubeInformerFactory(informer cafeinformers.SharedInformerFactory) {
	q.clusterResourceQuotaLister = informer.Cluster().V1alpha1().ClusterResourceQuotas().Lister()
	q.clusterResourceQuotaSynced = informer.Cluster().V1alpha1().ClusterResourceQuotas().Informer().HasSynced
	informer.Start(make(chan struct{}))
}

func (q *clusterResourceQuotaAdmission) SetRESTClientConfig(restClientConfig rest.Config) {
	var err error

	// ClusterResourceQuota is served using CRD resource any status update must use JSON
	jsonClientConfig := rest.CopyConfig(&restClientConfig)
	jsonClientConfig.ContentConfig.AcceptContentTypes = "application/json"
	jsonClientConfig.ContentConfig.ContentType = "application/json"

	q.clusterResourceQuotaClient = clustertypedclient.NewForConfigOrDie(jsonClientConfig)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
}

func (q *clusterResourceQuotaAdmission) ValidateInitialization() error {
	if q.clusterResourceQuotaLister == nil {
		return errors.New("missing clusterResourceQuotaLister")
	}
	if q.namespaceLister == nil {
		return errors.New("missing namespaceLister")
	}
	if q.clusterResourceQuotaClient == nil {
		return errors.New("missing clusterQuotaResourceClient")
	}
	if q.registry == nil {
		return errors.New("missing registry")
	}

	return nil
}

// NewClusterResourceQuota configures an admission controller that can enforce clusterResourceQuota constraints
// using the provided registry.  The registry must have the capability to handle group/kinds that are persisted
// by the server this admission controller is intercepting
func NewClusterResourceQuota() (admission.Interface, error) {
	quotaRegistry := generic.NewRegistry(core.NewEvaluators(nil))
	return &clusterResourceQuotaAdmission{
		Handler:     admission.NewHandler(admission.Create, admission.Update),
		lockFactory: NewDefaultLockFactory(),
		registry:    quotaRegistry,
	}, nil
}

// Admit makes decisions while enforcing clusterResourceQuota
func (q *clusterResourceQuotaAdmission) Validate(a admission.Attributes) (err error) {
	// ignore all operations that correspond to sub-resource actions
	if len(a.GetSubresource()) != 0 {
		return nil
	}
	// ignore cluster level resources
	if len(a.GetNamespace()) == 0 {
		return nil
	}

	objMeta, err := meta.Accessor(a.GetObject())
	if err != nil {
		// if we don't have objectmeta, just ignore this object
		return nil
	}
	clusterName := getObjectClusterName(objMeta)
	if len(clusterName) == 0 {
		return nil
	}

	if !q.waitForSyncedStore(time.After(timeToWaitForCacheSync)) {
		return admission.NewForbidden(a, errors.New("caches not synchronized"))
	}

	q.init.Do(func() {
		clusterQuotaAccessor := newQuotaAccessor(q.clusterResourceQuotaLister, q.clusterResourceQuotaClient)
		q.evaluator = NewQuotaEvaluator(clusterQuotaAccessor, ignoredResources, q.registry, q.lockAquisition, numEvaluatorThreads, wait.NeverStop)
	})

	return q.evaluator.Evaluate(a, clusterName)
}

func (q *clusterResourceQuotaAdmission) lockAquisition(quotas []corev1.ResourceQuota) func() {
	var locks []sync.Locker

	// acquire the locks in alphabetical order because I'm too lazy to think of something clever
	sort.Sort(ByName(quotas))
	for _, rq := range quotas {
		lock := q.lockFactory.GetLock(rq.Name)
		lock.Lock()
		locks = append(locks, lock)
	}

	return func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].Unlock()
		}
	}
}

func (q *clusterResourceQuotaAdmission) waitForSyncedStore(timeout <-chan time.Time) bool {
	for !q.clusterResourceQuotaSynced() || !q.namespaceSynced() {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-timeout:
			return q.clusterResourceQuotaSynced() && q.namespaceSynced()
		}
	}

	return true
}

func getObjectClusterName(obj v1.Object) string {
	clusterName, ok := obj.GetAnnotations()[AnnotationClusterName]
	if !ok {
		return ""
	}
	return clusterName
}

type ByName []corev1.ResourceQuota

func (v ByName) Len() int           { return len(v) }
func (v ByName) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v ByName) Less(i, j int) bool { return v[i].Name < v[j].Name }

// ignoredResources is the set of resources that clusterquota ignores.  It's larger because we have to ignore requests
// that the namespace lifecycle plugin ignores.  This is because of the need to have a matching namespace in order to be sure
// that the cache is current enough to have mapped the CRQ to the namespaces.  Normal RQ doesn't have that requirement.
var ignoredResources = map[schema.GroupResource]struct{}{
	{Group: "", Resource: "events"}: {},
}
