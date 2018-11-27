package viputil

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"math/rand"
	"net/http"
	netUrl "net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	supportedSignAlgorithm = "KeyCenter_SHA"
	supportedHashType      = "SHA256"

	appNum       = "47e743243fc446e8bf7f272181d01939"
	keyName      = "test-vip"
	keyCenterUrl = "http://daily.keycenter.alibaba.net/keycenter"
	vipUrl       = "http://xvip.alibaba-inc.com"
	vipSysName   = "sigma-k8s-controller"
	vipPe        = "shouchen.zz"

	vipStatusOK        = "OK"
	vipTaskStatusReady = "ready"
	vipTaskStatusProc  = "proc"
	vipTaskStatusSucc  = "succ"
	vipTaskStatusFail  = "fail"
)

var (
	client    = &http.Client{Timeout: time.Second * 5}
	keyCenter *KeyCenter
)

type KeyCenter struct {
	keys map[string]KeyInfo
}

type KeyResp struct {
	Data KeyData `json:"data"`
}

type KeyData struct {
	Object KeyObject `json:"object"`
}

type KeyObject struct {
	KeyRefMap map[string]SecData `json:"keyRefMap"`
}

type SecData struct {
	Name    string `json:"name"`
	KeyHead Head   `json:"keyHead"`
}

type Head struct {
	CurrentVersion string             `json:"currentVersion"`
	KeyVersionList map[string]KeyInfo `json:"keyVersionList"`
}

type KeyInfo struct {
	Algorithm       string   `json:"algorithm"`
	Content         string   `json:"content"`
	WorkingMetaData MetaData `json:"workingMetaData"`
}

type MetaData struct {
	HashType string `json:"hashType"`
}

func NewKeyCenter(url, appNum string) (*KeyCenter, error) {
	reqPara := fmt.Sprintf(`{"data": {"appNum": "%v"}, "clientInfo": {"language": "Go"}}`, appNum)
	encReqPara := base64.StdEncoding.EncodeToString([]byte(reqPara))
	reqBody := fmt.Sprintf("action=queryApplication&data=%v", encReqPara)

	code, body, err := doRequest(url, reqBody)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("get keys failed, code: %v, body: %v", code, body)
	}

	var keyResp KeyResp
	if err := json.Unmarshal([]byte(body), &keyResp); err != nil {
		return nil, err
	}
	keyList := keyResp.Data.Object

	kc := KeyCenter{
		keys: make(map[string]KeyInfo),
	}
	for keyName, keyData := range keyList.KeyRefMap {
		currentVersion := keyData.KeyHead.CurrentVersion
		kc.keys[keyName] = keyData.KeyHead.KeyVersionList[currentVersion]
	}
	return &kc, nil
}

func (kc *KeyCenter) sign(keyName, rawStr string) (string, error) {
	key, ok := kc.keys[keyName]
	if !ok {
		return "", fmt.Errorf("get key failed, keyname %v not found", keyName)
	}

	if key.Algorithm != supportedSignAlgorithm {
		return "", fmt.Errorf("unsupported sign algorithm: %v", key.Algorithm)
	}

	if key.WorkingMetaData.HashType != supportedHashType {
		return "", fmt.Errorf("unsupported sign hash type: %v", key.WorkingMetaData.HashType)
	}

	var h hash.Hash = sha256.New()
	h.Write([]byte(rawStr))

	tmp, err := base64.StdEncoding.DecodeString(key.Content)
	if err != nil {
		return "", fmt.Errorf("key content is error, err: %v", err)
	}
	keyContent, err := base64.StdEncoding.DecodeString(string(tmp))
	if err != nil {
		return "", fmt.Errorf("key content is error, err: %v", err)
	}
	h.Write([]byte(keyContent))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func doRequestWithSignature(url, path string, reqPara map[string]string) (code int, respBody string, err error) {
	reqPara["sysname"] = vipSysName
	reqPara["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	reqPara["nonce"] = strconv.Itoa(rand.Intn(65535))

	reqBody := genNormalizedPara(reqPara)
	rawStr := path + "?" + reqBody
	if keyCenter == nil {
		keyCenter, err = NewKeyCenter(keyCenterUrl, appNum)
		if err != nil {
			return 0, "", fmt.Errorf("init key center failed, err: %v", err)
		}
	}
	signature, err := keyCenter.sign(keyName, rawStr)
	if err != nil {
		return 0, "", fmt.Errorf("sign %v failed, err: %v", rawStr, err)
	}

	url += path
	reqBody = strings.TrimPrefix(reqBody+"&signature="+netUrl.QueryEscape(signature), path+"?")

	return doRequest(url, reqBody)
}

func doRequest(url, reqBody string) (code int, respBody string, err error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		return 0, "", fmt.Errorf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("send http request failed: %v", err)
	}
	defer resp.Body.Close()

	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read response failed: %v", err)
	}
	return resp.StatusCode, string(bodyByte), nil
}

