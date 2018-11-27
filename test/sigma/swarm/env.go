package swarm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	DefaultTLSDir        = "/tmp/tlscert/"
	DefaultSwarmPort     = "8442"
	DefaultSwarmIP       = "127.0.0.1"
	DefaultEtcdEndpoints = "127.0.0.1:2379"
	DefaultSite          = "et2sqa;et15sqa;zth"
)

var (
	// IP address of sigma 2.0 apiserver
	swarmIP string

	// port of sigma 2.0 apiserver
	swarmPort string

	// SwarmURL url of sigma 2.0 apiserver
	SwarmURL string

	// TLSDir dir to save CA files
	TLSDir string

	// EtcdEndpoints - url address for sigma2.0 etcd
	EtcdEndpoints []string

	// Site - swarm cluster IDC site name
	// one cluster may contains multi sites
	Site []string
)

func init() {
	swarmIP = getSwarmIP()
	swarmPort = getSwarmPort()
	SwarmURL = fmt.Sprintf("https://%s:%s", swarmIP, swarmPort)
	EtcdEndpoints = getEtcdEndpoints()
	TLSDir = getTLSDir()
	Site = getSite()
}

func getSwarmIP() string {
	get, ok := os.LookupEnv("SWARM_IP")
	if ok {
		return get
	}
	return DefaultSwarmIP
}

func getSchedulerInfos(site string) []*schedulerAddressInfo {
	prefix := "/iplist/scheduler/"
	values, err := EtcdGetPrefix(prefix)
	if err != nil {
		framework.Logf("getSchedulerInfos err:%v", err)
		return nil
	}
	siteToMap := map[string][]*schedulerAddressInfo{}
	for _, value := range values {
		key := string(value.Key)
		siteToHostPort := strings.Split(strings.TrimPrefix(key, prefix), "/")
		hostAndPort := strings.Split(siteToHostPort[1], "_")
		site := siteToHostPort[0]
		if siteToMap[site] == nil {
			siteToMap[site] = []*schedulerAddressInfo{}
		}
		siteToMap[site] = append(siteToMap[site], &schedulerAddressInfo{HostIp: hostAndPort[0], Port: hostAndPort[1]})
	}
	schedulerInfoBytes, _ := json.Marshal(siteToMap)
	framework.Logf("getSchedulerInfo:%s, etcdEndPoints:%s", string(schedulerInfoBytes), getEtcdEndpoints())
	return siteToMap[site]
}

type schedulerAddressInfo struct {
	HostIp string
	Port   string
}

func getSwarmPort() string {
	get, ok := os.LookupEnv("SWARM_PORT")
	if ok {
		return get
	}
	return DefaultSwarmPort
}

func getTLSDir() string {
	get, ok := os.LookupEnv("TLS_DIR")
	if ok {
		return get
	}
	return DefaultTLSDir
}

func getEtcdEndpoints() []string {
	var endpoints string
	var ok bool

	endpoints, ok = os.LookupEnv("SIGMA_ETCD_ENDPOINTS")
	if !ok {
		endpoints = DefaultEtcdEndpoints
	}
	return strings.Split(endpoints, ",")
}

func getSite() []string {
	site, ok := os.LookupEnv("SIGMA_SITE")
	if !ok {
		return strings.Split(DefaultSite, ";")
	}

	return strings.Split(site, ";")
}
