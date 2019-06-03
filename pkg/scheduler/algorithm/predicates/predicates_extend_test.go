package predicates

import (
	"testing"
)

func TestIsResourceApproximate(t *testing.T) {
	vmMemoryInByte := int64(120611) * 1024 * 1024
	allocatedMemoryInByte := int64(122880) * 1024 * 1024
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, 0.05)
	if r == false {
		t.Errorf("memory should be in the limit range")
	}
}

