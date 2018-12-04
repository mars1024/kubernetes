package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/test/sigma/util/skyline"
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

// QueryArmory query skyline and covert to armory info.
func QueryArmory(queryString string) ([]ArmoryInfo, error) {
	if strings.Contains(queryString, "dns_ip") {
		queryString = strings.Replace(queryString, "dns_ip", "ip", -1)
	}
	queryString = strings.Replace(queryString, "==", "=", -1)

	skylineManager := skyline.NewSkylineManager()
	queryItem := &skyline.QueryItem{
		From: "server",
		Select: strings.Join([]string{skyline.SelectDiskSize, skyline.SelectAppUseType, skyline.SelectParentSn,
			skyline.SelectSn, skyline.SelectIp, skyline.SelectAppGroup, skyline.SelectAppName,
			skyline.SelectParentSn, skyline.SelectHostName, skyline.SelectAppServerState,
			skyline.SelectSecurityDomain, skyline.SelectSite, skyline.SelectModel}, ","),
		Condition: queryString,
		Page:      1,
		Num:       100,
	}
	result, err := skylineManager.Query(queryItem)
	if err != nil {
		return nil, err
	}
	if result == nil {
		// try one more
		time.Sleep(5 * time.Second)
		result, err = skylineManager.Query(queryItem)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("no skyline record is found for %s", queryString)
		}
	}

	armoryInfos := make([]ArmoryInfo, 0)
	for _, item := range result.Value.ItemList {
		armoryInfo := ArmoryInfo{}
		val, ok := item[skyline.SelectIp]
		if ok {
			armoryInfo.IP = val.(string)
		}
		val, ok = item[skyline.SelectSn]
		if ok {
			armoryInfo.ServiceTag = val.(string)
		}
		val, ok = item[skyline.SelectHostName]
		if ok {
			armoryInfo.NodeName = val.(string)
		}
		val, ok = item[skyline.SelectAppGroup]
		if ok {
			armoryInfo.NodeGroup = val.(string)
		}
		val, ok = item[skyline.SelectParentSn]
		if ok {
			armoryInfo.ParentServiceTag = val.(string)
		}
		val, ok = item[skyline.SelectAppServerState]
		if ok {
			armoryInfo.State = val.(string)
		}
		val, ok = item[skyline.SelectSite]
		if ok {
			armoryInfo.Site = val.(string)
		}
		val, ok = item[skyline.SelectModel]
		if ok {
			armoryInfo.Model = val.(string)
		}
		val, ok = item[skyline.SelectAppName]
		if ok {
			armoryInfo.ProductName = val.(string)
		}

		armoryInfos = append(armoryInfos, armoryInfo)
	}
	return armoryInfos, nil
}

// GetHostSnFromHostIp get the host SN from host IP.
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
