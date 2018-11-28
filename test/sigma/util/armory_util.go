package util

import (
	"encoding/json"
	"errors"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"fmt"

	"github.com/golang/glog"
)

const (
	namingBackend = "armory"
)

// ArmoryInfo copy from sigma-k8s-controller/pkg/util/naming/ns.go
type ArmoryInfo struct {
	ID               int    `json:".id"`
	IP               string `json:"dns_ip"`
	ServiceTag       string `json:"sn"`
	NodeName         string `json:"nodename"`
	NodeGroup        string `json:"nodegroup"`
	Vmparent         string `json:"vmparent"`
	ParentServiceTag string `json:"parent_service_tag"`
	State            string `json:"state"`
	Site             string `json:"site"`
	Model            string `json:"model"`
	ProductName      string `json:"product_name"`
	AppUseType       string `json:"app_use_type"`
}

// ArmoryQueryResult copy from sigma-k8s-controller/pkg/util/naming/armory/armory.go
type ArmoryQueryResult struct {
	Error   string       `json:"error"`
	Message string       `json:"message"`
	Num     int          `json:"num"`
	Data    []ArmoryInfo `json:"result"`
}

// QueryArmory query the daily armory.
func QueryArmory(query string) ([]ArmoryInfo, error) {
	selectRows := "[default],rack,location_in_rack,logic_region_flag,product_name,app_use_type,product_id,product_name,hw_cpu,hw_harddisk,hw_mem,hw_raid,room,sm_name,app_use_type,create_time,modify_time,parent_service_tag"

	paramMap := map[string]string{
		"_username": "zeus",
		"key":       "iabSU71PfURu90Lz6LE5vg==",
		"select":    selectRows,
		"q":         query,
	}

	// hardcoding daily armory url
	armoryURL := "http://gapi.a.alibaba-inc.com/page/api/free/opsfreeInterface/search.htm"

	values := url.Values{}
	for k, v := range paramMap {
		values.Set(k, v)
	}

	//glog.Info("Query armory args:%v", values)

	resp, err := http.PostForm(armoryURL, values)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	armoryResult := &ArmoryQueryResult{}
	if err := json.Unmarshal(data, armoryResult); err != nil {
		return nil, err
	}

	if armoryResult.Error != "" {
		info, _ := url.QueryUnescape(armoryResult.Message)
		return nil, errors.New(html.UnescapeString(info))
	}

	//glog.Info("armoryresult :%v ", armoryResult)
	if armoryResult.Num <= 0 {
		return nil, nil
	}

	for _, data := range armoryResult.Data {
		data.Site = strings.ToLower(data.Site)
	}

	return armoryResult.Data, nil
}

// GetHostSnFromIp get the host SN from host IP.
func GetHostSnFromHostIp(ip string) string {
	nsInfo, err := QueryArmory(fmt.Sprintf("dns_ip=='%v'", ip))
	if err != nil {
		glog.Error("query armory error :%v ", err)
		return ""
	}
	if len(nsInfo) == 0 {
		glog.Error("no armory record for ip :%v ", ip)
		return ""
	}
	return nsInfo[0].ServiceTag
}
