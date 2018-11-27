package swarm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/samalba/dockerclient"
	"github.com/sirupsen/logrus"
	"gitlab.alibaba-inc.com/sigma/sigma-api/sigma"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

// Resource is the requested resources
type Resource struct {
	// CPUCount represents how many cpu the container needs
	CPUCount int
	// Memory is the memory size in bytes
	Memory int64
	// DiskSize is the requested disk size
	DiskSize int64
}

// ContainerOption representes all available configuration that is used by client
type ContainerOption struct {
	Resource
	Name      string
	ImageName string
	Labels    map[string]string
}

// ContainerUpdateOption contains all fields that can be used when updating a container.
// Right now only core resource(cpu, memory, disk) update supported
//
// Doc: https://yuque.antfin-inc.com/sys/sigma/ix6r5q
type ContainerUpdateOption struct {
	Resource
	MemorySwap int64
}

// ContainerUpgradeOption contains all fields that can be used when upgrading a container
// Doc: http://docs.alibaba-inc.com/pages/viewpage.action?pageId=417008557
type ContainerUpgradeOption struct {
	Labels map[string]string
	Image  string
	Env    []string
}

// GetContainerBody returns a default container configuration
func GetContainerBody(image, appName, deployUnit string) dockerclient.ContainerConfig {
	return dockerclient.ContainerConfig{
		Image: image,
		Env: []string{
			"ali_run_mode=common_vm",
			"ali_start_app=no",
		},
		Labels: map[string]string{
			"ali.AppDeployUnit":      deployUnit,
			"ali.AppName":            appName,
			"ali.BizName":            "smoking",
			"ali.CpuCount":           "2",
			"ali.EnableOverQuota":    "false",
			"ali.IncreaseReplica":    "1",
			"ali.InstanceGroup":      deployUnit,
			"ali.MemoryHardlimit":    "2048000000", // 2GB
			"ali.SpecifiedNcIps":     "",
			"ali.DiskSize":           "5g",
			"ali.MaxInstancePerHost": "10",
		},
	}
}

func createContainer(name string, q url.Values) (*ContainerCreationResp, error) {
	return createContainerWithLabels(name, q, map[string]string{})
}

func createContainerWithLabels(name string, q url.Values, labels map[string]string) (*ContainerCreationResp, error) {
	return doCreateContainer(name, q, "reg.docker.alibaba-inc.com/ali/os:7u2", "phyhost-ecs-trade", "container-test1", labels, []string{})
}

func createContainerWithMoreConditions(name, imageName, appName, deplyUnit string, q url.Values, labels map[string]string) (*ContainerCreationResp, error) {
	return doCreateContainer(name, q, imageName, appName, deplyUnit, labels, []string{})
}

func doPreviewCreateContainer(name string, q url.Values, imageName, appName, deployUnit string, labels map[string]string, env []string) (*PreviewContainerCreationResp, error) {
	query := WithQuery(q)
	form := WithForm("name", name)
	containerConfig := GetContainerBody(imageName, appName, deployUnit)

	// The "ali.Site" label is required in sigma2.0.
	// If user specify a nodeIp, then query the site from armory
	// If not, just assign one from the Site array
	isIpSpecified := false
	for k, v := range labels {
		if k == "ali.SpecifiedNcIps" {
			isIpSpecified = true

			nsInfo, err := util.QueryArmory(fmt.Sprintf("dns_ip=='%v'", v))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(nsInfo)).Should(Equal(1), fmt.Sprintf("dnsIP:%s not in armory", v))

			containerConfig.Labels["ali.Site"] = strings.ToLower(nsInfo[0].Site)
		}
	}
	// add user-defined labels
	for key, value := range labels {
		containerConfig.Labels[key] = value
	}
	var body Option
	var resp *http.Response
	var err error
	framework.Logf("container config:%+v", containerConfig)
	if isIpSpecified == false {
		// If not specify nodeIp, we need to loop all site util get a successful response...
		for _, v := range Site {
			logrus.Info("try to alloc container in site:", v)
			containerConfig.Labels["ali.Site"] = v
			body = WithJSONBody(containerConfig)
			resp, err = Post("/containers/preview_create", query, form, body)
			if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted) {
				break
			}
		}
	} else {
		body = WithJSONBody(containerConfig)
		resp, err = Post("/containers/preview_create", query, form, body)
	}

	if err != nil {
		logrus.Errorf("create 2.0 container return error:", err)
		return nil, err
	}

	if resp.StatusCode == http.StatusInternalServerError {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		framework.Logf("resp.Body:%s", string(bodyBytes))
		return &PreviewContainerCreationResp{}, nil
	}

	// create should return status code 201 or 202 depends on sync or async
	Expect(resp.StatusCode).Should(SatisfyAny(Equal(http.StatusCreated), Equal(http.StatusAccepted), Equal(http.StatusInternalServerError)))
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("received status code: %d, expect 201 or 202", resp.StatusCode)
	}

	fmt.Printf("Info: preview create container response body:%s", resp.Body)
	container, err := ParsePreviewResponseBody(resp.Body)
	fmt.Printf("Info: preview create container response body:%v", container)

	if err != nil {
		return nil, fmt.Errorf("parse response body failed: %v", err)
	}
	Expect(err).NotTo(HaveOccurred())
	return container, nil
}

