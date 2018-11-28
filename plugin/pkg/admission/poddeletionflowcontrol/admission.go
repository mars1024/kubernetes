package poddeletionflowcontrol

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	"k8s.io/kubernetes/staging/src/k8s.io/client-go/util/retry"
)

const (
	PluginName = "PodDeletionFlowControl"

	GlobalPdfcNamespace    = "kube-system"
	PdfcConfigName         = "pod-deletion-flow-control"
	PdfcConfigRuleKey      = "rules"
	PdfcConfigRecordKey    = "records"
	PdfcConfigWhiteListKey = "whitelist"

	// it's a little trick
	// we use int 201808211954 to represent time for convenience
	// so if one day pass, 201808221954-20180821954=10000 instead of 24*60
	maxTimeWindow = 10000
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// flowControlPlugin is an implementation of admission.Interface.
type flowControlPlugin struct {
	*admission.Handler
	client internalclientset.Interface
	lister corelisters.ConfigMapLister
}

type Rule struct {
	Duration    string `json:"duration"`
	DeleteLimit int    `json:"deleteLimit"`
}

type Record struct {
	DeleteCount int `json:"deleteCount"`
}

type Counter struct {
	totalCount  int
	deleteLimit int
	window      []int
}

var (
	_ admission.ValidationInterface = &flowControlPlugin{}
	_                               = kubeapiserveradmission.WantsInternalKubeInformerFactory(&flowControlPlugin{})
	_                               = kubeapiserveradmission.WantsInternalKubeClientSet(&flowControlPlugin{})

	// use rulesJson to check whether rules had changed
	rulesJson            = make(map[string]string)
	defaultCheckPeriod   = time.Minute
	defaultUpdatePeriod  = 40 * time.Second
	defaultCleanupPeriod = time.Hour

	countersLock       sync.Mutex
	firstCounters      = true
	namespacedCounters = make(map[string][]*Counter)

	debugMode     bool
	debugFakeTime time.Time
	oneMinutePass chan bool
	needUpdate    chan bool
	needClean     chan bool
)

// NewPlugin creates a new flow control admission plugin.
func NewPlugin() *flowControlPlugin {
	return &flowControlPlugin{
		Handler: admission.NewHandler(admission.Delete),
	}
}

func (plugin *flowControlPlugin) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	if plugin.lister == nil {
		return fmt.Errorf("%s requires a lister", PluginName)
	}
	return nil
}

func (f *flowControlPlugin) SetInternalKubeClientSet(client internalclientset.Interface) {
	f.client = client
}

func (f *flowControlPlugin) SetInternalKubeInformerFactory(factory informers.SharedInformerFactory) {
	cmInformer := factory.Core().InternalVersion().ConfigMaps()
	f.lister = cmInformer.Lister()
	f.SetReadyFunc(cmInformer.Informer().HasSynced)
}

// Validate rejects a pod deletion request if pods are deleted too quick
func (f *flowControlPlugin) Validate(a admission.Attributes) error {
	// Ignore all calls to subresources or resources other than pods.
	// Ignore all operations other than Delete.
	if len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() != api.Resource("pods") || a.GetOperation() != admission.Delete {
		return nil
	}

	dontCount, err := f.ignoreKubeletDeletion(a)
	if err != nil {
		return fmt.Errorf("ignore kubelet deletion failed, err: %v", err)
	}
	if dontCount {
		return nil
	}

	if err := f.initNamespacedCounter(GlobalPdfcNamespace); err != nil {
		return fmt.Errorf("init global counter failed, err: %v", err)
	}
	if err := f.initNamespacedCounter(a.GetNamespace()); err != nil {
		return fmt.Errorf("init app counter failed, err: %v", err)
	}

	if !tryAdmit(a.GetNamespace()) {
		return fmt.Errorf("%v/%v is rejected by flow control", a.GetNamespace(), a.GetName())
	}

	return nil
}

func (f *flowControlPlugin) ignoreKubeletDeletion(a admission.Attributes) (bool, error) {
	glog.V(5).Infof("delete pod request is sent by %v", a.GetUserInfo().GetName())

	cm, err := f.lister.ConfigMaps(GlobalPdfcNamespace).Get(PdfcConfigName)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.V(5).Infof("there is no pdfc rule in namespace %v", GlobalPdfcNamespace)
			return false, nil
		}
		return false, fmt.Errorf("listing pdfc failed: %v", err)
	}

	users := strings.Split(cm.Data[PdfcConfigWhiteListKey], ",")
	if len(users) == 1 && users[0] == "" {
		return false, nil
	}
	for _, user := range users {
		if strings.Contains(a.GetUserInfo().GetName(), user) {
			return true, nil
		}
	}
	return false, nil
}

