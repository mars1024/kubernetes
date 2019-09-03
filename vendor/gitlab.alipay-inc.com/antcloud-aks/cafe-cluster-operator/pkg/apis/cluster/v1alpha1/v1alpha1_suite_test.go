


package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/kubernetes-incubator/apiserver-builder-alpha/pkg/test"
	"k8s.io/client-go/rest"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/openapi"
)

var testenv *test.TestEnvironment
var config *rest.Config
var cs *clientset.Clientset

func TestV1alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "v1 Suite", []Reporter{test.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	testenv = test.NewTestEnvironment()
	config = testenv.Start(apis.GetAllApiBuilders(), openapi.GetOpenAPIDefinitions)
	cs = clientset.NewForConfigOrDie(config)
})

var _ = AfterSuite(func() {
	testenv.Stop()
})