func doCreateContainer(name string, q url.Values, imageName, appName, deployUnit string, labels map[string]string, env []string) (*ContainerCreationResp, error) {
	query := WithQuery(q)
	form := WithForm("name", name)
	containerConfig := GetContainerBody(imageName, appName, deployUnit)

	// The "ali.Site" label is required in sigma2.0.
	// If user specify a nodeIp, then query the site from armory
	// If not, just assign one from the Site array
	isIpSpecified := false
	for k, v := range labels {
		if k == "ali.SpecifiedNcIps" {
			isIpSpecified = true

			nsInfo, err := util.QueryArmory(fmt.Sprintf("dns_ip=='%v'", v))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(nsInfo)).Should(Equal(1), fmt.Sprintf("dnsIP:%s not in armory", v))

			containerConfig.Labels["ali.Site"] = strings.ToLower(nsInfo[0].Site)
		}
	}
	// add user-defined labels
	for key, value := range labels {
		containerConfig.Labels[key] = value
	}
	var body Option
	var resp *http.Response
	var err error
	if isIpSpecified == false {
		// If not specify nodeIp, we need to loop all site util get a successful response...
		for _, v := range Site {
			logrus.Info("try to alloc container in site:", v)
			containerConfig.Labels["ali.Site"] = v
			body = WithJSONBody(containerConfig)
			resp, err = Post("/containers/create", query, form, body)
			if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted) {
				break
			}
		}
	} else {
		body = WithJSONBody(containerConfig)
		resp, err = Post("/containers/create", query, form, body)
	}

	if err != nil {
		logrus.Errorf("create 2.0 container return error:", err)
		return nil, err
	}

	if resp.StatusCode == http.StatusInternalServerError {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		framework.Logf("resp.Body:%s", string(bodyBytes))
		return &ContainerCreationResp{}, nil
	}

	// create should return status code 201 or 202 depends on sync or async
	Expect(resp.StatusCode).Should(SatisfyAny(Equal(http.StatusCreated), Equal(http.StatusAccepted)))
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("received status code: %d, expect 201 or 202", resp.StatusCode)
	}

	fmt.Printf("Info: create container response body:%s", resp.Body)
	container, err := ParseResponseBody(resp.Body)
	fmt.Printf("Info: create container response body:%v", container)

	if err != nil {
		return nil, fmt.Errorf("parse response body failed: %v", err)
	}

	if container != nil && len(container.Containers) > 0 {
		for _, cont := range container.Containers {
			cont.DeployUnit = containerConfig.Labels["ali.AppDeployUnit"]
			cont.Site = containerConfig.Labels["ali.Site"]
		}
	}
	Expect(err).NotTo(HaveOccurred())
	return container, nil
}

// CreateContainerAsync create container asynchronous.
func CreateContainerAsync(name string) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "true")

	return createContainer(name, q)
}

// CreateContainerSync create container synchronous.
func CreateContainerSync(name string) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "false")

	return createContainer(name, q)
}

// CreateContainerSyncWithLabels creates a default container with user specified labels,
// and waiting for container running response(or error response).
// NOTE: User defined labels will override default values.
func CreateContainerSyncWithLabels(name string, labels map[string]string) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "false")
	return createContainerWithLabels(name, q, labels)
}

// CreateContainerSyncWithMoreConditions creates a container with more conditions(include appName. deployUnit, image, labels),
// and waiting for container running response(or error response).
// NOTE: User defined labels will override default values.
func CreateContainerSyncWithMoreConditions(name, image, appName, deployUnit string, labels map[string]string) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "false")
	return createContainerWithMoreConditions(name, image, appName, deployUnit, q, labels)
}

// CreateContainerAsyncWithLabels creates a default container with user specified labels,
// it does not wait for container running, only returns with a container id.
// User defined labels will override default values.
func CreateContainerAsyncWithLabels(name string, labels map[string]string) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "true")
	return createContainerWithLabels(name, q, labels)
}

