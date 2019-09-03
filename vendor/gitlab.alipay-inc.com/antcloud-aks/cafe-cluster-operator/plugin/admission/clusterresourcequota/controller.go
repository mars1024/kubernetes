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

package clusterresourcequota

import (
	"fmt"
	"sync"
	"strings"
	"sort"
	"time"

	"github.com/golang/glog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/api/core/v1"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota/generic"
)

// Evaluator is used to see if quota constraints are satisfied.
type Evaluator interface {
	// Evaluate takes an operation and checks to see if quota constraints are satisfied.  It returns an error if they are not.
	// The default implementation process related operations in chunks when possible.
	Evaluate(a admission.Attributes, cluster string) error
}

type admissionWaiter struct {
	attributes admission.Attributes
	cluster    string
	finished   chan struct{}
	result     error
}

type defaultDeny struct{}

func (defaultDeny) Error() string {
	return "DEFAULT DENY"
}

// IsDefaultDeny returns true if the error is defaultDeny
func IsDefaultDeny(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(defaultDeny)
	return ok
}

func newAdmissionWaiter(a admission.Attributes, c string) *admissionWaiter {
	return &admissionWaiter{
		attributes: a,
		cluster:    c,
		finished:   make(chan struct{}),
		result:     defaultDeny{},
	}
}

type quotaEvaluator struct {
	quotaAccessor QuotaAccessor
	// lockAcquisitionFunc acquires any required locks and returns a cleanup method to defer
	lockAcquisitionFunc func([]v1.ResourceQuota) func()

	ignoredResources map[schema.GroupResource]struct{}

	// registry that knows how to measure usage for objects
	registry quota.Registry

	queue      *workqueue.Type
	workLock   sync.Mutex
	work       map[string][]*admissionWaiter
	dirtyWork  map[string][]*admissionWaiter
	inProgress sets.String

	// controls the run method so that we can cleanly conform to the Evaluator interface
	workers int
	stopCh  <-chan struct{}
	init    sync.Once
}

// NewQuotaEvaluator configures an admission controller that can enforce quota constraints
// using the provided registry.  The registry must have the capability to handle group/kinds that
// are persisted by the server this admission controller is intercepting
func NewQuotaEvaluator(quotaAccessor QuotaAccessor, ignoredResources map[schema.GroupResource]struct{}, quotaRegistry quota.Registry, lockAcquisitionFunc func([]v1.ResourceQuota) func(), workers int, stopCh <-chan struct{}) Evaluator {
	return &quotaEvaluator{
		quotaAccessor:       quotaAccessor,
		lockAcquisitionFunc: lockAcquisitionFunc,

		ignoredResources: ignoredResources,
		registry:         quotaRegistry,

		queue:      workqueue.NewNamed("admission_cluster_quota_controller"),
		work:       map[string][]*admissionWaiter{},
		dirtyWork:  map[string][]*admissionWaiter{},
		inProgress: sets.String{},

		workers: workers,
		stopCh:  stopCh,
	}
}

// Run begins watching and syncing.
func (e *quotaEvaluator) run() {
	defer utilruntime.HandleCrash()

	for i := 0; i < e.workers; i++ {
		go wait.Until(e.doWork, time.Second, e.stopCh)
	}
	<-e.stopCh
	glog.Infof("Shutting down quota evaluator")
	e.queue.ShutDown()
}

func (e *quotaEvaluator) doWork() {
	workFunc := func() bool {
		clusterNs, admissionWaiters, quit := e.getWork()
		if quit {
			return true
		}
		defer e.completeWork(clusterNs)
		if len(admissionWaiters) == 0 {
			return false
		}
		e.checkAttributes(clusterNs, admissionWaiters)
		return false
	}
	for {
		if quit := workFunc(); quit {
			glog.Infof("quota evaluator worker shutdown")
			return
		}
	}
}

// checkAttributes iterates evaluates all the waiting admissionAttributes.  It will always notify all waiters
// before returning.  The default is to deny.
func (e *quotaEvaluator) checkAttributes(clusterNs string, admissionWaiters []*admissionWaiter) {
	// notify all on exit
	defer func() {
		for _, admissionWaiter := range admissionWaiters {
			close(admissionWaiter.finished)
		}
	}()

	clusterNsArray := strings.Split(clusterNs, "/")
	quotas, err := e.quotaAccessor.GetQuotas(clusterNsArray[0], clusterNsArray[1])
	if err != nil {
		for _, admissionWaiter := range admissionWaiters {
			admissionWaiter.result = err
		}
		return
	}
	if len(quotas) == 0 {
		for _, admissionWaiter := range admissionWaiters {
			admissionWaiter.result = nil
		}
		return
	}

	if e.lockAcquisitionFunc != nil {
		releaseLocks := e.lockAcquisitionFunc(quotas)
		defer releaseLocks()
	}

	e.checkQuotas(quotas, admissionWaiters, 3)
}

