package convertible

import (
	"sync/atomic"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog"
)

var metadataAccessor = meta.NewAccessor()

var ResourceAllocationPrioritiesConvertible atomic.Value

func Init(enableResourceAllocationPrioritiesConvertible bool) {
	ResourceAllocationPrioritiesConvertible.Store(enableResourceAllocationPrioritiesConvertible)

	if enableResourceAllocationPrioritiesConvertible {
		klog.V(3).Infof("both least and most requested priorities loaded. the scheduler is now convertible")
	}
}