func createContainerWithOption(q url.Values, o *ContainerOption) (*ContainerCreationResp, error) {
	labels := map[string]string{}

	// merge user-defined labels
	for key, value := range o.Labels {
		labels[key] = value
	}

	// parse resource request to sigma label
	if o.CPUCount != 0 {
		labels["ali.CpuCount"] = fmt.Sprintf("%d", o.CPUCount)
	}

	if o.Memory != 0 {
		labels["ali.MemoryHardlimit"] = fmt.Sprintf("%d", o.Memory)
	}

	if o.DiskSize != 0 {
		labels["ali.DiskSize"] = fmt.Sprintf("%d", o.DiskSize)
	}
	if o.ImageName == "" {
		o.ImageName = "reg.docker.alibaba-inc.com/ali/os:7u2"
	}

	return doCreateContainer(o.Name, q, o.ImageName, "phyhost-ecs-trade", "container-test1", labels, []string{})
}

// CreateContainerWithOption creates container with user provided option.
// This is the preferred method to call for complex container configuration,
// such as user defined resource, container image etc.
//
// Example usage, create a container with 4C8G resource:
// CreateContainerWithOption(
//    &ContainerOption{
//      CPU: 4,
//      Memory: 8 * 1024 * 1024 * 1024,
// })
func CreateContainerWithOption(o *ContainerOption) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "false")

	return createContainerWithOption(q, o)
}

// AntPreviewCreateContainerWithOption preview creates container with user provided option.
// This is the preferred method to call for complex container configuration,
// such as user defined resource.
//
// Example usage, create a container with 4C8G resource:
// CreateContainerWithOption(
//    &ContainerOption{
//      CPU: 4,
//      Memory: 8 * 1024 * 1024 * 1024,
// })
func AntPreviewCreateContainerWithOption(o *ContainerOption) (*PreviewContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "false")

	labels := map[string]string{}
	labels["ali.PreviewCache"] = "false"

	// merge user-defined labels
	for key, value := range o.Labels {
		labels[key] = value
	}

	// parse resource request to sigma label
	if o.CPUCount != 0 {
		labels["ali.CpuCount"] = fmt.Sprintf("%d", o.CPUCount)
	}

	if o.Memory != 0 {
		labels["ali.MemoryHardlimit"] = fmt.Sprintf("%d", o.Memory)
	}

	if o.DiskSize != 0 {
		labels["ali.DiskSize"] = fmt.Sprintf("%d", o.DiskSize)
	}
	if o.ImageName == "" {
		o.ImageName = "reg.docker.alibaba-inc.com/ali/os:7u2"
	}

	return doPreviewCreateContainer(o.Name, q, o.ImageName, "phyhost-ecs-trade", "scheduler-e2e-depoly-unit", labels, []string{})
}

// CreateContainerWithOptionAsync is the same as CreateContainerWithOption, but
// returns immediately.
func CreateContainerWithOptionAsync(o *ContainerOption) (*ContainerCreationResp, error) {
	q := url.Values{}
	q.Add("async", "true")

	return createContainerWithOption(q, o)
}

func inspectContainer(containerName string) (*http.Response, error) {
	if containerName == "" {
		return nil, fmt.Errorf("container ID can't be nil")
	}

	return Get("/containers/" + containerName + "/json")
}

// UpdateContainer updates container configuration.
// It is used to mainly update container resource
//
// Example usage(update container memory):
// UpdateContainer("my-container", &ContainerUpdateOption{
// 	Memory: 2 * 1024 * 1024 * 1024,
// })
func UpdateContainer(name string, option *ContainerUpdateOption) error {
	// make sure memoryswap is at least equal to memory
	if option.MemorySwap < option.Memory {
		option.MemorySwap = option.Memory
	}

	body := WithJSONBody(option)
	uri := fmt.Sprintf("/containers/%s/update", name)
	resp, err := Post(uri, body)

	if err != nil {
		return err
	}

	// update container failed, report to user as error
	if resp.StatusCode > 400 {
		data, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			return fmt.Errorf("can't update container %s：%s", name, string(data))
		}
		return fmt.Errorf("can't update container %s", name)
	}

	return nil
}

