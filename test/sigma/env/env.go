package env

import (
	"os"
)

const (
	DefaultTestdata   = "/tmp/testdata"
	DefaultKubeconfig = "/etc/kubernetes/kubelet.conf"
	TesterJituan      = "jituan"
	TesterAnt         = "ant"
)

var TestDataDir string
var Tester string

func init() {
	TestDataDir = GetTestDataDir()
	Tester = GetTester()
}

// GetTestDataDir get testdata directory.
func GetTestDataDir() string {
	get, ok := os.LookupEnv("TEST_DATA_DIR")
	if ok {
		return get
	}

	return DefaultTestdata
}

// GetKubeconfig get the kubeconfig file.
func GetKubeconfig() string {
	get, ok := os.LookupEnv("KUBECONFIG")
	if ok {
		return get
	}
	return DefaultKubeconfig
}

// GetTester return the tester, it could be jituan or ant.
func GetTester() string {
	get, ok := os.LookupEnv("TESTER")
	if ok {
		return get
	}
	return TesterJituan
}
