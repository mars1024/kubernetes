package filter

import (
	"testing"
	"time"
	"context"

	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/informers_generated/externalversions"
	"github.com/stretchr/testify/assert"
)

func TestBucketReload(t *testing.T) {

	fakeClient := fake.NewSimpleClientset(
		&cluster.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-bkt-1",
			},
			Spec: cluster.BucketSpec{
				Priority: cluster.SystemHighPriorityBand,
			},
		},
		&cluster.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-bkt-2",
			},
			Spec: cluster.BucketSpec{
				Priority: cluster.SystemMediumPriorityBand,
			},
		},
		&cluster.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-bkt-3",
			},
			Spec: cluster.BucketSpec{
				Priority: cluster.SystemTopPriorityBand,
			},
		},
		&cluster.BucketBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-bkt-binding-1",
			},
			Spec: cluster.BucketBindingSpec{
				BucketRef: &cluster.BucketReference{
					Name: "foo-bkt-1",
				},
			},
		},
		&cluster.BucketBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-bkt-binding-2",
			},
			Spec: cluster.BucketBindingSpec{
				BucketRef: &cluster.BucketReference{
					Name: "foo-bkt-2",
				},
			},
		},
	)
	informer := externalversions.NewSharedInformerFactory(fakeClient, 0)
	stopCh := make(chan struct{})
	defer close(stopCh)
	bktInformer := informer.Cluster().V1alpha1().Buckets().Informer()
	bktBindingInformer := informer.Cluster().V1alpha1().BucketBindings().Informer()
	assert.NotNil(t, bktInformer)
	assert.NotNil(t, bktBindingInformer)

	ctx := context.TODO()
	filter := NewInFlightFilter(nil, informer, fakeClient)
	filter.Run(ctx)

	time.Sleep(time.Second)
	if bindings := filter.bucketBindings.Load().([]*cluster.BucketBinding); assert.Equal(t, 2, len(bindings)) {
		assert.Equal(t, "foo-bkt-binding-1", bindings[0].Name)
		assert.Equal(t, "foo-bkt-binding-2", bindings[1].Name)
	}
	fakeClient.Cluster().BucketBindings().Create(&cluster.BucketBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo-bkt-binding-3",
		},
		Spec: cluster.BucketBindingSpec{
			BucketRef: &cluster.BucketReference{
				Name: "foo-bkt-3",
			},
		},
	})
	fakeClient.Cluster().BucketBindings().Create(&cluster.BucketBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo-bkt-binding-4",
		},
		Spec: cluster.BucketBindingSpec{
			Rules: []*cluster.BucketBindingRule{
				{},
				{},
			},
			BucketRef: &cluster.BucketReference{
				Name: "foo-bkt-3",
			},
		},
	})
	time.Sleep(time.Second)
	if bindings := filter.bucketBindings.Load().([]*cluster.BucketBinding); assert.Equal(t, 4, len(bindings)) {
		assert.Equal(t, "foo-bkt-binding-4", bindings[0].Name)
		assert.Equal(t, "foo-bkt-binding-3", bindings[1].Name)
		assert.Equal(t, "foo-bkt-binding-1", bindings[2].Name)
		assert.Equal(t, "foo-bkt-binding-2", bindings[3].Name)
	}

}