// GetRequestState is used for async request, to get the state of request
func GetRequestState(requestID string) *ContainerResult {
	if requestID == "" {
		logrus.Error("requestId is empty")
		return nil
	}

	logrus.Printf("Query requestId:%s", requestID)
	response, err := Get("/sigma/requests/" + requestID + "/json")
	Expect(err).NotTo(HaveOccurred())
	if response.StatusCode != http.StatusOK {
		logrus.Error("query request return %d", response.StatusCode)
		return nil
	}

	request := &Request{}
	bytes, err := ioutil.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())

	logrus.Info("GetRequestState, response:%s", string(bytes))

	err = json.Unmarshal(bytes, request)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(request.Actions) > 0).Should(BeTrue())

	sigmaRequirement := &sigma.Requirement{}
	err = json.Unmarshal([]byte(request.Body), sigmaRequirement)
	Expect(err).NotTo(HaveOccurred())

	for _, ac := range request.Actions {
		result := &AllocResult{}
		err = json.Unmarshal([]byte(ac.Result), result)
		Expect(err).NotTo(HaveOccurred())

		return &ContainerResult{
			CPUSet:        result.CpuSet,
			ContainerSN:   result.ContainerSn,
			ContainerHN:   result.ContainerHn,
			ContainerName: result.ContainerName,
			ContainerIP:   result.ContainerIp,
			DeployUnit:    sigmaRequirement.App.DeployUnit,
			Site:          sigmaRequirement.Site,
			HostIP:        result.HostIp,
			HostSN:        result.HostSn,
		}
	}
	return nil
}

// DeleteContainer delete a container.
func DeleteContainer(name string) (*http.Response, error) {
	if name == "" {
		return nil, fmt.Errorf("container ID can't be nil")
	}
	return Delete("/containers/" + name + "?v=1")
}

// MustDeleteContainer makes sure container is actually deleted for once and all
func MustDeleteContainer(name string) {
	if name == "" {
		return
	}
	resp, err := DeleteContainer(name)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).Should(Equal(http.StatusNoContent))

	err = wait.PollImmediate(2*time.Second, 2*time.Minute, containerDeleted(name))
	Expect(err).NotTo(HaveOccurred())
}

func containerDeleted(name string) wait.ConditionFunc {
	return func() (bool, error) {
		resp, err := inspectContainer(name)
		if err != nil {
			return false, err
		}
		switch resp.StatusCode {
		case http.StatusNotFound:
			return true, nil
		case http.StatusFound:
			return false, fmt.Errorf("container not deleted")
		}
		return false, nil
	}
}

// PreviewCreate returns the preview result
func PreviewCreateWithLabel(name string, labels map[string]string) (num int, err error) {
	if len(labels) == 0 {
		labels = make(map[string]string, 1)
	}
	labels["ali.Preview"] = "true"
	container, err := CreateContainerSyncWithLabels(name, labels)
	if err != nil {
		return 0, err
	}
	if container.ID == "" {
		return 0, nil
	}
	return strconv.Atoi(container.ID)
}

func PreviewCreateContainerWithOption(o *ContainerOption) (num int, err error) {
	labels := make(map[string]string, 1)
	labels["ali.Preview"] = "true"
	labels["ali.PreviewCache"] = "false"

	// merge user-defined labels
	for key, value := range o.Labels {
		labels[key] = value
	}

	q := url.Values{}
	q.Add("async", "true")

	// parse resource request to sigma label
	if o.CPUCount != 0 {
		labels["ali.CpuCount"] = fmt.Sprintf("%d", o.CPUCount)
	}

	if o.Memory != 0 {
		labels["ali.MemoryHardlimit"] = fmt.Sprintf("%d", o.Memory)
	}

	if o.DiskSize != 0 {
		labels["ali.DiskSize"] = fmt.Sprintf("%ds", o.DiskSize)
	}

	container, err := doCreateContainer(o.Name, q, o.ImageName, "phyhost-ecs-trade", "container-test1", labels, []string{})
	if err != nil {
		return 0, err
	}
	if container.ID == "" {
		return 0, nil
	}
	return strconv.Atoi(container.ID)
}

// StartContainer start a container.
func StartContainer(containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID can't be nil")
	}
	url := fmt.Sprintf("/containers/%s/start", containerID)
	resp, err := Post(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("start 2.0 container status is not http.StatusNoContent, status code is %v", resp.StatusCode)
	}
	return nil
}

// QueryContainer start a container.
func QueryContainer(containerID string) ([]byte, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container ID can't be nil")
	}
	url := fmt.Sprintf("/containers/%s/json", containerID)
	resp, err := Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query 2.0 container status is not http.StatusOK")
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetPodSnFromContainerResponse parse container query response to get the SN
func GetPodSnFromContainerResponse(resp []byte) (string, error) {
	var containerQueryResp ContainerQueryResp
	err := json.Unmarshal(resp, &containerQueryResp)
	if err != nil {
		return "", err
	}
	containerEnv := containerQueryResp.Config.Env
	for _, env := range containerEnv {
		if strings.Contains(env, "SN=") {
			ret := strings.Split(env, "=")
			return ret[1], nil
		}
	}
	return "", fmt.Errorf("can not find sn in container")
}

