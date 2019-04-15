package sigma

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDiskQuotaToLabel(t *testing.T) {
	tests := []struct {
		name             string
		diskQuota        map[string]string
		expectValue      string
		otherExpectValue string
	}{
		{
			name:             "'.*' only",
			diskQuota:        map[string]string{".*": "10g"},
			expectValue:      "10g",
			otherExpectValue: "10g",
		},
		{
			name:             "'/' only",
			diskQuota:        map[string]string{"/": "10g"},
			expectValue:      "/=10g",
			otherExpectValue: "/=10g",
		},
		{
			name:             "'.*' with other diskquota",
			diskQuota:        map[string]string{".*": "10g", "/home": "20g"},
			expectValue:      "10g",
			otherExpectValue: "/home=20g",
		},
		{
			name:             "'/' with other diskquota",
			diskQuota:        map[string]string{"/": "10g", "/home": "20g"},
			expectValue:      "/=10g",
			otherExpectValue: "/home=20g",
		},
		{
			name:             "test for pouch format",
			diskQuota:        map[string]string{"/&/home": "10g"},
			expectValue:      "/=10g",
			otherExpectValue: "/home=10g",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := ParseDiskQuotaToLabel(tt.diskQuota)
			rootDiskQuotapass := false
			items := strings.Split(value, ";")
			for _, item := range items {
				if item == tt.expectValue {
					rootDiskQuotapass = true
					break
				}
			}
			otherDiskQuotapass := false
			for _, item := range items {
				if item == tt.otherExpectValue {
					otherDiskQuotapass = true
					break
				}
			}
			assert.Equal(t, rootDiskQuotapass, true)
			assert.Equal(t, otherDiskQuotapass, true)
		})
	}
}

func TestParseDiskQuota(t *testing.T) {
	tests := []struct {
		name            string
		diskQuotaStr    string
		expectDiskQuota map[string]string
	}{
		{
			name:            "'.*' only",
			diskQuotaStr:    "10g",
			expectDiskQuota: map[string]string{".*": "10g"},
		},
		{
			name:            "'/' only",
			diskQuotaStr:    "/=10g",
			expectDiskQuota: map[string]string{"/": "10g"},
		},
		{
			name:            "'.*' with other diskquota",
			diskQuotaStr:    "10g;/home=20g;/home/t4=30g",
			expectDiskQuota: map[string]string{".*": "10g", "/home": "20g", "/home/t4": "30g"},
		},
		{
			name:            "'/' with other diskquota",
			diskQuotaStr:    "/=10g;/home=20g;/home/t4=30g",
			expectDiskQuota: map[string]string{"/": "10g", "/home": "20g", "/home/t4": "30g"},
		},
		{
			name:            "test for pouch format",
			diskQuotaStr:    "/home=10g;/home/t4=10g;/=10g",
			expectDiskQuota: map[string]string{"/&/home&/home/t4": "10g"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := ParseDiskQuota(tt.diskQuotaStr)
			assert.Equal(t, value, tt.expectDiskQuota)
		})
	}
}
