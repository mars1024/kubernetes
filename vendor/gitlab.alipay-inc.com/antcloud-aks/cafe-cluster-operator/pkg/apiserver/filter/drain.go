package filter

import (
	"sync"
	"context"

	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	clusterlisters "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	"time"
)

type queueDrainer struct {
	tracker      *quotaTracker
	bucketLister clusterlisters.BucketLister

	lock *sync.Mutex

	queueLength    int
	maxQueueLength int

	queueByPriorities map[cluster.PriorityBand]*WRRQueue
}

func (d *queueDrainer) reloadBuckets() {
	for i := 0; i < len(cluster.AllPriorities); i++ {
		d.queueByPriorities[cluster.AllPriorities[i]].reload()
	}
}

func (d *queueDrainer) Dequeue() (string, chan<- func()) {
	for i := 0; i < len(cluster.AllPriorities); i++ {
		if id, item := d.queueByPriorities[cluster.AllPriorities[i]].dequeue(); item != nil {
			d.lock.Lock()
			defer d.lock.Unlock()
			d.queueLength--
			return id, item.(chan<- func())
		}
	}
	return "", nil
}

func (d *queueDrainer) Run(ctx context.Context) {
	for {
		if d.queueLength == 0 {
			time.Sleep(20 * time.Millisecond)
		}
		func() {
			bktName, distributionCh := d.Dequeue()
			if distributionCh == nil {
				return
			}
			bkt, err := d.bucketLister.Get(bktName)
			if err != nil {
				// evict the queue
				d.tracker.ListenReservedQuota(bktName, distributionCh)
				return
			}
			if releaseFunc := d.tracker.GetReservedQuota(bkt); releaseFunc != nil {
				go func() {
					distributionCh <- releaseFunc
				}()
				return
			}

			if bkt.Spec.Weight == 0 {
				// requeue
				d.tracker.ListenReservedQuota(bktName, distributionCh)
				return
			}

			if releaseFunc := d.tracker.GetSharedQuota(bkt); releaseFunc != nil {
				go func() {
					distributionCh <- releaseFunc
				}()
				return
			}
			d.Requeue(bkt, distributionCh)
		}()
	}
}

func (d *queueDrainer) Enqueue(bkt *cluster.Bucket) <-chan func() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.queueLength > d.maxQueueLength {
		return nil
	}
	// Prioritizing
	d.queueLength++
	distributionCh := make(chan func(), 1)
	var receivingDistributionCh chan<- func() = distributionCh
	d.queueByPriorities[bkt.Spec.Priority].enqueue(bkt.Name, receivingDistributionCh)
	return distributionCh
}

func (d *queueDrainer) Requeue(bkt *cluster.Bucket, distributionCh chan<- func()) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.queueLength++
	d.queueByPriorities[bkt.Spec.Priority].enqueue(bkt.Name, distributionCh)
}
