package filter

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"

	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apiserver/filter/metrics"
)

type quotaTracker struct {
	lock                   *sync.Mutex
	remainingReservedQuota map[string]int
	remainingSharedQuota   map[string]int

	reservedQuotaListener map[string][]chan<- func()
}

func (c *quotaTracker) Run(ctx context.Context) {
	if glog.V(5) {
		t := time.NewTicker(time.Second)
		for {
			<-t.C
			func() {
				c.lock.Lock()
				defer c.lock.Unlock()
				glog.Infof("=================\n")
				for n, i := range c.remainingReservedQuota {
					glog.Infof("bucket %v remaining reserved quota %v", n, i)
				}
				for n, q := range c.reservedQuotaListener {
					glog.Infof("bucket %v pending listener %v", n, len(q))
				}
				for n, i := range c.remainingSharedQuota {
					glog.Infof("priority %v remaining shared quota %v", n, i)
				}
			}()
		}
	}
}

func (c *quotaTracker) SyncBucket(old, new *cluster.Bucket) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if old != nil {
		c.remainingReservedQuota[old.Name] -= old.Spec.ReservedQuota
		c.remainingSharedQuota[string(old.Spec.Priority)] -= old.Spec.SharedQuota
	}
	if new != nil {
		c.remainingReservedQuota[new.Name] += new.Spec.ReservedQuota
		c.remainingSharedQuota[string(new.Spec.Priority)] += new.Spec.SharedQuota
	}
}

func (c *quotaTracker) GetReservedQuota(bkt *cluster.Bucket) (quotaReleaseFunc func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	bktName := bkt.Name

	metrics.MonitorRemainingReservedQuota("reserved", bktName, bkt.Spec.Priority, c.remainingReservedQuota[bktName])

	if c.remainingReservedQuota[bktName] > 0 {
		c.remainingReservedQuota[bktName]--
		metrics.MonitorAcquireReservedQuota("reserved", bktName, bkt.Spec.Priority)
		quotaReleaseFunc = func() {
			c.lock.Lock()
			defer c.lock.Unlock()
			// notify
			if len(c.reservedQuotaListener[bktName]) > 0 {
				var distributionCh chan<- func()
				distributionCh, c.reservedQuotaListener[bktName] = c.reservedQuotaListener[bktName][0], c.reservedQuotaListener[bktName][1:]
				go func() {
					distributionCh <- func() {
						quotaReleaseFunc()
					}
				}()
			} else {
				c.remainingReservedQuota[bktName]++
				metrics.MonitorReleaseReservedQuota("reserved", bktName, bkt.Spec.Priority)
			}
		}
		return quotaReleaseFunc
	}
	return nil
}

func (c *quotaTracker) ListenReservedQuota(bktName string, distributionCh chan<- func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.reservedQuotaListener[bktName] = append(c.reservedQuotaListener[bktName], distributionCh)
}

func (c *quotaTracker) GetSharedQuota(bkt *cluster.Bucket) (quotaReleaseFunc func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	matched := false
	for i := 0; i < len(cluster.AllPriorities); i++ {
		metrics.MonitorRemainingReservedQuota("shared", "", cluster.AllPriorities[i], c.remainingSharedQuota[string(cluster.AllPriorities[i])] )
		if cluster.AllPriorities[i] != bkt.Spec.Priority && !matched {
			continue
		}
		matched = true

		if c.remainingSharedQuota[string(cluster.AllPriorities[i])] > 0 {
			c.remainingSharedQuota[string(cluster.AllPriorities[i])]--
			metrics.MonitorAcquireReservedQuota("shared", "", bkt.Spec.Priority)
			return func() {
				c.lock.Lock()
				defer c.lock.Unlock()
				c.remainingSharedQuota[string(cluster.AllPriorities[i])]++
				metrics.MonitorReleaseReservedQuota("shared", "", bkt.Spec.Priority)
			}
		}
	}
	return nil
}
