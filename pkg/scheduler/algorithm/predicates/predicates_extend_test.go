package predicates

import (
	"testing"
)

var (
	fixedOverhead = 1 + AdjustedMemoryOverhead
)

func TestIsResourceApproximate(t *testing.T) {
	vmMemoryInByte := int64(float64(int64(120611)*1024*1024) * fixedOverhead)
	allocatedMemoryInByte := int64(122880) * 1024 * 1024 // 120 GiB
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, AllowedMemoryOverhead)
	if r == false {
		t.Errorf("memory should be in the limit range")
	}
}

func TestIsResourceApproximate_1_G(t *testing.T) {
	vmMemoryInByte := int64(float64(int64(906684)*1024) * fixedOverhead)
	allocatedMemoryInByte := int64(1024) * 1024 * 1024 // 1GiB
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, AllowedMemoryOverhead)
	if r == false {
		t.Errorf("memory should be in the limit range")
	}
}

func TestIsResourceApproximate_0_5_G(t *testing.T) {
	vmMemoryInByte := int64(float64(int64(390588) * 1024) * fixedOverhead)
	allocatedMemoryInByte := int64(512) * 1024 * 1024 // 0.5GiB
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, AllowedMemoryOverhead)
	if r == false {
		t.Errorf("memory should be in the limit range")
	}
}

func TestIsResourceApproximate_4_G(t *testing.T) {
	vmMemoryInByte := int64(float64(int64(3740524) * 1024) * fixedOverhead)
	allocatedMemoryInByte := int64(4) * 1024 * 1024 * 1024 // 4GiB
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, AllowedMemoryOverhead)
	if r == false {
		t.Errorf("memory should be in the limit range")
	}
}

func TestIsResourceApproximate_12_G(t *testing.T) {
	vmMemoryInByte := int64(float64(int64(11932536) * 1024) * fixedOverhead)
	allocatedMemoryInByte := int64(12) * 1024 * 1024 * 1024 // 120GiB
	r := IsResourceApproximate(vmMemoryInByte, allocatedMemoryInByte, AllowedMemoryOverhead)
	if r == false {
		t.Errorf("memory should  be in the limit range")
	}
}
