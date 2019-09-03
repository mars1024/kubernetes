package filter

import (
	"testing"
	"github.com/stretchr/testify/assert"
	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWRRQueueBasicManipulation(t *testing.T) {
	// enqueue one
	wrrQueue := NewWRRQueueForBuckets([]*cluster.Bucket{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo",
			},
			Spec: cluster.BucketSpec{
				Weight: 1,
			},
		},
	})

	_, dequeueFromEmptyQueue := wrrQueue.dequeue()
	assert.Nil(t, dequeueFromEmptyQueue)

	item1 := "test_item1"
	item2 := "test_item2"
	wrrQueue.enqueue("foo", item1)
	wrrQueue.enqueue("foo", item2)

	_, item := wrrQueue.dequeue()
	assert.Equal(t, item1, item)
	_, item = wrrQueue.dequeue()
	assert.Equal(t, item2, item)
}

func TestWRRQueueZeroWeight(t *testing.T) {
	wrrQueue := NewWRRQueueForBuckets([]*cluster.Bucket{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo",
			},
			Spec: cluster.BucketSpec{
				Weight: 0,
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "bar",
			},
			Spec: cluster.BucketSpec{
				Weight: 1,
			},
		},
	})

	item1 := "test_item1"
	item2 := "test_item2"
	item3 := "test_item3"
	item4 := "test_item4"

	wrrQueue.enqueue("foo", item1)
	wrrQueue.enqueue("foo", item2)
	wrrQueue.enqueue("bar", item3)
	wrrQueue.enqueue("bar", item4)

	_, item := wrrQueue.dequeue()
	assert.Equal(t, item3, item)
	_, item = wrrQueue.dequeue()
	assert.Equal(t, item4, item)
	_, item = wrrQueue.dequeue()
	assert.Equal(t, item1, item)
	_, item = wrrQueue.dequeue()
	assert.Equal(t, item2, item)
}
