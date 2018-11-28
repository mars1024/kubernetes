package poddeletionflowcontrol

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kadmission "k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"
)

var (
	informerFactory = informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	simpleClient    *fake.Clientset
	plugin          kadmission.ValidationInterface

	cms [3]*api.ConfigMap
)

func initPlugin(cm *api.ConfigMap) {
	simpleClient = fake.NewSimpleClientset(cm)

	plugin = &flowControlPlugin{
		client:  simpleClient,
		lister:  informerFactory.Core().InternalVersion().ConfigMaps().Lister(),
		Handler: kadmission.NewHandler(kadmission.Delete),
	}
}

func createCm(cm *api.ConfigMap) {
	informerFactory.Core().InternalVersion().ConfigMaps().Informer().GetStore().Add(cm)
	simpleClient.Core().ConfigMaps(cm.Namespace).Create(cm)
}

func updateCm(cm *api.ConfigMap) {
	simpleClient.Core().ConfigMaps(cm.Namespace).Update(cm)
}

func validatePod(namespace string) error {
	attrs := kadmission.NewAttributesRecord(
		nil,
		nil,
		api.Kind("Pod").WithVersion("version"),
		namespace,
		"",
		api.Resource("pods").WithVersion("version"),
		"",
		kadmission.Delete,
		&user.DefaultInfo{},
	)

	err := plugin.Validate(attrs)
	if err != nil {
		return err
	}

	return nil
}

func TestTransferTimeToInt(t *testing.T) {
	testTime := time.Date(2018, time.August, 21, 12, 0, 0, 0, time.Local)
	intTime, _ := transferTimeToInt(testTime)
	if intTime != 201808211200 {
		t.Fatalf("expect 201808211200 but got %v", intTime)
	}
}

func TestValidatePodWithoutGlobalPdfc(t *testing.T) {
	cm := &api.ConfigMap{}
	initPlugin(cm)

	if err := validatePod("default"); err != nil {
		t.Fatalf("expect error nil but got %v", err)
	}
}

func TestValidatePodWithIncorrectUnit(t *testing.T) {
	cms[0] = &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: GlobalPdfcNamespace,
			Name:      PdfcConfigName,
		},
		Data: map[string]string{
			PdfcConfigRuleKey: `[{"duration":"1d","deleteLimit":12000}]`,
		},
	}
	createCm(cms[0])

	if err := validatePod("default"); !strings.Contains(err.Error(), "unknown unit") {
		t.Fatalf("expect 'unknown unit' error but got %v", err)
	}

	cms[0].Data[PdfcConfigRuleKey] = `[{"duration":"59s","deleteLimit":1000}]`
	if err := validatePod("default"); !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expect 'out of range' error but got %v", err)
	}
	cms[0].Data[PdfcConfigRuleKey] = `[{"duration":"24h1s","deleteLimit":1000}]`
	if err := validatePod("default"); !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expect 'out of range' error but got %v", err)
	}
	if rulesJson[GlobalPdfcNamespace] != "" {
		glog.Errorf("exepct rulesJson '' but got %v", rulesJson[GlobalPdfcNamespace])
	}
}

