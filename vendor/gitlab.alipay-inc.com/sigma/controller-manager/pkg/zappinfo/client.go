package zappinfo

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/axgle/mahonia"
	"github.com/golang/glog"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

type client struct {
	*http.Client
	endpoint string
	token    string
	rwMutex  sync.RWMutex
}

func postZappinfo(c *client, url string, data map[string]string) error {
	url = c.endpoint + url

	data["_zappinfoAuthToken"] = c.token
	formData := map[string][]string{}
	for k, v := range data {
		formData[k] = []string{v}
	}

	resp, err := c.PostForm(url, formData)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	enc := mahonia.NewDecoder("gb18030")

	body, err := ioutil.ReadAll(enc.NewReader(resp.Body))
	if err != nil {
		return err
	}

	result := &result{}
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	if !result.Success {
		return errors.New(result.ResultMessage)
	}

	return nil
}

func (c *client) UpdateMultiServerStatus(hostnames []string, status alipayapis.ZappinfoStatus) error {
	url := "/rest/server/updateServerStatus.json"
	data := map[string]string{
		"hostnames": strings.Join(hostnames, ","),
		"status":    string(status),
	}

	return postZappinfo(c, url, data)
}

type result struct {
	Success       bool   `json:"success"`
	ResultCode    string `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
}

type queryServerResult struct {
	result
	Server *alipayapis.PodZappinfoMetaSpec `json:"server,omitempty"`
}

type queryServerListResult struct {
	result
	Servers []alipayapis.PodZappinfoMetaSpec `json:"serverList"`
}

func NewZappinfoClient(endpoint, token string) (Client) {
	c := &client{
		Client:   &http.Client{},
		endpoint: endpoint,
		token:    token,
	}

	return c
}

// 写入服务器信息
func (c *client) AddServer(server *alipayapis.PodZappinfoMetaSpec) error {
	url := "/rest/server/add.json"
	data := map[string]string{}

	glog.V(5).Infof("add server: %#v", server)
	serverBytes, err := json.Marshal(&server)
	if err != nil {
		return err
	}

	glog.V(5).Infof("zappinfo addserver info: %s", string(serverBytes))

	err = json.Unmarshal(serverBytes, &data)
	if err != nil {
		return err
	}

	return postZappinfo(c, url, data)
}

func (c *client) DeleteServerByHostname(hostname string) error {
	url := "/rest/server/deleteByHostname.json"
	data := map[string]string{}
	data["hostname"] = hostname

	return postZappinfo(c, url, data)
}

func (c *client) UpdateServerStatus(hostname string, status alipayapis.ZappinfoStatus) error {
	url := "/rest/server/updateServerStatus.json"
	data := map[string]string{
		"hostnames": hostname,
		"status":     string(status),
	}

	return postZappinfo(c, url, data)
}

func (c *client) GetServerByHostname(hostname string) (*alipayapis.PodZappinfoMetaSpec, error) {
	resp, err := c.Get(c.endpoint + "/rest/server/queryServerByHostname.json?hostname=" + hostname)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	enc := mahonia.NewDecoder("gb18030")

	body, err := ioutil.ReadAll(enc.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}

	result := queryServerResult{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.ResultMessage)
	}

	return result.Server, nil
}

func (c *client) GetServerByIp(ip string) (*alipayapis.PodZappinfoMetaSpec, error) {
	url := c.endpoint + "/rest/server/getAppServerByIps.json?ips=" + ip

	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	enc := mahonia.NewDecoder("gb18030")

	body, err := ioutil.ReadAll(enc.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}

	result := queryServerListResult{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.ResultMessage)
	}

	if len(result.Servers) == 1 {
		return &result.Servers[0], nil
	} else if len(result.Servers) > 1 {
		return nil, errors.New("the count of server is wrong")
	}
	return nil, nil
}
