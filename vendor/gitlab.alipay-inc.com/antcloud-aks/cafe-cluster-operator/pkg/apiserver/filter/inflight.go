package filter

import (
	"net/http"
	"time"
	"sync"
	"context"
	"sort"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/informers_generated/externalversions"
	clusterlisters "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"sync/atomic"
	"k8s.io/client-go/tools/cache"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"k8s.io/client-go/rest"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apiserver/filter/metrics"
)

type inFlightFilter struct {
	factory             externalversions.SharedInformerFactory
	*queueDrainer
	bucketBindings      *atomic.Value
	bucketBindingLister clusterlisters.BucketBindingLister
	delegateHandler     http.Handler
}

func init() {
	// HACK: override extra bucket's quota
	if quotaStr := os.Getenv("FEATURE_RATE_LIMIT_EXTRA_QUOTA"); len(quotaStr) > 0 {
		if quota, err := strconv.Atoi(quotaStr); err == nil {
			extraBucket.Spec.SharedQuota = quota
		}
	}
}

const (
	maxTimeout     = time.Minute
	maxQueueLength = 1000
)

var extraBucket = &cluster.Bucket{
	ObjectMeta: metav1.ObjectMeta{
		Name: "extra",
	},
	Spec: cluster.BucketSpec{
		SharedQuota: 100,
		Priority:    cluster.SystemLowestPriorityBand,
		Weight:      1,
	},
}

func NewInFlightFilterWithRestConfig(delegateHandler http.Handler, cfg *rest.Config) *inFlightFilter {
	clientCreated.Do(func() {
		httpLoopbackConfig := rest.CopyConfig(cfg)
		httpLoopbackConfig.ContentType = "application/json"
		client = clientset.NewForConfigOrDie(httpLoopbackConfig)
		factory = externalversions.NewSharedInformerFactory(client, 0)
	})
	return NewInFlightFilter(delegateHandler, factory, client)
}

var (
	tracker        *quotaTracker
	client         clientset.Interface
	factory        externalversions.SharedInformerFactory
	trackerCreated sync.Once
	clientCreated  sync.Once
)

func NewInFlightFilter(delegateHandler http.Handler, factory externalversions.SharedInformerFactory, client clientset.Interface) *inFlightFilter {
	metrics.Register()

	factory.Cluster().V1alpha1().Buckets().Informer().GetIndexer().Add(extraBucket)

	trackerCreated.Do(func() {
		tracker = &quotaTracker{
			lock:                   &sync.Mutex{},
			remainingReservedQuota: make(map[string]int),
			remainingSharedQuota:   make(map[string]int),
			reservedQuotaListener:  make(map[string][]chan<- func()),
		}
		tracker.SyncBucket(nil, extraBucket)

		factory.Cluster().V1alpha1().Buckets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				tracker.SyncBucket(nil, obj.(*cluster.Bucket))
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				tracker.SyncBucket(oldObj.(*cluster.Bucket), newObj.(*cluster.Bucket))
			},
			DeleteFunc: func(obj interface{}) {
				if s, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					if s.Obj == extraBucket {
						factory.Cluster().V1alpha1().Buckets().Informer().GetIndexer().Add(extraBucket)
						return
					}
					obj = s.Obj
				}
				tracker.SyncBucket(obj.(*cluster.Bucket), nil)
			},
		})
	})

	drainer := &queueDrainer{
		tracker:      tracker,
		bucketLister: factory.Cluster().V1alpha1().Buckets().Lister(),

		lock:              &sync.Mutex{},
		maxQueueLength:    maxQueueLength,
		queueByPriorities: make(map[cluster.PriorityBand]*WRRQueue),
	}

	for i := 0; i < len(cluster.AllPriorities); i++ {
		drainer.queueByPriorities[cluster.AllPriorities[i]] = NewWRRQueueForBucketLister(
			cluster.AllPriorities[i], factory.Cluster().V1alpha1().Buckets().Lister())
	}
	bindings := &atomic.Value{}
	bindings.Store([]*cluster.BucketBinding(nil))
	instance := &inFlightFilter{
		factory:             factory,
		bucketBindings:      bindings,
		bucketBindingLister: factory.Cluster().V1alpha1().BucketBindings().Lister(),
		queueDrainer:        drainer,
		delegateHandler:     delegateHandler,
	}

	factory.Cluster().V1alpha1().BucketBindings().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			instance.reloadBindings()
		},
		UpdateFunc: func(_, _ interface{}) {
			instance.reloadBindings()
		},
		DeleteFunc: func(_ interface{}) {
			instance.reloadBindings()
		},
	})

	factory.Cluster().V1alpha1().Buckets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			instance.queueDrainer.reloadBuckets()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			instance.queueDrainer.reloadBuckets()
		},
		DeleteFunc: func(obj interface{}) {
			instance.queueDrainer.reloadBuckets()
		},
	})

	return instance
}

func (f *inFlightFilter) reloadBindings() {
	bindings, err := f.bucketBindingLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	copiedBindings := make([]*cluster.BucketBinding, len(bindings))
	copy(copiedBindings, bindings)
	sort.Slice(copiedBindings, func(a, b int) bool {
		bucketA, err := f.bucketLister.Get(copiedBindings[a].Spec.BucketRef.Name)
		if err != nil {
			return false
		}
		bucketB, err := f.bucketLister.Get(copiedBindings[b].Spec.BucketRef.Name)
		if err != nil {
			return true
		}
		cmp := cmpPriority(bucketA.Spec.Priority, bucketB.Spec.Priority)
		if cmp == 0 {
			return len(copiedBindings[a].Spec.Rules) > len(copiedBindings[b].Spec.Rules)
		}
		return cmp < 0
	})
	f.bucketBindings.Store(copiedBindings)
	glog.V(7).Infof("reloading bucket bindings %#v", copiedBindings)
}

func cmpPriority(p1 cluster.PriorityBand, p2 cluster.PriorityBand) int {
	var i1, i2 int
	for i, p := range cluster.AllPriorities {
		i := i
		if p == p1 {
			i1 = i
		}
		if p == p2 {
			i2 = i
		}
	}
	return i1 - i2
}

func (f *inFlightFilter) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// 0. Matching request w/ bindings API
	matchedBkt, err := matchBucketBindings(r, f.bucketLister, f.bucketBindings.Load().([]*cluster.BucketBinding))
	if err != nil {
		responsewriters.InternalError(w, r, err)
	}

	// 1. Waiting to be notified by either a reserved quota or a shared quota
	startTime := time.Now()
	defer func() {
		endTime := time.Now()
		w.Header().Set("X-RATE-LIMIT-TIME", endTime.Sub(startTime).String())
	}()
	distributionCh := f.queueDrainer.Enqueue(matchedBkt)
	if distributionCh == nil {
		// too many requests
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("too many queuing inflight requests"))
	}
	ticker := time.NewTicker(maxTimeout)
	defer ticker.Stop()
	select {
	case finishFunc := <-distributionCh:
		glog.V(8).Infof("distributed")
		defer finishFunc()
		f.delegateHandler.ServeHTTP(w, r)
	case <-ticker.C:
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("inflight rate-limit timeout"))
	}
}

func (f *inFlightFilter) Run(ctx context.Context) {
	if f.queueDrainer != nil {
		go f.queueDrainer.Run(ctx)
		go f.queueDrainer.tracker.Run(ctx)
	}
	f.factory.Start(ctx.Done())
}