func TestValidatePodWithOnlyGlobalPdfc(t *testing.T) {
	debugMode = true
	// consider 0.5 second as 1 minute in debug mode
	defaultUpdatePeriod = 500 * time.Millisecond
	defaultCheckPeriod = 100 * time.Millisecond
	defaultCleanupPeriod = 100 * time.Millisecond
	debugFakeTime = time.Date(2018, time.August, 21, 19, 53, 0, 0, time.Local)
	oneMinutePass = make(chan bool)
	needUpdate = make(chan bool)
	needClean = make(chan bool)

	cms[0].Data[PdfcConfigRuleKey] =
		`[{"duration":"1m","deleteLimit":10},{"duration":"2m","deleteLimit":15},{"duration":"5m","deleteLimit":18},{"duration":"10m","deleteLimit":20}]`
	updateCm(cms[0])

	// 201808211953.00-53.59
	for i := 1; i <= 7; i++ {
		validatePod("default")
	}
	oneMinutePass <- true
	// 1954
	oneMinutePass <- true
	// 1955
	for i := 1; i <= 5; i++ {
		validatePod("default")
	}
	oneMinutePass <- true
	// 1956
	oneMinutePass <- true
	// 1957
	oneMinutePass <- true
	// 1958
	time.Sleep(100 * time.Millisecond) // wait for last round of updateCacheAndStorage finished
	for i := 1; i <= 3; i++ {
		validatePod("default")
	}

	if len(namespacedCounters[GlobalPdfcNamespace]) != 4 {
		t.Fatalf("expect couters num 4 but got %v", len(namespacedCounters[GlobalPdfcNamespace]))
	}
	totalCount := []int{3, 3, 8, 15}
	deleteLimit := []int{10, 15, 18, 20}
	windowLen := []int{1, 2, 5, 10}
	for i, counter := range namespacedCounters[GlobalPdfcNamespace] {
		if counter.totalCount != totalCount[i] {
			t.Fatalf("%v: expect totalCount %v but got %v", i, totalCount[i], counter.totalCount)
		}
		if counter.deleteLimit != deleteLimit[i] {
			t.Fatalf("%v: expect deleteLimit %v but got %v", i, deleteLimit[i], counter.deleteLimit)
		}
		if len(counter.window) != windowLen[i] {
			t.Fatalf("%v: expect windowLen %v but got %v", i, windowLen[i], len(counter.window))
		}
		if counter.window[len(counter.window)-1] != 3 {
			t.Fatalf("%v: expect lastMinuteCount 7 but got %v", i, counter.window[len(counter.window)-1])
		}
		if i >= 1 {
			if counter.window[len(counter.window)-2] != 0 {
				t.Fatalf("%v: expect twoMinsAgoCount 0 but got %v", i, counter.window[len(counter.window)-2])
			}
		}
		if i >= 2 {
			if counter.window[len(counter.window)-4] != 5 {
				t.Fatalf("%v: expect fourMinsAgoCount 5 but got %v", i, counter.window[len(counter.window)-4])
			}
		}
		if i >= 3 {
			if counter.window[len(counter.window)-6] != 7 {
				t.Fatalf("%v: expect sixMinsAgoCount 3 but got %v", i, counter.window[len(counter.window)-6])
			}
		}
	}

	cm, _ := simpleClient.Core().ConfigMaps(GlobalPdfcNamespace).Get(PdfcConfigName, metav1.GetOptions{})
	records := `{"201808211953":{"deleteCount":7},"201808211955":{"deleteCount":5}}`
	if cm.Data[PdfcConfigRecordKey] != records {
		t.Fatalf("expect records %v but got %v", records, cm.Data[PdfcConfigRecordKey])
	}

	oneMinutePass <- true
	time.Sleep(100 * time.Millisecond) // wait for last round deletion write to etcd
}