// checkQuotas checks the admission attributes against the passed quotas.  If a quota applies, it will attempt to update it
// AFTER it has checked all the admissionAttributes.  The method breaks down into phase like this:
// 0. make a copy of the quotas to act as a "running" quota so we know what we need to update and can still compare against the
//    originals
// 1. check each admission attribute to see if it fits within *all* the quotas.  If it doesn't fit, mark the waiter as failed
//    and the running quota don't change.  If it did fit, check to see if any quota was changed.  It there was no quota change
//    mark the waiter as succeeded.  If some quota did change, update the running quotas
// 2. If no running quota was changed, return now since no updates are needed.
// 3. for each quota that has changed, attempt an update.  If all updates succeeded, update all unset waiters to success status and return.  If the some
//    updates failed on conflict errors and we have retries left, re-get the failed quota from our cache for the latest version
//    and recurse into this method with the subset.  It's safe for us to evaluate ONLY the subset, because the other quota
//    documents for these waiters have already been evaluated.  Step 1, will mark all the ones that should already have succeeded.
func (e *quotaEvaluator) checkQuotas(quotas []v1.ResourceQuota, admissionAttributes []*admissionWaiter, remainingRetries int) {
	// yet another copy to compare against originals to see if we actually have deltas
	originalQuotas, err := copyQuotas(quotas)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	atLeastOneChanged := false
	for i := range admissionAttributes {
		admissionAttribute := admissionAttributes[i]
		newQuotas, err := e.checkRequest(quotas, admissionAttribute.attributes)
		if err != nil {
			admissionAttribute.result = err
			continue
		}

		// Don't update quota for admissionAttributes that correspond to dry-run requests
		if admissionAttribute.attributes.IsDryRun() {
			admissionAttribute.result = nil
			continue
		}

		// if the new quotas are the same as the old quotas, then this particular one doesn't issue any updates
		// that means that no quota docs applied, so it can get a pass
		atLeastOneChangeForThisWaiter := false
		for j := range newQuotas {
			if !quota.Equals(quotas[j].Status.Used, newQuotas[j].Status.Used) {
				atLeastOneChanged = true
				atLeastOneChangeForThisWaiter = true
				break
			}
		}

		if !atLeastOneChangeForThisWaiter {
			admissionAttribute.result = nil
		}

		quotas = newQuotas
	}

	// if none of the requests changed anything, there's no reason to issue an update, just fail them all now
	if !atLeastOneChanged {
		return
	}

	// now go through and try to issue updates.  Things get a little weird here:
	// 1. check to see if the quota changed.  If not, skip.
	// 2. if the quota changed and the update passes, be happy
	// 3. if the quota changed and the update fails, add the original to a retry list
	var updatedFailedQuotas []v1.ResourceQuota
	var lastErr error
	for i := range quotas {
		newQuota := quotas[i]

		// if this quota didn't have its status changed, skip it
		if quota.Equals(originalQuotas[i].Status.Used, newQuota.Status.Used) {
			continue
		}

		if err := e.quotaAccessor.UpdateQuotaStatus(&newQuota); err != nil {
			updatedFailedQuotas = append(updatedFailedQuotas, newQuota)
			lastErr = err
		}
	}

	if len(updatedFailedQuotas) == 0 {
		// all the updates succeeded.  At this point, anything with the default deny error was just waiting to
		// get a successful update, so we can mark and notify
		for _, admissionAttribute := range admissionAttributes {
			if IsDefaultDeny(admissionAttribute.result) {
				admissionAttribute.result = nil
			}
		}
		return
	}

	// at this point, errors are fatal.  Update all waiters without status to failed and return
	if remainingRetries <= 0 {
		for _, admissionAttribute := range admissionAttributes {
			if IsDefaultDeny(admissionAttribute.result) {
				admissionAttribute.result = lastErr
			}
		}
		return
	}

	// this retry logic has the same bug that its possible to be checking against quota in a state that never actually exists where
	// you've added a new documented, then updated an old one, your resource matches both and you're only checking one
	// updates for these quota names failed.  Get the current quotas in the namespace, compare by name, check to see if the
	// resource versions have changed.  If not, we're going to fall through an fail everything.  If they all have, then we can try again
	newQuotas, err := e.quotaAccessor.GetQuotas(admissionAttributes[0].cluster, quotas[0].Namespace)
	if err != nil {
		// this means that updates failed.  Anything with a default deny error has failed and we need to let them know
		for _, admissionAttribute := range admissionAttributes {
			if IsDefaultDeny(admissionAttribute.result) {
				admissionAttribute.result = lastErr
			}
		}
		return
	}

	// this logic goes through our cache to find the new version of all quotas that failed update.  If something has been removed
	// it is skipped on this retry.  After all, you removed it.
	quotasToCheck := []v1.ResourceQuota{}
	for _, newQuota := range newQuotas {
		for _, oldQuota := range updatedFailedQuotas {
			if newQuota.Name == oldQuota.Name {
				quotasToCheck = append(quotasToCheck, newQuota)
				break
			}
		}
	}
	e.checkQuotas(quotasToCheck, admissionAttributes, remainingRetries-1)
}

