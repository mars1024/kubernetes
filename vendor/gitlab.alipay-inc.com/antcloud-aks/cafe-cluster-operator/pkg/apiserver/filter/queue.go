package filter

import (
	"sync"
	"math/rand"
	clusterlisters "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/golang/glog"
)

type WRRQueue struct {
	lock *sync.Mutex

	weights map[string]float64
	queues  map[string][]interface{}
	reload  func()
}

func NewWRRQueueForBucketLister(priority cluster.PriorityBand, bucketLister clusterlisters.BucketLister) *WRRQueue {
	q := &WRRQueue{
		lock:    &sync.Mutex{},
		weights: make(map[string]float64),
		queues:  make(map[string][]interface{}),
	}
	q.reload = func() {
		q.lock.Lock()
		defer q.lock.Unlock()
		bkts, _ := bucketLister.List(labels.Everything())
		q.weights = make(map[string]float64)
		glog.V(7).Infof("reloading buckets...")
		for _, bkt := range bkts {
			if bkt.Spec.Priority != priority {
				continue
			}
			q.weights[bkt.Name] = float64(bkt.Spec.Weight)
			glog.V(7).Infof("reloading buckets %v weight %v...", bkt.Name, bkt.Spec.Weight)
		}
	}
	q.reload()
	return q
}

func NewWRRQueueForBuckets(bkts []*cluster.Bucket) *WRRQueue {
	q := &WRRQueue{
		lock:    &sync.Mutex{},
		weights: make(map[string]float64),
		queues:  make(map[string][]interface{}),
	}
	q.reload = func() {
		q.lock.Lock()
		defer q.lock.Unlock()
		q.weights = make(map[string]float64)
		for _, bkt := range bkts {
			q.weights[bkt.Name] = float64(bkt.Spec.Weight)
		}
	}
	q.reload()
	return q
}

func (q *WRRQueue) enqueue(queueIdentifier string, item interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Distributing
	q.queues[queueIdentifier] = append(q.queues[queueIdentifier], item)
}

func (q *WRRQueue) dequeue() (queueIdentifier string, item interface{}) {
	// TODO: optimize time cost to Log(N) here by applying segment tree algorithm
	q.lock.Lock()
	defer q.lock.Unlock()

	var totalWeight float64
	for id, w := range q.weights {
		if len(q.queues[id]) == 0 {
			continue
		}
		totalWeight += w
	}

	randomPtr := rand.Float64() * totalWeight
	var distributionPtr float64
	for id, w := range q.weights {
		if len(q.queues[id]) == 0 {
			continue
		}
		distributionPtr += w
		if randomPtr <= distributionPtr {
			item, q.queues[id] = q.queues[id][0], q.queues[id][1:]
			return id, item
		}
	}

	// drains all-the-rest
	for id := range q.queues {
		if len(q.queues[id]) > 0 {
			item, q.queues[id] = q.queues[id][0], q.queues[id][1:]
			return id, item
		}
	}

	return "", nil
}