func genNormalizedPara(reqPara map[string]string) (norPara string) {
	keys := make([]string, len(reqPara))
	i := 0
	for key := range reqPara {
		keys[i] = key
		i++
	}
	sort.Strings(keys)

	for _, key := range keys {
		norPara = norPara + key + "=" + reqPara[key] + "&"
	}
	return strings.TrimSuffix(norPara, "&")
}

type GetRsResponse struct {
	Data    []Backend `json:"data"`
	ErrCode string    `json:"errCode"`
	ErrMsg  string    `json:"errMsg"`
}

type UpdateVipResponse struct {
	ErrCode string `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}

type GetTaskResponse struct {
	Data    TaskResult `json:"data"`
	ErrCode string     `json:"errCode"`
	ErrMsg  string     `json:"errMsg"`
}

// backend is rs
type Backend struct {
	IP   string `json:"rsIp"`
	Port int    `json:"rsPort"`
}

type RsChange struct {
	OP     string `json:"op"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

type TaskResult struct {
	Status string `json:"status"`
	Info   string `json:"info"`
}

func GetRsOfVs(ip string, port int, protocol string) ([]Backend, error) {
	data := map[string]string{
		"ip":       ip,
		"port":     strconv.Itoa(port),
		"protocol": protocol,
	}

	code, body, err := doRequestWithSignature(vipUrl, "/xvip/api/getRsOfVs", data)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("code: %v, body: %v", code, body)
	}

	var rsResp GetRsResponse
	if err := json.Unmarshal([]byte(body), &rsResp); err != nil {
		return nil, err
	}
	if rsResp.ErrCode != vipStatusOK {
		return nil, fmt.Errorf("errCode: %v, errMsg: %v", rsResp.ErrCode, rsResp.ErrMsg)
	}
	return rsResp.Data, nil
}

func CleanupVip(ip string, port int, protocol string) error {
	backends, err := GetRsOfVs(ip, port, protocol)
	if err != nil {
		return err
	}

	if len(backends) == 0 {
		return nil
	}

	if err := removeRsFromVip(ip, port, protocol, backends); err != nil {
		return err
	}
	return nil
}

func removeRsFromVip(vip string, vport int, vprotocol string, backends []Backend) error {
	rsChange, err := getRsChange(backends, "DELETE")
	if err != nil {
		return err
	}

	orderId := uuid.NewUUID()
	if err := updateVip(vipPe, vip, vport, vprotocol, rsChange, string(orderId)); err != nil {
		return err
	}

	if err := getTaskInfo(string(orderId)); err != nil {
		return err
	}
	return nil
}

