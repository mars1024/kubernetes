package ant_sigma_bvt

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/samalba/dockerclient"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

type AdapterServer struct {
	// AlipayCeritficatePath sigma2.0 certificate file path.
	AlipayCeritficatePath string

	// AdapterAddress adapter address, e.g. eu95.alipay.net:xxxx
	AdapterAddress string
}

type CreateResp struct {
	Id         string                       `json:"Id"`
	Warnings   []string                     `json:"Warnings"`
	Containers map[string]map[string]string `json:"Containers"`
}

//NewHttpClient() init adapter client.
func (s *AdapterServer) NewHttpClient() (*http.Client, error) {
	client := &http.Client{}
	pool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(s.AlipayCeritficatePath + "/ca.pem")
	if err != nil {
		glog.Errorf("read Ca cert file failed, err:%v", err)
		return client, err
	}
	pool.AppendCertsFromPEM(ca)

	cli, err := tls.LoadX509KeyPair(s.AlipayCeritficatePath+"/cert.pem", s.AlipayCeritficatePath+"/key.pem")
	if err != nil {
		glog.Errorf("Load client key pair failed, err:%v", err)
		return client, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{cli},
			InsecureSkipVerify: true,
		},
	}
	client.Transport = tr
	return client, nil
}

//GetContainer() get containerInfo.
func (s *AdapterServer) GetContainer(name, sn string) (*dockertypes.ContainerJSON, string, error) {
	url := fmt.Sprintf("https://%v/containers/%v/json", s.AdapterAddress, name)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodGet, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return nil, "", err
	}

	if sn != "" {
		params := req.URL.Query()
		params.Add("sn", sn)
		req.URL.RawQuery = params.Encode()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return nil, "", err
	}
	rawJSON := &dockertypes.ContainerJSON{}
	rawJSON.ContainerJSONBase = &dockertypes.ContainerJSONBase{}
	rawJSON.HostConfig = &container.HostConfig{}
	rawJSON.Config = &container.Config{}
	rawJSON.NetworkSettings = &dockertypes.NetworkSettings{}
	if resp.StatusCode == http.StatusOK {
		err = json.Unmarshal(body, rawJSON)
		if err != nil {
			glog.Errorf("Unmarshal response body failed, err:%v", err)
			return rawJSON, "", err
		}
	} else {
		return rawJSON, string(body), nil
	}
	return rawJSON, "", nil
}

//GetAsyncJson() get async request result from etcd.
func (s *AdapterServer) GetAsyncJson(requestId string) (*swarm.Request, string, error) {
	url := fmt.Sprintf("https://%v/sigma/requests/%v/json", s.AdapterAddress, requestId)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodGet, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return nil, "", err
	}

	task := &swarm.Request{}

	if resp.StatusCode == http.StatusOK {
		err = json.Unmarshal(body, task)
		if err != nil {
			glog.Errorf("Unmarshal response body failed, err:%v", err)
			return task, "", err
		}
	} else {
		return task, string(body), nil
	}
	return task, "", nil
}

//MustCreatePod() create pods and wait until pods is ready.
func MustCreatePod(s *AdapterServer, client clientset.Interface, c *dockerclient.ContainerConfig) (v1.Pod, *swarm.AllocResult) {
	reqInfo, err := json.Marshal(c)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle]marshal ReqInfo failed.")
	createResp, message, err := s.CreateContainer(reqInfo)
	framework.Logf("Create resp, message:%v, err:%v, site:%v", message, err, site)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle]Create pod error.")
	Expect(message).To(Equal(""), "[AdapterLifeCycle]create pod failed.")
	Expect(createResp).NotTo(BeNil(), "[AdapterLifeCycle]get create response failed.")
	Expect(createResp.Id).NotTo(BeEmpty(), "[AdapterLifeCycle]get requestId failed.")
	By("Get sigma-adapter create async response.")
	result, err := GetCreateResultWithTimeOut(client, createResp.Id, 3*time.Minute, c.Labels["ali.AppName"])
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get async response failed.")
	Expect(result).NotTo(BeNil(), "[AdapterLifeCycle] result should not be nil.")
	By("Get created pod.")
	pods, err := GetPodLists(client, "ali.RequestId", createResp.Id, c.Labels["ali.AppName"])
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get pod list failed.")
	Expect(len(pods)).To(Equal(1), "[AdapterLifeCycle] Bad pod number.")
	testPod := pods[0]
	return testPod, result
}

//CreateContainer() create container.
func (s *AdapterServer) CreateContainer(reqInfo []byte) (*CreateResp, string, error) {
	url := fmt.Sprintf("https://%v/containers/create", s.AdapterAddress)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodPost, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqInfo))
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return nil, "", err
	}

	createResp := &CreateResp{}
	if resp.StatusCode == http.StatusAccepted {
		err = json.Unmarshal(body, createResp)
		if err != nil {
			glog.Errorf("Unmarshal response body failed, err:%v", err)
			return createResp, "", err
		}
	} else {
		return createResp, string(body), nil
	}
	return createResp, "", nil
}

