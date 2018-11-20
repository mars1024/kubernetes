package xvip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/axgle/mahonia"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type client struct {
	BaseURL *url.URL

	name  string
	token string

	httpClient        *http.Client
	customHTTPHeaders map[string]string
	queryInterval     time.Duration
}

// WithHTTPClient 用于给 xvip client 设置 HTTP client
func WithHTTPClient(httpClient *http.Client) func(*client) error {
	return func(client *client) error {
		if httpClient != nil {
			client.httpClient = httpClient
			return nil
		}
		return fmt.Errorf("empty http client")
	}
}

// WithHTTPHeaders overrides the client default http headers
func WithHTTPHeaders(headers map[string]string) func(*client) error {
	return func(c *client) error {
		c.customHTTPHeaders = headers
		return nil
	}
}

func WithInterval(interval time.Duration) func(*client) error {
	return func(c *client) error {
		c.queryInterval = interval
		return nil
	}
}

const (
	apiAuthName  = "x-apiauth-name"
	apiAuthToken = "x-apiauth-token"

	apiCodeCreateVip   = "CreateVip"
	apiCodeDeleteVip   = "DeleteVip"
	apiCodeAddRs       = "AddRs"
	apiCodeDeleteRs    = "DeleteRs"
	apiCodeEnableRs    = "EnableRs"
	apiCodeDisableRs   = "DisableRs"
	apiCodeGetTaskInfo = "GetTaskInfo"
	apiCodeGetVsInfo   = "GetVsInfo"
	apiCodeGetRsInfo   = "GetRsInfo"
)