// ExecBody body of exec cmd
type ExecBody struct {
	Cmd []string `json:"Cmd"`
}

// ExecResponse response of exec cmd, include an exec id
type ExecResponse struct {
	ID string `json:"Id"`
}

// QueryResponse final exec result
type QueryResponse struct {
	Running bool `json:"Running"`
}

// ExecCommandInContainer exec an command in 2.0 container
func ExecCommandInContainer(containerSN string, cmd []string) error {

	// step 1: create exec
	url := fmt.Sprintf("/containers/%v/exec", containerSN)
	execBody := ExecBody{Cmd: cmd}
	body := WithJSONBody(execBody)

	resp, err := Post(url, body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create exec failed, code: %v", resp.StatusCode)
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read exec id failed, err: %v", err)
	}
	var execResp ExecResponse
	if err := json.Unmarshal(data, &execResp); err != nil {
		return fmt.Errorf("get exec id failed, err: %v", err)
	}

	// step 2: start exec
	url = fmt.Sprintf("/exec/%v/start", execResp.ID)
	resp, err = Post(url, body)
	if err != nil {
		return fmt.Errorf("start exec failed, err: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("start exec failed, code: %v", resp.StatusCode)
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read exec result failed, err: %v", err)
	}
	logrus.Infof("exec cmd[%s] result is: %s", url, string(data))

	return nil
}

// UpgradeContainer upgrade a container with specified config, if success, return a request ID
func UpgradeContainer(containerID string, upgradeOption ContainerUpgradeOption) (string, error) {
	body := WithJSONBody(upgradeOption)
	q := url.Values{}
	q.Add("async", "true")
	query := WithQuery(q)
	url := fmt.Sprintf("/containers/%s/upgrade", containerID)
	resp, err := Post(url, query, body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("upgrade container failed, error code: %v", resp.StatusCode)
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read upgrade container result failed, err: %v", err)
	}
	upgradeResp := &ContainerCreationResp{}
	err = json.Unmarshal(data, upgradeResp)
	framework.Logf("request ID of 2.0 upgrade is %s", upgradeResp.ID)
	return upgradeResp.ID, nil
}

// QueryRequestStateWithTimeout query request state until timeout.
// If creating/upgrading container successfully, return the container info
func QueryRequestStateWithTimeout(requestID string, timeout time.Duration) (*ContainerResult, error) {
	t := time.Now()
	for {
		request, err := queryRequest(requestID)
		if err != nil {
			return nil, fmt.Errorf("quest request id[%s] error: %s", requestID, err.Error())
		}
		if request.State == "finish" {
			framework.Logf("finish to query sigma 2.0 request id[%s]", requestID)
			sigmaRequirement := &sigma.Requirement{}
			err = json.Unmarshal([]byte(request.Body), sigmaRequirement)
			if err != nil {
				return nil, fmt.Errorf("parse 2.0 requirement error: %s", err.Error())
			}

			for _, ac := range request.Actions {
				if ac.State != "success" {
					continue
				}
				result := &AllocResult{}
				err = json.Unmarshal([]byte(ac.Result), result)
				if err != nil {
					return nil, fmt.Errorf("parse 2.0 request action error: %s", err.Error())
				}
				return &ContainerResult{
					CPUSet:      result.CpuSet,
					ContainerSN: result.ContainerSn,
					ContainerID: result.ContainerId,
					DeployUnit:  sigmaRequirement.App.DeployUnit,
					Site:        sigmaRequirement.Site,
					HostIP:      result.HostIp,
					HostSN:      result.HostSn,
				}, nil
			}
			break
		}
		if time.Since(t) >= timeout {
			return nil, fmt.Errorf("timeout for querying the request id[%s]", requestID)
		}
		framework.Logf("retrying to query sigma 2.0 request id[%s]...", requestID)
		time.Sleep(10 * time.Second)
	}
	return nil, nil
}

func queryRequest(requestID string) (*Request, error) {
	response, err := Get("/sigma/requests/" + requestID + "/json")
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query request return %d", response.StatusCode)
	}
	request := &Request{}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	framework.Logf("GetRequestState, response:%s", string(bytes))
	err = json.Unmarshal(bytes, request)
	if err != nil {
		return nil, err
	}
	return request, nil
}