// checkRequest verifies that the request does not exceed any quota constraint. it returns a copy of quotas not yet persisted
// that capture what the usage would be if the request succeeded.  It return an error if there is insufficient quota to satisfy the request
func (e *quotaEvaluator) checkRequest(quotas []v1.ResourceQuota, a admission.Attributes) ([]v1.ResourceQuota, error) {
	evaluator := e.registry.Get(a.GetResource().GroupResource())
	if evaluator == nil {
		return quotas, nil
	}

	return CheckRequest(quotas, a, evaluator)
}

func CheckRequest(quotas []v1.ResourceQuota, a admission.Attributes, evaluator quota.GroupResourceEvaluator) (
	[]v1.ResourceQuota, error) {
	if !evaluator.Handles(a) {
		return quotas, nil
	}

	// if we have limited resources enabled for this resource, always calculate usage
	inputObject := a.GetObject()

	// determine the set of resource names that must exist in a covering quota
	var limitedResourceNames []v1.ResourceName
	limitedResourceNamesSet := quota.ToSet(limitedResourceNames)

	// find the set of quotas that are pertinent to this request
	// reject if we match the quota, but usage is not calculated yet
	// reject if the input object does not satisfy quota constraints
	// if there are no pertinent quotas, we can just return
	interestingQuotaIndexes := []int{}
	// track the cumulative set of resources that were required across all quotas
	// this is needed to know if we have satisfied any constraints where consumption
	// was limited by default.
	restrictedResourcesSet := sets.String{}
	var restrictedScopes []v1.ScopedResourceSelectorRequirement
	for i := range quotas {
		resourceQuota := quotas[i]
		scopeSelectors := getScopeSelectorsFromResourceQuota(resourceQuota)
		localRestrictedScopes, err := evaluator.MatchingScopes(inputObject, scopeSelectors)
		if err != nil {
			return nil, fmt.Errorf("error matching scopes of quota %s, err: %v", resourceQuota.Name, err)
		}
		for _, scope := range localRestrictedScopes {
			restrictedScopes = append(restrictedScopes, scope)
		}

		match, err := evaluator.Matches(&resourceQuota, inputObject)
		if err != nil {
			glog.Errorf("Error occurred while matching resource quota, %v, against input object. Err: %v", resourceQuota, err)
			return quotas, err
		}
		if !match {
			continue
		}

		hardResources := quota.ResourceNames(resourceQuota.Status.Hard)
		restrictedResources := evaluator.MatchingResources(hardResources)
		if err := evaluator.Constraints(restrictedResources, inputObject); err != nil {
			return nil, admission.NewForbidden(a, fmt.Errorf("failed quota: %s: %v", resourceQuota.Name, err))
		}
		if !hasUsageStats(&resourceQuota) {
			return nil, admission.NewForbidden(a, fmt.Errorf("status unknown for quota: %s", resourceQuota.Name))
		}
		interestingQuotaIndexes = append(interestingQuotaIndexes, i)
		localRestrictedResourcesSet := quota.ToSet(restrictedResources)
		restrictedResourcesSet.Insert(localRestrictedResourcesSet.List()...)
	}

	// verify that for every resource that had limited by default consumption
	// enabled that there was a corresponding quota that covered its use.
	// if not, we reject the request.
	hasNoCoveringQuota := limitedResourceNamesSet.Difference(restrictedResourcesSet)
	if len(hasNoCoveringQuota) > 0 {
		return quotas, admission.NewForbidden(a, fmt.Errorf("insufficient quota to consume: %v", strings.Join(hasNoCoveringQuota.List(), ",")))
	}

	// verify that for every scope that had limited access enabled
	// that there was a corresponding quota that covered it.
	// if not, we reject the request.
	var limitedScopes []v1.ScopedResourceSelectorRequirement
	scopesHasNoCoveringQuota, err := evaluator.UncoveredQuotaScopes(limitedScopes, restrictedScopes)
	if err != nil {
		return quotas, err
	}
	if len(scopesHasNoCoveringQuota) > 0 {
		return quotas, fmt.Errorf("insufficient quota to match these scopes: %v", scopesHasNoCoveringQuota)
	}

	if len(interestingQuotaIndexes) == 0 {
		return quotas, nil
	}

	// Usage of some resources cannot be counted in isolation. For example, when
	// the resource represents a number of unique references to external
	// resource. In such a case an evaluator needs to process other objects in
	// the same namespace which needs to be known.
	namespace := a.GetNamespace()
	if accessor, err := meta.Accessor(inputObject); namespace != "" && err == nil {
		if accessor.GetNamespace() == "" {
			accessor.SetNamespace(namespace)
		}
	}
	// there is at least one quota that definitely matches our object
	// as a result, we need to measure the usage of this object for quota
	// on updates, we need to subtract the previous measured usage
	// if usage shows no change, just return since it has no impact on quota
	deltaUsage, err := evaluator.Usage(inputObject)
	if err != nil {
		return quotas, err
	}

	// ensure that usage for input object is never negative (this would mean a resource made a negative resource requirement)
	if negativeUsage := quota.IsNegative(deltaUsage); len(negativeUsage) > 0 {
		return nil, admission.NewForbidden(a, fmt.Errorf("quota usage is negative for resource(s): %s", prettyPrintResourceNames(negativeUsage)))
	}

	if admission.Update == a.GetOperation() {
		prevItem := a.GetOldObject()
		if prevItem == nil {
			return nil, admission.NewForbidden(a, fmt.Errorf("unable to get previous usage since prior version of object was not found"))
		}

		// if we can definitively determine that this is not a case of "create on update",
		// then charge based on the delta.  Otherwise, bill the maximum
		metadata, err := meta.Accessor(prevItem)
		if err == nil && len(metadata.GetResourceVersion()) > 0 {
			prevUsage, innerErr := evaluator.Usage(prevItem)
			if innerErr != nil {
				return quotas, innerErr
			}
			deltaUsage = quota.SubtractWithNonNegativeResult(deltaUsage, prevUsage)
		}
	}

	if quota.IsZero(deltaUsage) {
		return quotas, nil
	}

	outQuotas, err := copyQuotas(quotas)
	if err != nil {
		return nil, err
	}

	for _, index := range interestingQuotaIndexes {
		resourceQuota := outQuotas[index]

		hardResources := quota.ResourceNames(resourceQuota.Spec.Hard)
		requestedUsage := quota.Mask(deltaUsage, hardResources)
		newUsage := quota.Add(resourceQuota.Status.Used, requestedUsage)
		maskedNewUsage := quota.Mask(newUsage, quota.ResourceNames(requestedUsage))

		if allowed, exceeded := quota.LessThanOrEqual(maskedNewUsage, resourceQuota.Spec.Hard); !allowed {
			failedRequestedUsage := quota.Mask(requestedUsage, exceeded)
			failedUsed := quota.Mask(resourceQuota.Status.Used, exceeded)
			failedHard := quota.Mask(resourceQuota.Status.Hard, exceeded)
			return nil, admission.NewForbidden(a,
				fmt.Errorf("exceeded quota: %s, requested: %s, used: %s, limited: %s",
					resourceQuota.Name,
					prettyPrint(failedRequestedUsage),
					prettyPrint(failedUsed),
					prettyPrint(failedHard)))
		}

		// update to the new usage number
		outQuotas[index].Status.Used = newUsage
	}

	return outQuotas, nil
}

