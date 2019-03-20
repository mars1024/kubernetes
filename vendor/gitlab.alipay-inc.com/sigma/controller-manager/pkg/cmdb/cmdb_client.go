package cmdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"unsafe"

	"github.com/golang/glog"
)

type ContainerInfo struct {
	GMTCreate         int64             `json:"gmtCreate"`
	GMTModified       int64             `json:"gmtModified"`
	NodeSn            string            `json:"ncSn"`
	AllocPlanStatus   string            `json:"allocplanStatus"`
	InstanceStatus    string            `json:"instanceStatus"`
	InstanceType      string            `json:"instanceType"`
	ContainerId       string            `json:"containerId"`
	ContainerSn       string            `json:"containerSn"`
	ContainerIp       string            `json:"containerIp"`
	ContainerHostName string            `json:"containerHostname"`
	BizName           string            `json:"bizName"`
	AppName           string            `json:"appName"`
	DeployUnit        string            `json:"deployUnit"`
	CpuNum            int64             `json:"cpuNum"`
	CpuIds            string            `json:"cpus"`
	DiskSize          int64             `json:"diskSizeB"`
	MemorySize        int64             `json:"memorySizeB"`
	PoolSystem        string            `json:"poolSystem"` //sigma3_1
	Labels            map[string]string `json:"labels"`
}

type CMDBResp struct {
	Code    int            `json:"code"`
	Data    *ContainerInfo `json:"data"`
	Info    string         `json:"info"`
	Success bool           `json:"success"`
}

type CMDBClient struct {
	URL   string
	User  string
	Token string
}

func NewCMDBClient(addr, user, token string) Client {
	return &CMDBClient{
		URL:   addr,
		User:  user,
		Token: token,
	}
}

// AddContainerInfo() add new container into cmdb.
func (c *CMDBClient) AddContainerInfo(reqInfo []byte) error {
	addrs := strings.Split(c.URL, ",")
	if len(addrs) == 0 {
		glog.Errorf("[CMDBClient] CMDB url must be specified, url:%v", c.URL)
		return fmt.Errorf("cmdb url must be specified, url:%v", c.URL)
	}
	for _, addr := range addrs {
		err := c.AddOneContainerInfo(reqInfo, addr)
		if err != nil {
			glog.Errorf("[CMDBClient] Add container info into cmdb %v failed, info:%v, err:%v", addr, string(reqInfo), err)
			return err
		}
	}
	return nil
}

// AddOneContainerInfo() add container info into one cmdb.
func (c *CMDBClient) AddOneContainerInfo(reqInfo []byte, addr string) error {
	requestUrl := fmt.Sprintf("%v/container/add", addr)
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
	addrs := strings.Split(c.URL, ",")
	if len(addrs) == 0 {
		glog.Errorf("[CMDBClient] CMDB url must be specified, url:%v", c.URL)
		return fmt.Errorf("cmdb url must be specified, url:%v", c.URL)
	}
	for _, addr := range addrs {
		err := c.UpdateOneContainerInfo(reqInfo, addr)
		if err != nil {
			glog.Errorf("[CMDBClient] Update container info into cmdb %v failed, err:%v", addr, err)
			return err
		}
	}
	return nil
}

//UpdateOneContainerInfo() update container into one cmdb.
func (c *CMDBClient) UpdateOneContainerInfo(reqInfo []byte, addr string) error {
	boosUrl := fmt.Sprintf("%v/container/updateByContainerSn", addr)
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
		return fmt.Errorf("Update cmdb failed, resp:%v, req:%v", DumpJson(*resp), string(reqInfo))
	}
	return nil
}

//UpdateContainerInfo() delete container info from cmdb.
func (c *CMDBClient) DeleteContainerInfo(sn string) error {
	addrs := strings.Split(c.URL, ",")
	if len(addrs) == 0 {
		glog.Errorf("[CMDBClient] CMDB url must be specified, url:%v", c.URL)
		return fmt.Errorf("cmdb url must be specified, url:%v", c.URL)
	}
	for _, addr := range addrs {
		err := c.DeleteOneContainerInfo(sn, addr)
		if err != nil {
			glog.Errorf("[CMDBClient] Delete container %v info from cmdb %v failed, err:%v", sn, addr, err)
			return err
		}
	}
	return nil
}

//DeleteOneContainerInfo() delete container info from one cmdb.
func (c *CMDBClient) DeleteOneContainerInfo(sn string, addr string) error {
	bossUrl := fmt.Sprintf("%v/container/deleteByContainerSn?containerSn=%v", addr, sn)
	glog.V(5).Infof("Method:%v, URL:%v", http.MethodPost, bossUrl)

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
		return fmt.Errorf("Delete container %v from cmdb failed, resp:%v", sn, DumpJson(*resp))
	}
	return nil
}

//GetContainerStatus() get container info from cmdb.
func (c *CMDBClient) GetContainerStatus(sn string) (int, error) {
	addrs := strings.Split(c.URL, ",")
	if len(addrs) == 0 {
		glog.Errorf("[CMDBClient] CMDB url must be specified, url:%v", c.URL)
		return 0, fmt.Errorf("cmdb url must be specified, url:%v", c.URL)
	}
	var count200, count404 int
	for _, addr := range addrs {
		cmdbResp, err := c.GetOneContainerInfo(sn, addr)
		if err != nil {
			glog.Errorf("[CMDBClient] Get container %v info from cmdb %v failed, err:%v", sn, addr, err)
			return 0, err
		}
		glog.Infof("url:%v, sn:%v, info:%v", sn, DumpJson(cmdbResp), c.URL)
		if cmdbResp == nil {
			glog.Errorf("[CMDBClient] Get container %v info from cmdb %v is nil, info:%v", sn, addr, cmdbResp)
			return 0, err
		}
		if cmdbResp.Code == http.StatusOK {
			count200++
			continue
		} else if cmdbResp.Code == http.StatusNotFound {
			count404++
			continue
		} else {
			return cmdbResp.Code, nil
		}
	}
	if count404 == len(addrs) {
		return http.StatusNotFound, nil
	} else if count200 == len(addrs) {
		return http.StatusOK, nil
	}
	return 0, nil
}

// GetContainerInfo() get container info from cmdb.
func (c *CMDBClient) GetContainerInfo(sn string) ([]*CMDBResp, error) {
	addrs := strings.Split(c.URL, ",")
	if len(addrs) == 0 {
		glog.Errorf("[CMDBClient] CMDB url must be specified, url:%v", c.URL)
		return nil, fmt.Errorf("cmdb url must be specified, url:%v", c.URL)
	}
	resp := make([]*CMDBResp, 0)
	for _, addr := range addrs {
		cmdbResp, err := c.GetOneContainerInfo(sn, addr)
		if err != nil {
			glog.Errorf("[CMDBClient] Get container %v info from cmdb %v failed, err:%v", sn, addr, err)
			return nil, err
		}
		resp = append(resp, cmdbResp)
	}
	return resp, nil
}

//GetOneContainerInfo() get container info from one cmdb.
func (c *CMDBClient) GetOneContainerInfo(sn string, addr string) (*CMDBResp, error) {
	bossUrl := fmt.Sprintf("%v/container/getByContainerSn?containerSn=%v", addr, sn)
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

func DumpJson(v interface{}) string {
	str, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return String(str)
}

// ToString convert slice to string without mem copy.
func String(b []byte) (s string) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pstring.Data = pbytes.Data
	pstring.Len = pbytes.Len
	return
}
