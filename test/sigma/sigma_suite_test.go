package sigma

import (
	"flag"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/golang/glog"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	morereporters "github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"

	// test sources
	_ "k8s.io/kubernetes/test/sigma/ant-sigma-bvt"
	_ "k8s.io/kubernetes/test/sigma/apiserver"
	_ "k8s.io/kubernetes/test/sigma/cni"
	_ "k8s.io/kubernetes/test/sigma/common"
	_ "k8s.io/kubernetes/test/sigma/controller"
	_ "k8s.io/kubernetes/test/sigma/kubelet"
	_ "k8s.io/kubernetes/test/sigma/scheduler"
)

func init() {
	flag.StringVar(&util.TestDataDir, "test-data-dir", os.Getenv("TEST_DATA_DIR"), "dir which contains many test files.")
	flag.StringVar(&util.SigmaPauseImage, "sigma-pause-image", os.Getenv("SIGMA_PAUSE_IMAGE"), "sigma pause image name")

	// for ant e2e use
	//load cert-file and server address.
	flag.StringVar(&util.AlipayCertPath, "alipay-cert-path", os.Getenv("ALIPAY_CERT_PATH"), "sigma2.0 cert file path.")
	flag.StringVar(&util.AlipayAdapterAddress, "alipay-adapter-addr", os.Getenv("ALIPAY_ADAPTER"), "sigma adapter address, e.g. ip:port.")
	flag.StringVar(&util.ArmoryUser, "armory-user", os.Getenv("ARMORY_USER"), "armory user.")
	flag.StringVar(&util.ArmoryKey, "armory-key", os.Getenv("ARMORY_KEY"), "armory key.")

	framework.ViperizeFlags()
}

func TestE2eNode(t *testing.T) {
	RegisterFailHandler(Fail)
	reporters := []Reporter{}
	reportDir := framework.TestContext.ReportDir
	if reportDir != "" {
		// Create the directory if it doesn't already exists
		if err := os.MkdirAll(reportDir, 0755); err != nil {
			glog.Errorf("Failed creating report directory: %v", err)
		} else {
			// Configure a junit reporter to write to the directory
			junitFile := fmt.Sprintf("junit_%s_%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode)
			junitPath := path.Join(reportDir, junitFile)
			reporters = append(reporters, morereporters.NewJUnitReporter(junitPath))
		}
	}
	RunSpecsWithDefaultAndCustomReporters(t, "Sigma E2E Suite", reporters)
}
