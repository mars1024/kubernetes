/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package cluster

import (
	"context"
	"fmt"

	"github.com/kubernetes-incubator/apiserver-builder-alpha/pkg/builders"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	InternalBucket = builders.NewInternalResource(
		"buckets",
		"Bucket",
		func() runtime.Object { return &Bucket{} },
		func() runtime.Object { return &BucketList{} },
	)
	InternalBucketStatus = builders.NewInternalResourceStatus(
		"buckets",
		"BucketStatus",
		func() runtime.Object { return &Bucket{} },
		func() runtime.Object { return &BucketList{} },
	)
	InternalBucketBinding = builders.NewInternalResource(
		"bucketbindings",
		"BucketBinding",
		func() runtime.Object { return &BucketBinding{} },
		func() runtime.Object { return &BucketBindingList{} },
	)
	InternalBucketBindingStatus = builders.NewInternalResourceStatus(
		"bucketbindings",
		"BucketBindingStatus",
		func() runtime.Object { return &BucketBinding{} },
		func() runtime.Object { return &BucketBindingList{} },
	)
	InternalClusterResourceQuota = builders.NewInternalResource(
		"clusterresourcequotas",
		"ClusterResourceQuota",
		func() runtime.Object { return &ClusterResourceQuota{} },
		func() runtime.Object { return &ClusterResourceQuotaList{} },
	)
	InternalClusterResourceQuotaStatus = builders.NewInternalResourceStatus(
		"clusterresourcequotas",
		"ClusterResourceQuotaStatus",
		func() runtime.Object { return &ClusterResourceQuota{} },
		func() runtime.Object { return &ClusterResourceQuotaList{} },
	)
	// Registered resources and subresources
	ApiVersion = builders.NewApiGroup("cluster.aks.cafe.sofastack.io").WithKinds(
		InternalBucket,
		InternalBucketStatus,
		InternalBucketBinding,
		InternalBucketBindingStatus,
		InternalClusterResourceQuota,
		InternalClusterResourceQuotaStatus,
	)

	// Required by code generated by go2idl
	AddToScheme        = ApiVersion.SchemaBuilder.AddToScheme
	SchemeBuilder      = ApiVersion.SchemaBuilder
	localSchemeBuilder = &SchemeBuilder
	SchemeGroupVersion = ApiVersion.GroupVersion
)

// Required by code generated by go2idl
// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Required by code generated by go2idl
// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

type PriorityBand string

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Bucket struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   BucketSpec
	Status BucketStatus
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterResourceQuota struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   ClusterResourceQuotaSpec
	Status ClusterResourceQuotaStatus
}

type BucketStatus struct {
	Phase string
}

type ClusterResourceQuotaStatus struct {
	Total      corev1.ResourceQuotaStatus
	Namespaces []NamespaceResourceQuotaStatus
}

type ClusterResourceQuotaSpec struct {
	Selector ClusterResourceQuotaSelector
	Quota    corev1.ResourceQuotaSpec
}

type NamespaceResourceQuotaStatus struct {
	Name   string
	Status corev1.ResourceQuotaStatus
}

type ClusterResourceQuotaSelector struct {
	AnnotationSelector map[string]string
}

type BucketSpec struct {
	ReservedQuota int
	SharedQuota   int
	Priority      PriorityBand
	Weight        int
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BucketBinding struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   BucketBindingSpec
	Status BucketBindingStatus
}

type BucketBindingSpec struct {
	Rules     []*BucketBindingRule
	BucketRef *BucketReference
}

type BucketBindingStatus struct {
	Phase string
}

type BucketReference struct {
	Name string
}

type BucketBindingRule struct {
	Field  string
	Values []string
}

//
// Bucket Functions and Structs
//
// +k8s:deepcopy-gen=false
type BucketStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type BucketStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BucketList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Bucket
}

func (Bucket) NewStatus() interface{} {
	return BucketStatus{}
}

func (pc *Bucket) GetStatus() interface{} {
	return pc.Status
}

func (pc *Bucket) SetStatus(s interface{}) {
	pc.Status = s.(BucketStatus)
}

func (pc *Bucket) GetSpec() interface{} {
	return pc.Spec
}

func (pc *Bucket) SetSpec(s interface{}) {
	pc.Spec = s.(BucketSpec)
}

func (pc *Bucket) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *Bucket) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc Bucket) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store Bucket.
// +k8s:deepcopy-gen=false
type BucketRegistry interface {
	ListBuckets(ctx context.Context, options *internalversion.ListOptions) (*BucketList, error)
	GetBucket(ctx context.Context, id string, options *metav1.GetOptions) (*Bucket, error)
	CreateBucket(ctx context.Context, id *Bucket) (*Bucket, error)
	UpdateBucket(ctx context.Context, id *Bucket) (*Bucket, error)
	DeleteBucket(ctx context.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewBucketRegistry(sp builders.StandardStorageProvider) BucketRegistry {
	return &storageBucket{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageBucket struct {
	builders.StandardStorageProvider
}

func (s *storageBucket) ListBuckets(ctx context.Context, options *internalversion.ListOptions) (*BucketList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*BucketList), err
}

func (s *storageBucket) GetBucket(ctx context.Context, id string, options *metav1.GetOptions) (*Bucket, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*Bucket), nil
}

func (s *storageBucket) CreateBucket(ctx context.Context, object *Bucket) (*Bucket, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*Bucket), nil
}

func (s *storageBucket) UpdateBucket(ctx context.Context, object *Bucket) (*Bucket, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*Bucket), nil
}

