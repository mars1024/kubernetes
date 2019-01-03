package filter

import (
	"testing"
	"sync"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestQuotaTracker(t *testing.T) {
	quotaTracker := &quotaTracker{
		lock: &sync.Mutex{},
		remainingReservedQuota: map[string]int{
			"bkt1": 1,
			"bkt2": 1,
		},
		remainingSharedQuota: map[string]int{
			string(cluster.SystemTopPriorityBand):    1,
			string(cluster.SystemLowestPriorityBand): 1,
		},
	}
	bkt1 := &cluster.Bucket{
		ObjectMeta: v1.ObjectMeta{
			Name: "bkt1",
		},
		Spec: cluster.BucketSpec{
			Priority: cluster.SystemTopPriorityBand,
		},
	}
	bkt2 := &cluster.Bucket{
		ObjectMeta: v1.ObjectMeta{
			Name: "bkt2",
		},
		Spec: cluster.BucketSpec{
			Priority: cluster.SystemLowestPriorityBand,
		},
	}
	releaseFunc := quotaTracker.GetReservedQuota(bkt1)
	assert.NotNil(t, releaseFunc)
	releaseFunc()

	releaseFunc = quotaTracker.GetReservedQuota(bkt1)
	assert.NotNil(t, releaseFunc)

	releaseFunc = quotaTracker.GetReservedQuota(bkt2)
	assert.NotNil(t, releaseFunc)
	releaseFunc = quotaTracker.GetReservedQuota(bkt2)
	assert.Nil(t, releaseFunc)

	releaseFunc = quotaTracker.GetSharedQuota(bkt1)
	assert.NotNil(t, releaseFunc)
	releaseFunc = quotaTracker.GetSharedQuota(bkt1)
	assert.NotNil(t, releaseFunc)
	releaseFunc()

	releaseFunc = quotaTracker.GetSharedQuota(bkt2)
	assert.NotNil(t, releaseFunc)
	releaseFunc = quotaTracker.GetSharedQuota(bkt2)
	assert.Nil(t, releaseFunc)
}
