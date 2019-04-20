package predicates

import (
	"fmt"
)

// InsufficientResourceError is an error type that indicates what kind of resource limit is
// hit and caused the unfitting failure.
type InsufficientCPUSetError struct {
	// resourceName is the name of the resource that is insufficient
	ResourceName string
	requested    int64
	used         int64
	capacity     int64
}

// NewInsufficientResourceError returns an InsufficientResourceError.
func NewInsufficientCPUSetError(resourceName string, requested, used, capacity int64) *InsufficientCPUSetError {
	return &InsufficientCPUSetError{
		ResourceName: resourceName,
		requested:    requested,
		used:         used,
		capacity:     capacity,
	}
}

func (e *InsufficientCPUSetError) Error() string {
	return fmt.Sprintf("Node didn't have enough resource: %s, requested: %d, used: %d, capacity: %d",
		e.ResourceName, e.requested, e.used, e.capacity)
}

// GetReason returns the reason of the InsufficientResourceError.
func (e *InsufficientCPUSetError) GetReason() string {
	return fmt.Sprintf("Insufficient %v", e.ResourceName)
}

// GetInsufficientAmount returns the amount of the insufficient resource of the error.
func (e *InsufficientCPUSetError) GetInsufficientAmount() int64 {
	return e.requested - (e.capacity - e.used)
}