func TestValidatePodWithUserPdfc(t *testing.T) {
	cms[0].Data[PdfcConfigRuleKey] = `[{"duration":"2m","deleteLimit":15},{"duration":"1m","deleteLimit":10},{"duration":"5m","deleteLimit":20}]`
	cms[1] = &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "app1",
			Name:      PdfcConfigName,
		},
		Data: map[string]string{
			PdfcConfigRuleKey: `[{"duration":"2m","deleteLimit":5},{"duration":"1m","deleteLimit":3},{"duration":"5m","deleteLimit":8}]`,
		},
	}
	cms[2] = &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "app2",
			Name:      PdfcConfigName,
		},
		Data: map[string]string{
			PdfcConfigRuleKey: `[{"duration":"2m","deleteLimit":11},{"duration":"1m","deleteLimit":8},{"duration":"5m","deleteLimit":13}]`,
		},
	}
	updateCm(cms[0])
	createCm(cms[1])
	createCm(cms[2])

	debugFakeTime = time.Date(2018, time.August, 21, 20, 53, 0, 0, time.Local)
	needUpdate <- true
	time.Sleep(200 * time.Millisecond) // wait for checkFlowControlRules finished

	// 2053
	for i := 1; i <= 3; i++ {
		validatePod("app1")
	}

	if err := validatePod("app1"); !strings.Contains(err.Error(), "rejected by flow control") {
		t.Fatalf("expect 'rejected' but got %v", err)
	}
	for i := 1; i <= 7; i++ {
		validatePod("app2")
	}
	if err := validatePod("app2"); !strings.Contains(err.Error(), "rejected by flow control") {
		t.Fatalf("expect 'rejected' but got %v", err)
	}
	oneMinutePass <- true
	// 2054
	time.Sleep(100 * time.Millisecond)
	validatePod("app1")
	validatePod("app2")
	validatePod("app2")

	judgeOrder := []string{GlobalPdfcNamespace, "app1", "app2"}
	totalCount := [][]int{{13, 3, 13}, {4, 1, 4}, {9, 2, 9}} // counter sequence is decided by rules sequence
	for i, namespace := range judgeOrder {
		for j, counter := range namespacedCounters[namespace] {
			if counter.totalCount != totalCount[i][j] {
				t.Fatalf("%v/%v: expect totalCount %v but got %v", i, j, totalCount[i][j], counter.totalCount)
			}
		}
	}

	if len(namespacedCounters[GlobalPdfcNamespace]) != 3 {
		t.Fatalf("expect couter len 3 but got %v", len(namespacedCounters[GlobalPdfcNamespace]))
	}

	oneMinutePass <- true
	time.Sleep(100 * time.Millisecond)
}

func TestReloadOldDeletionRecord(t *testing.T) {
	cms[0].Data[PdfcConfigRuleKey] =
		`[{"duration":"1m","deleteLimit":10},{"duration":"2m","deleteLimit":15},{"duration":"5m","deleteLimit":20}]`
	cms[0].Data[PdfcConfigRecordKey] =
		`{"201808202152":{"deleteCount":9},"201808212151":{"deleteCount":5},"201808212153":{"deleteCount":3}}`
	cms[1].Data[PdfcConfigRuleKey] =
		`[{"duration":"1m","deleteLimit":3},{"duration":"2m","deleteLimit":5},{"duration":"5m","deleteLimit":8}]`
	cms[1].Data[PdfcConfigRecordKey] =
		`{"201808202152":{"deleteCount":3},"201808202156":{"deleteCount":2},"201808212152":{"deleteCount":1}}`
	updateCm(cms[0])
	updateCm(cms[1])

	debugFakeTime = time.Date(2018, time.August, 21, 21, 53, 5, 0, time.Local)
	needUpdate <- true
	needClean <- true
	time.Sleep(200 * time.Millisecond)

	// 2153
	for i := 1; i <= 2; i++ {
		validatePod("app1")
	}

	globalWindows := [][]int{
		{5},
		{0, 5},
		{0, 0, 5, 0, 5},
	}
	for i, counter := range namespacedCounters[GlobalPdfcNamespace] {
		for j, _ := range counter.window {
			if counter.window[j] != globalWindows[i][j] {
				t.Fatalf("%v/%v: expect window value %v but got %v", i, j, globalWindows[i][j], counter.window[j])
			}
		}
	}

	cm, _ := simpleClient.Core().ConfigMaps("app1").Get(PdfcConfigName, metav1.GetOptions{})
	data := `{"201808202156":{"deleteCount":2},"201808212152":{"deleteCount":1}}`
	if cm.Data[PdfcConfigRecordKey] != data {
		t.Fatalf("exepct records %v but got %v", data, cm.Data[PdfcConfigRecordKey])
	}
}
