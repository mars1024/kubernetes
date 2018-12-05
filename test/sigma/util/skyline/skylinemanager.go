/**
操作skyline的原生交互接口

账号权限: http://docs.skyline.alibaba-inc.com/authority/auth_apply_common.html
查询: http://docs.skyline.alibaba-inc.com/search/lql_search.html
注册: http://docs.skyline.alibaba-inc.com/server/vm_add.html
取消: http://docs.skyline.alibaba-inc.com/server/vm_delete.html
updateNode: http://docs.skyline.alibaba-inc.com/server/server_update.html
*/
package skyline

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
)

type SkyConfig struct {
	App         string
	Url         string
	User        string
	Key         string
	Concurrency int
	Switch      bool
}

type SkylineManager struct {
	config *SkyConfig
}

func NewSkylineManager() *SkylineManager {
	return &SkylineManager{
		config: &SkyConfig{
			App:         "sigma-apiserver",
			Url:         "http://sky.alibaba-inc.com",
			User:        "sigma_engin_app",
			Key:         "bqtABUMCUoV2J9XR",
			Concurrency: 100,
			Switch:      true,
		},
	}
}

// 查询
// FIXME 当error==nil时，Result还是可能为nil
func (skyline *SkylineManager) Query(queryItem *QueryItem) (*Result, error) {
	queryParam := &QueryParam{
		Auth: skyline.buildAuth(),
		Item: queryItem,
	}
	body, err := json.Marshal(queryParam)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf(queryUri, skyline.config.Url)
	result, err := skyline.httpRequestToSky(url, body)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("skyline query queryItem:%v, request url: %v, body: %v, message: %v",
			queryItem, url, string(body), result.ErrorMessage)
	}
	return result, nil
}

// 简单的认证
func (skyline *SkylineManager) buildAuth() *auth {
	auth := &auth{
		Account:   skyline.config.User,
		AppName:   skyline.config.App,
		Timestamp: time.Now().Unix(),
	}
	md5Cal := md5.New()
	io.WriteString(md5Cal, auth.Account)
	io.WriteString(md5Cal, skyline.config.Key)
	io.WriteString(md5Cal, fmt.Sprintf("%v", auth.Timestamp))
	auth.Signature = hex.EncodeToString(md5Cal.Sum(nil))
	return auth
}

// 绑定用户信息
// FIXME 先写死
func (skyline *SkylineManager) buildSkyOperator(operatorType interface{}) *skyOperator {
	skyOperator := &skyOperator{
		Type:     operatorType,
		Nick:     "fengxiu.fl",
		WorkerId: "169572",
	}
	return skyOperator
}

const (
	DEFAULT_DIAL_TIMEOUT    int = 10
	DEFAULT_END2END_TIMEOUT int = 120

	RETRY_COUNT              = 2
	RETRY_INTERVAL           = 10
	RETRY_INTERVAL_INCREMENT = 10
)

// 通用的http-post
func (skyline *SkylineManager) httpRequestToSky(url string, body []byte) (*Result, error) {
	glog.Infof("skylineHttpRequestToSky. request body: %s", string(body))
	//fmt.Println(fmt.Sprintf("skylineHttpRequestToSky. request body: %s", string(body)))

	resByte, rErr := httpPostJsonWithHeadersWithTime(url, body, nil, nil, DEFAULT_DIAL_TIMEOUT, DEFAULT_END2END_TIMEOUT)
	if rErr != nil {
		glog.Errorf("httpRequestToSky failed: %s", rErr.Error())
		return nil, rErr
	}
	result := &Result{}
	pErr := json.Unmarshal(resByte, result)
	if pErr != nil {
		glog.Errorf("skylineHttpRequestToSkyUnmarshal failed: %s", pErr.Error())
		return nil, pErr
	}
	glog.Infof("skylineHttpRequestToSky success. response: %s", string(resByte))
	//fmt.Println(fmt.Sprintf("skylineHttpRequestToSky success. response: %s", string(resByte)))

	return result, nil
}

func httpPostJsonWithHeadersWithTime(httpUrl string, body []byte, headers map[string]string, params map[string]string,
	timeoutInSecond int, end2endTimeoutInSecond int) ([]byte, error) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Duration(timeoutInSecond) * time.Second,
		}).Dial,
	}
	var client = &http.Client{
		Timeout:   time.Duration(end2endTimeoutInSecond) * time.Second,
		Transport: netTransport,
	}

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	var data []byte
	err = RetryInc(func() (err error) {
		req, err = http.NewRequest("POST", fmt.Sprintf("%v?%v", httpUrl, values.Encode()), bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Add("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Add(k, v)
		}

		resp, err = client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err = ioutil.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			if err != nil {
				return errors.New(fmt.Sprintf("request %v with json %v and Header %v failed, StatusCode:%v, parse body error:%v",
					httpUrl, string(body), req.Header, resp.StatusCode, err.Error()))
			}
			return errors.New(fmt.Sprintf("request %v with json %v and Header %v failed, Status:%v, msg:%v",
				httpUrl, string(body), req.Header, resp.Status, string(data)))
		}
		return nil
	}, "HttpPostJsonWithHeaders", RETRY_COUNT, RETRY_INTERVAL, RETRY_INTERVAL_INCREMENT)

	return data, err
}

func RetryInc(operation func() error, name string, attempts int, retryWaitSeconds int, retryWaitIncSeconds int) (err error) {
	for i := 0; ; i++ {
		err = operation()
		if err == nil {
			if i > 0 {
				glog.Infof("retry #%d %v finally succeed", i, name)
			}
			return nil
		}
		glog.Errorf("retry #%d %v, error: %s", i, name, err)

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(time.Second * time.Duration(retryWaitSeconds))
		retryWaitSeconds = retryWaitSeconds + retryWaitIncSeconds
	}
	return err
}
