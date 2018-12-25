// +build linux

package cni

import (
	"testing"

	"encoding/json"
	"io/ioutil"

	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/file"
)

func TestUpdateCNIConf(t *testing.T) {
	testCases := []struct {
		name              string
		expectErr         bool
		cniServiceAddress string
		confDir           string
		initConf          func(confFileName string) error
		verifyFunc        func(confFileName string) (string, error)
	}{
		{
			name:       "cniServiceAddress is empty, so error",
			expectErr:  true,
			initConf:   func(confFileName string) error { return nil },
			verifyFunc: func(confFileName string) (string, error) { return "", nil },
		},
		{
			name:       "conf dir is empty, so error",
			expectErr:  true,
			initConf:   func(confFileName string) error { return nil },
			verifyFunc: func(confFileName string) (string, error) { return "", nil },
		},
		{
			name:       "conf dir contain nothing, so error",
			expectErr:  true,
			confDir:    "testdata/confdir",
			initConf:   func(confFileName string) error { return nil },
			verifyFunc: func(confFileName string) (string, error) { return "", nil },
		},
		{
			name:              "conf dir exist, so no error",
			expectErr:         false,
			cniServiceAddress: "1.1.1.1:6443",
			confDir:           "testdata/confdir",
			initConf: func(confFileName string) error {
				netConf := types.NetConf{
					CNIVersion: "0.2.0",
					Name:       "mynet",
					Type:       "cni_alinet",
				}
				fileContext, err := json.Marshal(netConf)
				if err != nil {
					return err
				}

				err = ioutil.WriteFile(confFileName, fileContext, 0644)
				if err != nil {
					return err
				}

				return nil
			},
			verifyFunc: func(confFileName string) (string, error) {
				//check file exist
				exist, err := file.FileExists(confFileName)
				if !exist || err != nil {
					return "", fmt.Errorf("file not exist or err")
				}

				// read context from file
				fileContext, err := ioutil.ReadFile(confFileName)
				if err != nil {
					return "", fmt.Errorf("error reading %s: %s", confFileName, err)
				}

				// unmarshal context to map
				rawList := make(map[string]interface{})
				if err := json.Unmarshal(fileContext, &rawList); err != nil {
					return "", fmt.Errorf("error parsing configuration list: %s", err)
				}

				// get cni service address
				var cniServiceAddress string
				rawAddress, ok := rawList[confCNIServiceAddressName]
				if ok {
					cniServiceAddress, ok = rawAddress.(string)
					if !ok {
						return "", fmt.Errorf("error parsing configuration list: invalid cniServiceAddress type %T", rawAddress)
					}
				}

				// remove file
				err = os.Remove(confFileName)
				return cniServiceAddress, err
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initConf(tt.confDir + "/net.conf")
			assert.NoError(t, err)

			err = updateCNIConf(tt.cniServiceAddress, tt.confDir)
			if tt.expectErr {
				assert.Error(t, err, "expect an error")
				return
			}
			assert.NoError(t, err)

			address, err := tt.verifyFunc(tt.confDir + "/net.conf")
			assert.NoError(t, err)
			assert.Equal(t, address, tt.cniServiceAddress)
		})

	}
}

func TestParseConfigMapFromFIFO(t *testing.T) {
	testCases := []struct {
		name    string
		obj     interface{}
		address string
	}{
		{
			name:    "obj is nil, so address is empty",
			obj:     nil,
			address: "",
		},
		{
			name: "config map data is nil, so address is empty",
			obj: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapNameOfCNI,
					Namespace: configMapNameSpaceOfCNI,
				},
			},
			address: "",
		},
		{
			name: "address is what we expect",
			obj: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapNameOfCNI,
					Namespace: configMapNameSpaceOfCNI,
				},
				Data: map[string]string{
					configMapCNIServiceAddress: "1.1.1.1",
				},
			},
			address: "1.1.1.1",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			address, err := parseConfigMapFromFIFO(tt.obj, "")
			assert.Error(t, err)
			assert.Equal(t, tt.address, address)
		})
	}
}
