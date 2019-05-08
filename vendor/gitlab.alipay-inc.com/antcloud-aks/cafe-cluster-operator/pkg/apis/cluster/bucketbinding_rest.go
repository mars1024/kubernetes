package cluster

import (
	"context"

	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/kubernetes-incubator/apiserver-builder-alpha/pkg/builders"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/golang/glog"
)

// +k8s:deepcopy-gen=false
type BucketBindingREST struct {
	*genericregistry.Store
}

func NewBucketBindingREST(getter generic.RESTOptionsGetter) rest.Storage {
	groupResource := schema.GroupResource{
		Group:    "cluster",
		Resource: "bucketbindings",
	}
	strategy := &BucketBindingStrategy{builders.StorageStrategySingleton}
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &BucketBinding{} },
		NewListFunc:              func() runtime.Object { return &BucketBindingList{} },
		DefaultQualifiedResource: groupResource,

		CreateStrategy: strategy, // TODO: specify create strategy
		UpdateStrategy: strategy, // TODO: specify update strategy
		DeleteStrategy: strategy, // TODO: specify delete strategy
	}

	options := &generic.StoreOptions{RESTOptions: getter}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}
	bktBindingREST := &BucketBindingREST{store}
	bktBindingREST.ensureSystemBucketBindings()
	return bktBindingREST
}

func (r *BucketBindingREST) ensureSystemBucketBindings() error {
	for _, bktBinding := range systemBucketBindings {
		_, err := r.Store.Create(context.TODO(), bktBinding, nil, &metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			glog.Errorf("ensure bucketbindings failed: %v", err)
			return err
		}
	}
	return nil
}
