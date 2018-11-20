package cmdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	//"github.com/golang/glog"
)

type ContainerInfo struct {
	GMTCreate         int64  `json:"gmtCreate"`
	GMTModified       int64  `json:"gmtModified"`
	NodeSn            string `json:"ncSn"`
	AllocPlanStatus   string `json:"allocplanStatus"`
	InstanceStatus    string `json:"instanceStatus"`
	InstanceType      string `json:"instanceType"`
	ContainerId       string `json:"containerId"`
	ContainerSn       string `json:"containerSn"`
	ContainerIp       string `json:"containerIp"`
	ContainerHostName string `json:"containerHostname"`
	BizName           string `json:"bizName"`
	AppName           string `json:"appName"`
	DeployUnit        string `json:"deployUnit"`
	CpuNum            int64  `json:"cpuNum"`
	CpuIds            string `json:"cpus"`
	DiskSize          int64  `json:"diskSizeB"`
	MemorySize        int64  `json:"memorySizeB"`
	PoolSystem        string `json:"poolSystem"` //sigma3_1
}

type CMDBResp struct {
	Code    int            `json:"code"`
	Data    *ContainerInfo `json:"data"`
	Info    string         `json:"info"`
	Success bool           `json:"success"`
}

type CMDBClient struct {
	Url   string
	User  string
	Token string
}

func NewCMDBClient(url, user, token string) Client {
	return &CMDBClient{
		Url:   url,
		User:  user,
		Token: token,
	}
}

//AddContainerInfo() add new container into cmdb.
func (c *CMDBClient) AddContainerInfo(reqInfo []byte) error {
	requestUrl := fmt.Sprintf("%v/open-api/meta/container/add", c.Url)
	//glog.V(5).Infof("Method:%v, URL:%v", http.MethodPost, requestUrl)
	data := url.Values{"containerParams": {string(reqInfo)}}
	req, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		//glog.Errorf("Init new post request failed, err: %v", err)
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doHttpRequest(req)
	if err != nil {
		//glog.Errorf("Add container into cmdb failed, req:%v, err:%v", string(reqInfo), err)
		return err
	}
	if resp.Code != 200 || !resp.Success {
		//glog.Errorf("Add container into cmdb failed, resp:%v, req:%v", *resp, string(reqInfo))
		return fmt.Errorf("Add container into cmdb failed, resp:%v, req:%v", *resp, string(reqInfo))
	}
	return nil
}

//UpdateContainerInfo() update container into cmdb.
func (c *CMDBClient) UpdateContainerInfo(reqInfo []byte) error {
	boosUrl := fmt.Sprintf("%v/open-api/meta/container/updateByContainerSn", c.Url)
	//glog.V(5).Infof("Method:%v, URL:%v", http.MethodPost, boosUrl)
	data := url.Values{"containerParams": {string(reqInfo)}}
	req, err := http.NewRequest(http.MethodPost, boosUrl, strings.NewReader(data.Encode()))
	if err != nil {
		//glog.Errorf("Init new update request failed, err: %v", err)
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doHttpRequest(req)
	if err != nil {
		//glog.Errorf("Update cmdb failed, req:%v, err:%v", string(reqInfo), err)
		return err
	}
	if resp.Code != 200 || !resp.Success {
		//glog.Errorf("Update cmdb failed, resp:%v, req:%v", *resp, string(reqInfo))
		return fmt.Errorf("Update cmdb failed, resp:%v, req:%v", *resp, string(reqInfo))
	}
	return nil
}

//UpdateContainerInfo() update container into cmdb.
func (c *CMDBClient) DeleteContainerInfo(sn string) error {
	bossUrl := fmt.Sprintf("%v/open-api/meta/container/deleteByContainerSn?containerSn=%v", c.Url, sn)
	//glog.V(5).Infof("Method:%v, URL:%v", http.MethodPost, bossUrl)

	req, err := http.NewRequest(http.MethodPost, bossUrl, nil)
	if err != nil {
		//glog.Errorf("Init new delete request failed, err: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doHttpRequest(req)
	if err != nil {
		//glog.Errorf("Delete container %v from cmdb failed, err:%v", sn, err)
		return err
	}
	if resp.Code != 200 || !resp.Success {
		//glog.Errorf("Delete container %v from cmdb failed, resp:%v", sn, *resp)
		return fmt.Errorf("Delete container %v from cmdb failed, resp:%v", sn, *resp)
	}
	return nil
}

//GetContainerInfo() get container info from cmdb.
func (c *CMDBClient) GetContainerInfo(sn string) (*CMDBResp, error) {
	bossUrl := fmt.Sprintf("%v/open-api/meta/container/getByContainerSn?containerSn=%v", c.Url, sn)
	//glog.V(5).Infof("Method:%v, URL:%v", http.MethodPost, bossUrl)

	req, err := http.NewRequest(http.MethodGet, bossUrl, nil)
	if err != nil {
		//glog.Errorf("Init new post request failed, err: %v", err)
		return nil, err
	}

	cmdbResp, err := c.doHttpRequest(req)
	if err != nil {
		//glog.Errorf("get container %v info failed, %v", sn, err)
		return nil, err
	}

	return cmdbResp, nil
}

func (c *CMDBClient) doHttpRequest(req *http.Request) (*CMDBResp, error) {
	req.Header.Add("x-apiauth-name", c.User)
	req.Header.Add("x-apiauth-token", c.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//glog.Errorf("Send request failed, err:%v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//glog.Errorf("Read response body failed, err:%v", err)
		return nil, err
	}
	cmdbResp := &CMDBResp{}
	err = json.Unmarshal(body, cmdbResp)
	if err != nil {
		//glog.Errorf("Unmarshal response body failed, err:%v", err)
		return nil, err
	}
	return cmdbResp, nil
}