func getRsChange(backends []Backend, op string) (string, error) {
	var changes []RsChange
	if op == "ADD" {
		for _, rs := range backends {
			changes = append(changes, RsChange{
				OP:     op,
				IP:     rs.IP,
				Port:   rs.Port,
				Status: "enable",
			})
		}
	} else if op == "DELETE" {
		for _, rs := range backends {
			changes = append(changes, RsChange{
				OP:   op,
				IP:   rs.IP,
				Port: rs.Port,
			})
		}
	}

	changeByte, err := json.Marshal(changes)
	if err != nil {
		return "", err
	}
	return string(changeByte), nil
}

func updateVip(applyUser, ip string, port int, protocol, rs, changeOrderId string) error {
	data := map[string]string{
		"applyUser":     applyUser,
		"ip":            ip,
		"port":          strconv.Itoa(port),
		"protocol":      protocol,
		"rs":            rs,
		"changeOrderId": changeOrderId,
	}

	code, body, err := doRequestWithSignature(vipUrl, "/xvip/api/updateVip", data)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("code: %v, body: %v", code, body)
	}

	var updateResp UpdateVipResponse
	if err := json.Unmarshal([]byte(body), &updateResp); err != nil {
		return err
	}
	if updateResp.ErrCode != vipStatusOK {
		return fmt.Errorf("errCode: %v, errMsg: %v", updateResp.ErrCode, updateResp.ErrMsg)
	}
	return nil
}

func getTaskInfo(changeOrderId string) error {
	data := map[string]string{"changeOrderId": changeOrderId}
	var taskResp GetTaskResponse

	// 200ms/400ms/800ms/1.6s/3.2s
	checkPeriod := 200 * time.Millisecond
	for i := 0; i < 5; i++ {
		time.Sleep(checkPeriod)
		checkPeriod *= 2

		code, body, err := doRequestWithSignature(vipUrl, "/xvip/api/getTaskInfo", data)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("code: %v, body: %v", code, body)
		}

		if err := json.Unmarshal([]byte(body), &taskResp); err != nil {
			return err
		}
		if taskResp.ErrCode != vipStatusOK {
			return fmt.Errorf("errCode: %v, errMsg: %v", taskResp.ErrCode, taskResp.ErrMsg)
		}

		switch taskResp.Data.Status {
		case vipTaskStatusReady, vipTaskStatusProc:
			continue
		case vipTaskStatusFail:
			return fmt.Errorf("info: %v", taskResp.Data.Info)
		case vipTaskStatusSucc:
			return nil
		default:
			return fmt.Errorf("unexpected task status: %v", taskResp.Data.Status)
		}
	}
	return fmt.Errorf("timeout waitting for result")
}

func WaitUntilBackendsCorrect(podList *v1.PodList, vip string, vport int, vprotocol string, checkPeriod, timeout time.Duration) error {
	return wait.PollImmediate(checkPeriod, timeout, checkBackends(podList, vip, vport, vprotocol))
}

func checkBackends(pods *v1.PodList, vip string, vport int, vprotocol string) wait.ConditionFunc {
	return func() (bool, error) {
		backends, err := GetRsOfVs(vip, vport, vprotocol)
		if err != nil {
			return false, err
		}

		if backendsIsCorrect(pods, backends) {
			return true, nil
		}
		return false, nil
	}
}

func backendsIsCorrect(podList *v1.PodList, backends []Backend) bool {
	// filter out unready pods
	var pods []*v1.Pod
	if podList != nil {
		for _, pod := range podList.Items {
			if pod.Status.ContainerStatuses[0].Ready == true {
				pods = append(pods, &pod)
			}
		}
	}

	framework.Logf("len: %v/%v", len(pods), len(backends))
	if len(pods) != len(backends) {
		return false
	}

	ipInDomain := make(map[string]bool)
	for _, addr := range backends {
		ipInDomain[addr.IP] = true
	}
	framework.Logf("current backends: %v", ipInDomain)
	for _, pod := range pods {
		if !ipInDomain[pod.Status.PodIP] {
			return false
		}
	}
	return true
}