func New(endpoint, name, token string, options ...func(*client) error) (Client, error) {
	var err error
	baseUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	client := &client{
		BaseURL: baseUrl,
		name:    name,
		token:   token,
		httpClient: &http.Client{
			Timeout: time.Second * 600,
		},
		queryInterval: time.Second * 3,
	}

	options = append(options, WithHTTPHeaders(map[string]string{
		apiAuthName:  name,
		apiAuthToken: token,
	}))
	for _, option := range options {
		err = option(client)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

func (c *client) newRequest(method string, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(c.customHTTPHeaders) > 0 {
		for header, val := range c.customHTTPHeaders {
			req.Header.Set(header, val)
		}
	}
	glog.V(5).Infof("request body: %#v", body)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *client) do(req *http.Request, v interface{}) (*http.Response, error) {
	glog.V(11).Infof("xvip client do request: %#v", req)
	resp, err := c.httpClient.Do(req)
	glog.V(11).Infof("xvip client do response: %#v", resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	enc := mahonia.NewDecoder("gbk")
	body, err := ioutil.ReadAll(enc.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("response: %s", string(body))
	err = json.Unmarshal(body, v)

	return resp, err
}

type CommonResponse struct {
	Total         int    `json:"total"`
	Success       bool   `json:"success"`
	ResultCode    string `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
	//ReqId         string `json:"reqId"`
}

type AddVipResponse struct {
	CommonResponse
	Data struct {
		Vip string `json:"vip"`
	} `json:"data"`
}

type taskResponse struct {
	CommonResponse
	Data *struct {
		Status string `json:"status"`
		Info   string `json:"info"`
	}
}

const (
	TaskStatusReady = "ready"
	TaskStatusProc  = "proc"
	TaskStatusSucc  = "succ"
	TaskStatusFail  = "fail"
)

var maxRetryCount = 1200
var maxRsPerRequest = 500

func (c *client) wait(reqId string) (err error) {
	for i := 1; i < maxRetryCount; i++ {
		time.Sleep(c.queryInterval)
		var req *http.Request
		req, err = c.newRequest("GET", "/openapi/call.json", struct {
			ChangeOrderId string `json:"changeOrderId"`
			ApiCode       string `json:"_apiCode"`
		}{
			ChangeOrderId: reqId,
			ApiCode:       apiCodeGetTaskInfo,
		})

		if err != nil {
			glog.Errorf("query xvip task error: %s", err)
			time.Sleep(c.queryInterval)
			continue
		}

		var resp taskResponse
		_, err = c.do(req, &resp)
		if err != nil {
			glog.Errorf("query xvip task error: %s", err)
			time.Sleep(c.queryInterval)
			continue
		}

		if !resp.Success {
			glog.Errorf("err code: %s, err massage: %s", resp.ResultCode, resp.ResultMessage)
			continue
		} else {
			switch resp.Data.Status {
			case TaskStatusReady, TaskStatusProc:
				continue
			case TaskStatusFail:
				return fmt.Errorf("task fail: %s", resp.Data.Info)
			case TaskStatusSucc:
				return nil
			}
		}
	}
	return fmt.Errorf("maxRetryCount reached for query task: %s", reqId)
}

func (c *client) AddVIP(spec *XVIPSpec) (ip string, err error) {
	if spec == nil || len(spec.RealServerList) == 0 {
		return "", fmt.Errorf("wrong xvip spec: empty spec or empty real server list")
	}
	spec.ChangeOrderId = string(uuid.NewUUID())
	req, err := c.newRequest("POST", "/openapi/call.json", struct {
		XVIPSpec
		ApiCode string `json:"_apiCode"`
	}{
		XVIPSpec: *spec,
		ApiCode:  apiCodeCreateVip,
	})
	if err != nil {
		return "", err
	}
	var resp AddVipResponse
	_, err = c.do(req, &resp)
	if err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("request id: %s, err code: %s, err massage: %s", spec.ChangeOrderId, resp.ResultCode, resp.ResultMessage)
	}
	return resp.Data.Vip, c.wait(spec.ChangeOrderId)
}

func (c *client) DeleteVIP(spec *XVIPSpec) (err error) {
	spec.ChangeOrderId = string(uuid.NewUUID())
	req, err := c.newRequest("POST", "/openapi/call.json", struct {
		XVIPSpec
		ApiCode string `json:"_apiCode"`
	}{
		XVIPSpec: *spec,
		ApiCode:  apiCodeDeleteVip,
	})
	if err != nil {
		return err
	}
	var resp CommonResponse
	_, err = c.do(req, &resp)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("request id: %s, err code: %s, err massage: %s", spec.ChangeOrderId, resp.ResultCode, resp.ResultMessage)
	}
	return c.wait(spec.ChangeOrderId)
}

type OperateRealServerRequest struct {
	ApiCode string `json:"_apiCode"`

	ApplyUser      string         `json:"applyUser,omitempty"`
	Ip             string         `json:"ip,omitempty"`
	Port           int32          `json:"port,omitempty"`
	Protocol       v1.Protocol    `json:"protocol,omitempty"`
	RealServerList RealServerList `json:"rs"`

	ChangeOrderId string `json:"changeOrderId,omitempty"`
}

func (c *client) operateRealServer(spec *XVIPSpec, op Operation, status Status, rss ...*RealServer) (err error) {
	if len(rss) == 0 {
		return nil
	}
	var operateRequest OperateRealServerRequest
	operateRequest = OperateRealServerRequest{
		ApplyUser:     spec.ApplyUser,
		Ip:            spec.Ip,
		Port:          spec.Port,
		Protocol:      spec.Protocol,
		ChangeOrderId: string(uuid.NewUUID()),
	}
	if status == StatusDynamic {
		status = rss[0].Status
	}
	switch op {
	case OpAdd:
		operateRequest.ApiCode = apiCodeAddRs
	case OpDelete:
		operateRequest.ApiCode = apiCodeDeleteRs
	case OpUpdate:
		if status == StatusEnable {
			operateRequest.ApiCode = apiCodeEnableRs
		} else if status == StatusDisable {
			operateRequest.ApiCode = apiCodeDisableRs
		} else {
			return fmt.Errorf("unknown status: %s", status)
		}
	}

	var remainder int
	var count int

	remainder = len(rss)
PROCESS:
	count = 0
	for _, rs := range rss {
		operateRequest.RealServerList = append(operateRequest.RealServerList, &RealServer{
			Ip:     rs.Ip,
			Port:   rs.Port,
			Status: status,
			Op:     op,
		})
		count++
		remainder--
		if count == maxRsPerRequest {
			rss = rss[maxRsPerRequest:]
			break
		}
	}
	req, err := c.newRequest("POST", "/openapi/call.json", operateRequest)
	if err != nil {
		return err
	}
	var resp CommonResponse
	_, err = c.do(req, &resp)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("request id: %s, err code: %s, err massage: %s", operateRequest.ChangeOrderId, resp.ResultCode, resp.ResultMessage)
	}
	err = c.wait(operateRequest.ChangeOrderId)
	if err != nil {
		return err
	}
	if remainder > 0 {
		goto PROCESS
	}
	return nil
}

func (c *client) AddRealServer(spec *XVIPSpec, rss ...*RealServer) (err error) {
	return c.operateRealServer(spec, OpAdd, StatusDynamic, rss...)
}

func (c *client) DeleteRealServer(spec *XVIPSpec, rss ...*RealServer) (err error) {
	return c.operateRealServer(spec, OpDelete, StatusDisable, rss...)
}

func (c *client) EnableRealServer(spec *XVIPSpec, rss ...*RealServer) (err error) {
	return c.operateRealServer(spec, OpUpdate, StatusEnable, rss...)
}

func (c *client) DisableRealServer(spec *XVIPSpec, rss ...*RealServer) (err error) {
	return c.operateRealServer(spec, OpUpdate, StatusDisable, rss...)
}

type GetRsRequest struct {
	VIP       string `json:"vsIp"`
	Port      int32  `json:"vsPort,omitempty"`
	VipBuType string `json:"vipBuType,omitempty"`
	Scope     string `json:"scope"`
	ApiCode   string `json:"_apiCode"`
}

type GetRsResponse struct {
	CommonResponse
	Data []GetRsResponseData `json:"data"`
}

type GetRsResponseData struct {
	VsIp     string      `json:"vsIp"`
	VsPort   int32       `json:"vsPort"`
	Protocol v1.Protocol `json:"protocol"`
	RsList   []rs        `json:"rsList"`
}

type rs struct {
	RsIp   string `json:"rsIp"`
	RsPort int32  `json:"rsPort"`
	Status Status `json:"status"`
}

var (
	ScopeAll = "all"
	ScopeSLB = "slb"
	ScopeLVS = "lvs"
)

func (c *client) GetRsInfo(spec *XVIPSpec) (XVIPSpecList, error) {
	req, err := c.newRequest("GET", "/openapi/call.json",
		GetRsRequest{
			VIP:       spec.Ip,
			Port:      spec.Port,
			VipBuType: spec.VipBuType,
			Scope:     ScopeAll,
			ApiCode:   apiCodeGetRsInfo,
		})

	spec.ChangeOrderId = string(uuid.NewUUID())
	if err != nil {
		return nil, err
	}
	var resp GetRsResponse
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("request id: %s, err code: %s, err massage: %s", spec.ChangeOrderId, resp.ResultCode, resp.ResultMessage)
	}
	var list XVIPSpecList
	for _, vs := range resp.Data {
		s := &XVIPSpec{
			Ip:       vs.VsIp,
			Port:     vs.VsPort,
			Protocol: vs.Protocol,

			ApplyUser:       spec.ApplyUser,
			AppGroup:        spec.AppGroup,
			AppId:           spec.AppId,
			VipBuType:       spec.VipBuType,
			HealthcheckType: spec.HealthcheckType,
			HealthcheckPath: spec.HealthcheckPath,
			ReqAvgSize:      spec.ReqAvgSize,
			QpsLimit:        spec.QpsLimit,
			LbName:          spec.LbName,
		}
		for _, rs := range vs.RsList {
			s.RealServerList = append(s.RealServerList, &RealServer{
				Ip:     rs.RsIp,
				Port:   rs.RsPort,
				Status: rs.Status,
			})
		}
		list = append(list, s)
	}
	return list, nil
}

type TaskInfo = taskResponse

func (c *client) GetTaskInfo(requestId string) (*TaskInfo, error) {
	req, err := c.newRequest("GET", "/openapi/call.json", struct {
		ChangeOrderId string `json:"changeOrderId"`
		ApiCode       string `json:"_apiCode"`
	}{
		ChangeOrderId: requestId,
		ApiCode:       apiCodeGetTaskInfo,
	})

	if err != nil {
		return nil, err
	}
	var resp TaskInfo
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("request id: %s, err code: %s, err massage: %s", requestId, resp.ResultCode, resp.ResultMessage)
	}
	return &resp, nil
}