func copyQuotas(in []v1.ResourceQuota) ([]v1.ResourceQuota, error) {
	out := make([]v1.ResourceQuota, 0, len(in))
	for _, original := range in {
		out = append(out, *original.DeepCopy())
	}

	return out, nil
}

func getScopeSelectorsFromResourceQuota(quota v1.ResourceQuota) []v1.ScopedResourceSelectorRequirement {
	var selectors []v1.ScopedResourceSelectorRequirement
	for _, scope := range quota.Spec.Scopes {
		selectors = append(selectors, v1.ScopedResourceSelectorRequirement{
			ScopeName: scope,
			Operator:  v1.ScopeSelectorOpExists})
	}
	if quota.Spec.ScopeSelector != nil {
		for _, scopeSelector := range quota.Spec.ScopeSelector.MatchExpressions {
			selectors = append(selectors, scopeSelector)
		}
	}
	return selectors
}

func (e *quotaEvaluator) Evaluate(a admission.Attributes, cluster string) error {
	e.init.Do(func() {
		go e.run()
	})

	// is this resource ignored?
	gvr := a.GetResource()
	gr := gvr.GroupResource()
	if _, ok := e.ignoredResources[gr]; ok {
		return nil
	}

	// if we do not know how to evaluate use for this resource, create an evaluator
	evaluator := e.registry.Get(gr)
	if evaluator == nil {
		// create an object count evaluator if no evaluator previously registered
		// note, we do not need aggregate usage here, so we pass a nil informer func
		evaluator = generic.NewObjectCountEvaluator(gr, nil, "")
		e.registry.Add(evaluator)
	}
	// for this kind, check if the operation could mutate any quota resources
	// if no resources tracked by quota are impacted, then just return
	if !evaluator.Handles(a) {
		return nil
	}
	waiter := newAdmissionWaiter(a, cluster)

	e.addWork(waiter)

	// wait for completion or timeout
	select {
	case <-waiter.finished:
	case <-time.After(10 * time.Second):
		return apierrors.NewInternalError(fmt.Errorf("resource quota evaluates timeout"))
	}

	return waiter.result
}