func (f *flowControlPlugin) initNamespacedCounter(namespace string) error {
	curTime := time.Now()
	if debugMode {
		curTime = debugFakeTime
	}

	countersLock.Lock()
	defer countersLock.Unlock()

	_, ok := namespacedCounters[namespace]
	if ok {
		return nil
	}

	cm, err := f.lister.ConfigMaps(namespace).Get(PdfcConfigName)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.V(5).Infof("there is no pdfc rule in namespace %v", namespace)
			return nil
		}
		return fmt.Errorf("listing pdfc failed: %v", err)
	}

	counters, err := initCounters(cm, curTime)
	if err != nil {
		return err
	}
	rulesJson[namespace] = cm.Data[PdfcConfigRuleKey]

	// happens once when apiserver startup
	if firstCounters {
		go wait.Until(f.checkFlowControlRules, defaultCheckPeriod, wait.NeverStop)
		go wait.Until(f.updateCacheAndStorage, defaultUpdatePeriod, wait.NeverStop)
		go wait.Until(f.cleanupStaleRecord, defaultCleanupPeriod, wait.NeverStop)
		firstCounters = false
	}

	namespacedCounters[namespace] = counters
	return nil
}

func transferTimeToInt(curTime time.Time) (int64, error) {
	strTime := time.Unix(curTime.Unix(), 0).Format("200601021504")
	intTime, err := strconv.ParseInt(strTime, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %v to int failed, err: %v", strTime, err)
	}
	return intTime, nil
}

func initCounters(cm *api.ConfigMap, curTime time.Time) ([]*Counter, error) {
	var rules []Rule
	if err := json.Unmarshal([]byte(cm.Data[PdfcConfigRuleKey]), &rules); err != nil {
		return nil, fmt.Errorf("unmarshal %v failed, err: %v", cm.Data[PdfcConfigRuleKey], err)
	}
	records := make(map[int64]*Record)
	if cm.Data[PdfcConfigRecordKey] != "" {
		if err := json.Unmarshal([]byte(cm.Data[PdfcConfigRecordKey]), &records); err != nil {
			return nil, fmt.Errorf("unmarshal %v failed, err: %v", cm.Data[PdfcConfigRecordKey], err)
		}
	}

	var counters []*Counter
	for _, rule := range rules {
		duration, err := time.ParseDuration(rule.Duration)
		if err != nil {
			return nil, err
		}
		if duration.Minutes() < 1.0 || duration.Hours() > 24.0 {
			return nil, fmt.Errorf("rule duration %v is invalid, out of range [1m, 24h]", rule.Duration)
		}

		winLen := int(duration.Minutes())
		counter := Counter{
			deleteLimit: rule.DeleteLimit,
			window:      make([]int, winLen),
		}

		curTime, err := transferTimeToInt(curTime)
		if err != nil {
			return nil, err
		}
		for i := 1; i <= winLen; i++ {
			if records[curTime] != nil {
				counter.window[winLen-i] = records[curTime].DeleteCount
				counter.totalCount += records[curTime].DeleteCount
			}
			curTime--
		}

		counters = append(counters, &counter)
	}
	return counters, nil
}

func tryAdmit(namespace string) bool {
	countersLock.Lock()
	defer countersLock.Unlock()

	var counters []*Counter
	counters = append(counters, namespacedCounters[GlobalPdfcNamespace]...)
	if namespace != GlobalPdfcNamespace {
		counters = append(counters, namespacedCounters[namespace]...)
	}
	for _, counter := range counters {
		if counter.totalCount >= counter.deleteLimit {
			return false
		}
	}

	for _, counter := range counters {
		counter.totalCount++
		counter.window[len(counter.window)-1]++
	}
	return true
}

func (f *flowControlPlugin) checkFlowControlRules() {
	curTime := time.Now()
	if debugMode {
		<-needUpdate
		curTime = debugFakeTime
	}

	countersLock.Lock()
	defer countersLock.Unlock()

	for namespace, _ := range namespacedCounters {
		cm, err := f.lister.ConfigMaps(namespace).Get(PdfcConfigName)
		if err != nil {
			if errors.IsNotFound(err) {
				glog.V(3).Infof("pod deletion flow control rules doesn't exist anymore")
				delete(namespacedCounters, namespace)
				delete(rulesJson, namespace)
			} else {
				glog.Errorf("get %v/%v failed, err: %v", namespace, PdfcConfigName)
			}
			continue
		}

		if cm.Data[PdfcConfigRuleKey] != rulesJson[namespace] {
			glog.V(3).Infof("pod deletion flow control rules changed")

			counters, err := initCounters(cm, curTime)
			if err != nil {
				glog.Error(err)
				continue
			}

			rulesJson[namespace] = cm.Data[PdfcConfigRuleKey]
			namespacedCounters[namespace] = counters
		}
	}
}

