/*
Copyright 2018 The Alipay.com Inc Authors.

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

package generic

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota"
)

// implements a basic registry
type simpleRegistry struct {
	lock sync.RWMutex
	// evaluators tracked by the registry
	evaluators map[schema.GroupResource]quota.GroupResourceEvaluator
}

func NewRegistry(evaluators []quota.GroupResourceEvaluator) quota.Registry {
	return &simpleRegistry{
		evaluators: evaluatorsByGroupResource(evaluators),
	}
}

func (r *simpleRegistry) Add(e quota.GroupResourceEvaluator) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.evaluators[e.GroupResource()] = e
}

func (r *simpleRegistry) Remove(e quota.GroupResourceEvaluator) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.evaluators, e.GroupResource())
}

func (r *simpleRegistry) Get(gr schema.GroupResource) quota.GroupResourceEvaluator {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.evaluators[gr]
}

func (r *simpleRegistry) List() []quota.GroupResourceEvaluator {
	r.lock.Lock()
	defer r.lock.Unlock()
	return evaluatorsList(r.evaluators)
}

// evaluatorsByGroupResource converts a list of evaluators to a map by group resource.
func evaluatorsByGroupResource(items []quota.GroupResourceEvaluator) map[schema.GroupResource]quota.GroupResourceEvaluator {
	result := map[schema.GroupResource]quota.GroupResourceEvaluator{}
	for _, item := range items {
		result[item.GroupResource()] = item
	}
	return result
}

func evaluatorsList(input map[schema.GroupResource]quota.GroupResourceEvaluator) []quota.GroupResourceEvaluator {
	var result []quota.GroupResourceEvaluator
	for _, item := range input {
		result = append(result, item)
	}
	return result
}
