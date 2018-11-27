package verflag

import (
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

func TestVersionValue_Get(t *testing.T) {

	buildTime := time.Now()
	testCasees := []struct {
		name          string
		version       apimachineryversion.Info
		expectVersion string
	}{
		{
			name:          "version  is nil",
			version:       apimachineryversion.Info{},
			expectVersion: "  ",
		},
		{
			name: "version  is v1.10",
			version: apimachineryversion.Info{
				GitVersion: "v1.10",
			},
			expectVersion: "v1.10  ",
		},
		{
			name: "version  is v1.10 + time",
			version: apimachineryversion.Info{
				GitVersion: "v1.10",
				BuildDate:  buildTime.String(),
			},
			expectVersion: "v1.10 " + buildTime.String() + " ",
		},
		{
			name: "version  is v1.10 + time + commit shot id, with truncation",
			version: apimachineryversion.Info{
				GitVersion: "v1.10",
				BuildDate:  buildTime.String(),
				GitCommit:  "123456789",
			},
			expectVersion: "v1.10 " + buildTime.String() + " 1234567",
		},
		{
			name: "version  is v1.10 + time + commit shot id，with out truncation",
			version: apimachineryversion.Info{
				GitVersion: "v1.10",
				BuildDate:  buildTime.String(),
				GitCommit:  "1234567",
			},
			expectVersion: "v1.10 " + buildTime.String() + " 1234567",
		},
		{
			name: "version  is v1.10 + time + commit shot id",
			version: apimachineryversion.Info{
				GitVersion: "v1.10",
				BuildDate:  buildTime.String(),
				GitCommit:  "123456",
			},
			expectVersion: "v1.10 " + buildTime.String() + " 123456",
		},
	}
	for _, cs := range testCasees {
		t.Run(cs.name, func(t *testing.T) {
			assert.Equal(t, cs.expectVersion, GetVersion(cs.version))
		})
	}
}