func (e *quotaEvaluator) addWork(a *admissionWaiter) {
	e.workLock.Lock()
	defer e.workLock.Unlock()

	clusterNs := a.cluster + "/" + a.attributes.GetNamespace()
	// this Add can trigger a Get BEFORE the work is added to a list, but this is ok because the getWork routine
	// waits the worklock before retrieving the work to do, so the writes in this method will be observed
	e.queue.Add(clusterNs)

	if e.inProgress.Has(clusterNs) {
		e.dirtyWork[clusterNs] = append(e.dirtyWork[clusterNs], a)
		return
	}

	e.work[clusterNs] = append(e.work[clusterNs], a)
}

func (e *quotaEvaluator) completeWork(cluster string) {
	e.workLock.Lock()
	defer e.workLock.Unlock()

	e.queue.Done(cluster)
	e.work[cluster] = e.dirtyWork[cluster]
	delete(e.dirtyWork, cluster)
	e.inProgress.Delete(cluster)
}

// getWork returns a namespace, a list of work items in that
// namespace, and a shutdown boolean.  If not shutdown then the return
// must eventually be followed by a call on completeWork for the
// returned namespace (regardless of whether the work item list is
// empty).
func (e *quotaEvaluator) getWork() (string, []*admissionWaiter, bool) {
	uncastCluster, shutdown := e.queue.Get()
	if shutdown {
		return "", []*admissionWaiter{}, shutdown
	}
	clusterNs := uncastCluster.(string)

	e.workLock.Lock()
	defer e.workLock.Unlock()
	// at this point, we know we have a coherent view of e.work.  It is entirely possible
	// that our workqueue has another item requeued to it, but we'll pick it up early.  This ok
	// because the next time will go into our dirty list

	work := e.work[clusterNs]
	delete(e.work, clusterNs)
	delete(e.dirtyWork, clusterNs)
	e.inProgress.Insert(clusterNs)
	return clusterNs, work, false
}

// prettyPrint formats a resource list for usage in errors
// it outputs resources sorted in increasing order
func prettyPrint(item v1.ResourceList) string {
	parts := []string{}
	keys := []string{}
	for key := range item {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := item[v1.ResourceName(key)]
		constraint := key + "=" + value.String()
		parts = append(parts, constraint)
	}
	return strings.Join(parts, ",")
}

func prettyPrintResourceNames(a []v1.ResourceName) string {
	var values []string
	for _, value := range a {
		values = append(values, string(value))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

// hasUsageStats returns true if for each hard constraint there is a value for its current usage
func hasUsageStats(resourceQuota *v1.ResourceQuota) bool {
	for resourceName := range resourceQuota.Status.Hard {
		if _, found := resourceQuota.Status.Used[resourceName]; !found {
			return false
		}
	}
	return true
}