// update local cache and configMap in etcd every minute
func (f *flowControlPlugin) updateCacheAndStorage() {
	curTime := time.Now()
	if debugMode {
		<-oneMinutePass
		debugFakeTime = debugFakeTime.Add(time.Minute)
		curTime = debugFakeTime
	}
	// wait until second becomes zero
	if curTime.Second() != 0 {
		waitDuration := time.Duration(60 - curTime.Second())
		time.Sleep(waitDuration * time.Second)
		curTime = curTime.Add(waitDuration * time.Second)
	}

	countersLock.Lock()
	defer countersLock.Unlock()

	for namespace, counters := range namespacedCounters {
		if len(counters) == 0 {
			continue
		}

		window := counters[0].window
		lastMinuteCount := window[len(window)-1]
		for _, counter := range counters {
			counter.totalCount -= counter.window[0]
			for i := 0; i < len(counter.window)-1; i++ {
				counter.window[i] = counter.window[i+1]
			}
			counter.window[len(counter.window)-1] = 0
		}
		if lastMinuteCount == 0 {
			continue
		}

		// REASON:
		// multiple apiserver will update pdfc on the minute
		// try three times to ensure success rate
		retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// reduce competition rate
			if !debugMode {
				time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
			}

			err := f.updateStorage(namespace, curTime, lastMinuteCount)
			return err
		})
		if retryErr != nil {
			glog.Errorf("update etcd pdfc records failed, err: %v", retryErr)
		} else {
			glog.V(4).Infof("update etcd pdfc records successfully")
		}
	}
}

func (f *flowControlPlugin) updateStorage(namespace string, curTime time.Time, lastMinuteCount int) error {
	cm, err := f.client.Core().ConfigMaps(namespace).Get(PdfcConfigName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get %v/%v failed, err: %v", namespace, PdfcConfigName, err)
	}

	records := make(map[int64]*Record)
	if cm.Data[PdfcConfigRecordKey] != "" {
		if err := json.Unmarshal([]byte(cm.Data[PdfcConfigRecordKey]), &records); err != nil {
			return fmt.Errorf("unmarshal %v failed, err: %v", cm.Data[PdfcConfigRecordKey], err)
		}
	}
	intTime, err := transferTimeToInt(curTime)
	if err != nil {
		return err
	}
	prevTime := intTime - 1
	if _, ok := records[prevTime]; ok {
		records[prevTime].DeleteCount += lastMinuteCount
	} else {
		records[prevTime] = &Record{DeleteCount: lastMinuteCount}
	}
	recordsByte, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal %v failed, err: %v", records, err)
	}

	newCm := cm.DeepCopy()
	newCm.Data[PdfcConfigRecordKey] = string(recordsByte)
	_, err = f.client.Core().ConfigMaps(namespace).Update(newCm)
	return err
}

// cleanup stale deletion record(ttl: 24h) periodically(default: hour)
func (f *flowControlPlugin) cleanupStaleRecord() {
	curTime := time.Now()
	if debugMode {
		<-needClean
		curTime = debugFakeTime
	}

	countersLock.Lock()
	defer countersLock.Unlock()

	for namespace, counter := range namespacedCounters {
		if len(counter) == 0 {
			continue
		}

		cm, err := f.client.Core().ConfigMaps(namespace).Get(PdfcConfigName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("get %v/%v failed, err: %v", namespace, PdfcConfigName, err)
			continue
		}

		records := make(map[int64]*Record)
		if cm.Data[PdfcConfigRecordKey] != "" {
			if err := json.Unmarshal([]byte(cm.Data[PdfcConfigRecordKey]), &records); err != nil {
				glog.Errorf("%v", err)
			}
		}
		curTime, err := transferTimeToInt(curTime)
		if err != nil {
			glog.Error(err)
			continue
		}
		newRecords := make(map[int64]*Record)
		for deleteTime, record := range records {
			if curTime-deleteTime < maxTimeWindow {
				newRecords[deleteTime] = record
			}
		}
		if len(newRecords) == len(records) {
			continue
		}
		recordsByte, err := json.Marshal(newRecords)
		if err != nil {
			glog.Errorf("marshal %v failed, err: %v", records, err)
			continue
		}

		newCm := cm.DeepCopy()
		newCm.Data[PdfcConfigRecordKey] = string(recordsByte)
		_, err = f.client.Core().ConfigMaps(namespace).Update(newCm)
		if err != nil {
			glog.Errorf("update %v/%v failed, err: %v", namespace, PdfcConfigName, err)
		}
	}
}
