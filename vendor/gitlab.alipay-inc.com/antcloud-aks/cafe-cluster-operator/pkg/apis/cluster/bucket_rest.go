package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/kubernetes-incubator/apiserver-builder-alpha/pkg/builders"
	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apimachinery/pkg/api/errors"
	"github.com/golang/glog"
)

// +k8s:deepcopy-gen=false
type BucketREST struct {
	*genericregistry.Store
}

func NewBucketREST(getter generic.RESTOptionsGetter) rest.Storage {
	groupResource := schema.GroupResource{
		Group:    "cluster",
		Resource: "buckets",
	}
	strategy := &BucketStrategy{builders.StorageStrategySingleton}
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &Bucket{} },
		NewListFunc:              func() runtime.Object { return &BucketList{} },
		DefaultQualifiedResource: groupResource,

		CreateStrategy: strategy, // TODO: specify create strategy
		UpdateStrategy: strategy, // TODO: specify update strategy
		DeleteStrategy: strategy, // TODO: specify delete strategy
	}
	options := &generic.StoreOptions{RESTOptions: getter}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}
	bktREST := &BucketREST{store}
	bktREST.ensureSystemBuckets()
	return bktREST
}

func (r *BucketREST) ensureSystemBuckets() error {
	for _, bkt := range systemBuckets {
		_, err := r.Store.Create(context.TODO(), bkt, nil, &metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			glog.Errorf("ensure buckets failed: %v", err)
			return err
		}
	}
	return nil
}