func (s *storageBucket) DeleteBucket(ctx context.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, &metav1.DeleteOptions{})
	return sync, err
}

//
// BucketBinding Functions and Structs
//
// +k8s:deepcopy-gen=false
type BucketBindingStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type BucketBindingStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BucketBindingList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []BucketBinding
}

func (BucketBinding) NewStatus() interface{} {
	return BucketBindingStatus{}
}

func (pc *BucketBinding) GetStatus() interface{} {
	return pc.Status
}

func (pc *BucketBinding) SetStatus(s interface{}) {
	pc.Status = s.(BucketBindingStatus)
}

func (pc *BucketBinding) GetSpec() interface{} {
	return pc.Spec
}

func (pc *BucketBinding) SetSpec(s interface{}) {
	pc.Spec = s.(BucketBindingSpec)
}

func (pc *BucketBinding) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *BucketBinding) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc BucketBinding) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store BucketBinding.
// +k8s:deepcopy-gen=false
type BucketBindingRegistry interface {
	ListBucketBindings(ctx context.Context, options *internalversion.ListOptions) (*BucketBindingList, error)
	GetBucketBinding(ctx context.Context, id string, options *metav1.GetOptions) (*BucketBinding, error)
	CreateBucketBinding(ctx context.Context, id *BucketBinding) (*BucketBinding, error)
	UpdateBucketBinding(ctx context.Context, id *BucketBinding) (*BucketBinding, error)
	DeleteBucketBinding(ctx context.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewBucketBindingRegistry(sp builders.StandardStorageProvider) BucketBindingRegistry {
	return &storageBucketBinding{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageBucketBinding struct {
	builders.StandardStorageProvider
}

func (s *storageBucketBinding) ListBucketBindings(ctx context.Context, options *internalversion.ListOptions) (*BucketBindingList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*BucketBindingList), err
}

func (s *storageBucketBinding) GetBucketBinding(ctx context.Context, id string, options *metav1.GetOptions) (*BucketBinding, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*BucketBinding), nil
}

func (s *storageBucketBinding) CreateBucketBinding(ctx context.Context, object *BucketBinding) (*BucketBinding, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*BucketBinding), nil
}

func (s *storageBucketBinding) UpdateBucketBinding(ctx context.Context, object *BucketBinding) (*BucketBinding, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*BucketBinding), nil
}

func (s *storageBucketBinding) DeleteBucketBinding(ctx context.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, &metav1.DeleteOptions{})
	return sync, err
}

//
// ClusterResourceQuota Functions and Structs
//
// +k8s:deepcopy-gen=false
type ClusterResourceQuotaStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type ClusterResourceQuotaStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterResourceQuotaList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []ClusterResourceQuota
}

func (ClusterResourceQuota) NewStatus() interface{} {
	return ClusterResourceQuotaStatus{}
}

func (pc *ClusterResourceQuota) GetStatus() interface{} {
	return pc.Status
}

func (pc *ClusterResourceQuota) SetStatus(s interface{}) {
	pc.Status = s.(ClusterResourceQuotaStatus)
}

func (pc *ClusterResourceQuota) GetSpec() interface{} {
	return pc.Spec
}

func (pc *ClusterResourceQuota) SetSpec(s interface{}) {
	pc.Spec = s.(ClusterResourceQuotaSpec)
}

func (pc *ClusterResourceQuota) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *ClusterResourceQuota) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc ClusterResourceQuota) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store ClusterResourceQuota.
// +k8s:deepcopy-gen=false
type ClusterResourceQuotaRegistry interface {
	ListClusterResourceQuotas(ctx context.Context, options *internalversion.ListOptions) (*ClusterResourceQuotaList, error)
	GetClusterResourceQuota(ctx context.Context, id string, options *metav1.GetOptions) (*ClusterResourceQuota, error)
	CreateClusterResourceQuota(ctx context.Context, id *ClusterResourceQuota) (*ClusterResourceQuota, error)
	UpdateClusterResourceQuota(ctx context.Context, id *ClusterResourceQuota) (*ClusterResourceQuota, error)
	DeleteClusterResourceQuota(ctx context.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewClusterResourceQuotaRegistry(sp builders.StandardStorageProvider) ClusterResourceQuotaRegistry {
	return &storageClusterResourceQuota{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageClusterResourceQuota struct {
	builders.StandardStorageProvider
}

func (s *storageClusterResourceQuota) ListClusterResourceQuotas(ctx context.Context, options *internalversion.ListOptions) (*ClusterResourceQuotaList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*ClusterResourceQuotaList), err
}

func (s *storageClusterResourceQuota) GetClusterResourceQuota(ctx context.Context, id string, options *metav1.GetOptions) (*ClusterResourceQuota, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*ClusterResourceQuota), nil
}

func (s *storageClusterResourceQuota) CreateClusterResourceQuota(ctx context.Context, object *ClusterResourceQuota) (*ClusterResourceQuota, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*ClusterResourceQuota), nil
}

func (s *storageClusterResourceQuota) UpdateClusterResourceQuota(ctx context.Context, object *ClusterResourceQuota) (*ClusterResourceQuota, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*ClusterResourceQuota), nil
}

func (s *storageClusterResourceQuota) DeleteClusterResourceQuota(ctx context.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, &metav1.DeleteOptions{})
	return sync, err
}