//MustOperatePod()  start/stop/restart container.
func MustOperatePod(s *AdapterServer, client clientset.Interface, sn string, pod *v1.Pod, action string, st v1.PodPhase) {
	resp, err := s.OperateContainer(sn, action)
	Expect(err).To(BeNil(), fmt.Sprintf("[AdapterLifeCycle] %s container failed.", action))
	Expect(resp).To(BeEmpty(), fmt.Sprintf("[AdapterLifeCycle] %s container failed with response.", action))
	err = util.WaitTimeoutForPodStatus(client, pod, st, 1*time.Minute)
	Expect(resp).To(BeEmpty(), fmt.Sprintf("[AdapterLifeCycle] %s container failed after 1 min..", action))
}

//OperateContainer if ok return 204. action : start/stop/restart
func (s *AdapterServer) OperateContainer(name, action string) (string, error) {
	url := fmt.Sprintf("https://%v/containers/%v/%v", s.AdapterAddress, name, action)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodPost, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return "", err
	}

	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	} else {
		return string(body), nil
	}
	return "", nil
}

//DeleteContainer() if ok return 204.
func (s *AdapterServer) DeleteContainer(name string, force bool) (string, error) {
	url := fmt.Sprintf("https://%v/containers/%v", s.AdapterAddress, name)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodDelete, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return "", err
	}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return "", err
	}
	if force {
		params := req.URL.Query()
		params.Add("force", "true")
		req.URL.RawQuery = params.Encode()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return "", err
	}

	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	} else {
		return string(body), nil
	}
	return "", nil
}

//MustUpgradeContainer() upgrade container, wait pod ready.
func MustUpgradeContainer(s *AdapterServer, name, requestId string, nostart bool, c *dockerclient.ContainerConfig) {
	reqInfo, err := json.Marshal(c)
	framework.Logf("ReqInfo:%v", string(reqInfo))
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle]marshal upgrade ReqInfo failed.")
	upgradeResp, message, err := s.UpgradeContainer(name, requestId, nostart, reqInfo)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] upgrade pod error.")
	Expect(message).To(Equal(""), "[AdapterLifeCycle] upgrade pod failed.")
	Expect(upgradeResp).NotTo(BeNil(), "[AdapterLifeCycle] get upgrade response failed.")
	Expect(upgradeResp.Id).NotTo(BeEmpty(), "[AdapterLifeCycle] get upgrade requestId failed.")
	By("Get sigma-adapter upgrade async response.")
	isOk, err := GetUpgradeResultWithTimeOut(upgradeResp.Id, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get async response failed.")
	Expect(isOk).To(BeTrue(), "[AdapterLifeCycle] get upgrade result should not be ok.")

}

//UpgradeContainer() upgrade container.
func (s *AdapterServer) UpgradeContainer(name, requestId string, nostart bool, reqInfo []byte) (*CreateResp, string, error) {
	url := fmt.Sprintf("https://%v/containers/%s/upgrade", s.AdapterAddress, name)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodPost, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqInfo))
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return nil, "", err
	}

	params := req.URL.Query()
	params.Add("requestId", requestId)
	params.Add("async", "true")
	if nostart {
		params.Add("nostart", "true")
	}
	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return nil, "", err
	}

	upgradeResp := &CreateResp{}
	if resp.StatusCode == http.StatusAccepted {
		err = json.Unmarshal(body, upgradeResp)
		if err != nil {
			glog.Errorf("Unmarshal response body failed, err:%v", err)
			return upgradeResp, "", err
		}
	} else {
		return upgradeResp, string(body), nil
	}
	return upgradeResp, "", nil
}

//UpdateContainer() update container.
func (s *AdapterServer) UpdateContainer(name string, reqInfo []byte) (*CreateResp, string, error) {
	url := fmt.Sprintf("https://%v:%v/containers/%s/update", s.AdapterAddress, name)
	glog.V(2).Infof("Method:%v, URL:%v", http.MethodPost, url)
	client, err := s.NewHttpClient()
	if err != nil {
		glog.Errorf("Init http client failed, err:%v", err)
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqInfo))
	if err != nil {
		glog.Errorf("Init new request failed, err: %v", err)
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Request failed, err:%v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Read response body failed, err:%v", err)
		return nil, "", err
	}

	updateResp := &CreateResp{}
	if resp.StatusCode == http.StatusOK {
		err = json.Unmarshal(body, updateResp)
		if err != nil {
			glog.Errorf("Unmarshal response body failed, err:%v", err)
			return updateResp, "", err
		}
	} else {
		return updateResp, string(body), nil
	}
	return updateResp, "", nil
}
