package armory

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"time"
	"net"

	"github.com/golang/glog"
)

type client struct {
	httpClient  *http.Client
	URL         string
	User        string
	Key         string
	Concurrency int
	Timeout     time.Duration
}

type ArmoryQueryResult struct {
	Error   string          `json:"error"`
	Message string          `json:"message"`
	Num     int             `json:"num"`
	Data    json.RawMessage `json:"result"`
}

const (
	DeviceSelectRows = "[default],rack,location_in_rack,logic_region_flag,product_name,app_use_type,product_id," +
		"product_name,hw_cpu,hw_harddisk,hw_mem,hw_raid,room,sm_name,app_use_type,create_time,modify_time,parent_service_tag"
	NetworkClusterSelectRows = "id,object_name,parent_id,full_parent_id,object_type,logic_type,idcId,idcName,archVersion,security_domain"
)

const (
	TABLE_DEVICE          = "device"
	TABLE_NETWORK_CLUSTER = "networkcluster"
)

type DeviceInfo struct {
	Ip               string `json:"dns_ip"`
	ServiceTag       string `json:"sn"`
	NodeName         string `json:"nodename"`
	NodeGroup        string `json:"nodegroup"`
	Vmparent         string `json:"vmparent"`
	ParentServiceTag string `json:"parent_service_tag"`
	State            string `json:"state"`
	Site             string `json:"site"`
	Model            string `json:"model"`
	ProductName      string `json:"product_name"`
	SmName           string `json:"sm_name"`
	Rack             string `json:"rack"`
	Room             string `json:"room"`
}

type NetWorkClusterInfo struct {
	ObjectName     string `json:"object_name"` //对应ipdb的logic site
	ArchVersion    string `json:"archVersion"`
	SecurityDomain string `json:"security_domain"`
}

func NewClient(url, user, key string) Client {
	return &client{
		URL:  url,
		User: user,
		Key:  key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          1024,
				MaxIdleConnsPerHost:   512,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

//QueryDevice()  query: sn==''/nodename==''/dns_ip==''
func (c *client) QueryDevice(query string) (*DeviceInfo, error) {
	data, err := c.QueryBySearch(query, DeviceSelectRows, TABLE_DEVICE)
	if err != nil {
		return nil, err
	}
	deviceInfo := make([]DeviceInfo, 0)
	err = json.Unmarshal(data, &deviceInfo)
	if err != nil {
		glog.Errorf("unmarshal armory result info failed, %v, %v", string(data), err)
		return nil, err
	}

	if len(deviceInfo) == 0 {
		return nil, nil
	} else if len(deviceInfo) > 1 {
		glog.Errorf("multiple armory entry found for %v", query)
		return nil, errors.New("multiple armory entry found")
	}
	return &deviceInfo[0], nil
}

//QueryNetWorkCluster()  query: object_name=='EM14-ALIPAY'  object_name==logic_site
func (c *client) QueryNetWorkCluster(query string) (*NetWorkClusterInfo, error) {
	data, err := c.QueryBySearch(query, NetworkClusterSelectRows, TABLE_NETWORK_CLUSTER)
	if err != nil {
		return nil, err
	}
	deviceInfo := []NetWorkClusterInfo{}
	err = json.Unmarshal(data, &deviceInfo)
	if err != nil {
		glog.Errorf("unmarshal armory result info failed, %v, %v", string(data), err)
		return nil, err
	}

	if len(deviceInfo) == 0 {
		return nil, nil
	} else if len(deviceInfo) > 1 {
		glog.Errorf("multiple armory entry found for %v", query)
		return nil, errors.New("multiple armory entry found")
	}
	return &deviceInfo[0], nil
}

//QueryBySearch()
func (c *client) QueryBySearch(query, selectRows, from string) ([]byte, error) {
	str := c.User + time.Now().Format("20060102") + c.Key
	h := md5.New()
	h.Write([]byte(str))
	key := hex.EncodeToString(h.Sum(nil))
	paramMap := map[string]string{
		"_username": c.User,
		"key":       key,
		"select":    selectRows,
		"q":         query,
		"from":      from,
	}
	values := url.Values{}
	for k, v := range paramMap {
		values.Set(k, v)
	}
	armory_url := fmt.Sprintf("%v/page/api/free/opsfreeInterface/search.htm?%v", c.URL, values.Encode())
	glog.V(5).Infof("ArmoryURL:%v", armory_url)
	req, err := http.NewRequest(http.MethodGet, armory_url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		glog.Errorf("query container serverInfo failed. armory url: %v, err:%v", armory_url, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	armoryResult := &ArmoryQueryResult{}
	if err = json.NewDecoder(resp.Body).Decode(armoryResult); err != nil {
		glog.Errorf("call %v response json decode failed, err:%v", armory_url, err)
		return nil, err
	}

	if armoryResult.Error != "" {
		info, _ := url.QueryUnescape(armoryResult.Message)
		glog.Errorf("call %v response json failed, err:%v", armory_url, html.UnescapeString(info))
		return nil, errors.New(html.UnescapeString(info))
	}
	if armoryResult.Num <= 0 {
		return nil, nil
	}
	return armoryResult.Data, nil
}
