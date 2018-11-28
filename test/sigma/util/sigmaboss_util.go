package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
)

// RebuildTask response from rebuildApi api
type RebuildTask struct {
	TaskID  string `json:"taskId"`
	Success bool   `json:"success"`
}

// RebuildResult response from queryRebuildApi api
type RebuildResult struct {
	Success     bool     `json:"success"`
	Finish      bool     `json:"finish"`
	TaskID      string   `json:"taskID"`
	SuccessApps []string `json:"successApps"`
	FailureApps []string `json:"failureApps"`
}

// RebuildSigma3Pod rebuild sigma3 pod object from app name and site
func RebuildSigma3Pod(appName, site string) (string, error) {
	url := fmt.Sprintf("http://daily.sigmaboss.alibaba-inc.com/api/sigma3/rebuildApi.htm?env=TEST&appNames=%s&cellName=%s&operate=startBuilding&kubeCluster=sigma3_gray_test", appName, site)
	glog.Infof(url)
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Get error from rebuildApi api, error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var task RebuildTask
	err = json.Unmarshal(respBody, &task)
	if err != nil {
		glog.Errorf("parse json rebuildApi response error, %v", err)
		return "", err
	}
	if task.Success {
		return task.TaskID, nil
	}
	return "", nil
}

// ShutDownRebuildSigma3Pod shut down the rebuild process
func ShutDownRebuildSigma3Pod(appName, site string) (string, error) {
	url := fmt.Sprintf("http://daily.sigmaboss.alibaba-inc.com/api/sigma3/rebuildApi.htm?env=TEST&appNames=%s&cellName=%s&operate=finishStockBuilding&kubeCluster=sigma3_gray_test", appName, site)
	glog.Infof(url)
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Get error from rebuildApi api, error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var task RebuildTask
	err = json.Unmarshal(respBody, &task)
	if err != nil {
		glog.Errorf("parse json rebuildApi response error, %v", err)
		return "", err
	}
	if task.Success {
		return task.TaskID, nil
	}
	return "", nil
}

// QuerySigma3RebuildPodWithTimeout get sigma3.1 rebuild pod from task ID, if timeout, return nil
func QuerySigma3RebuildPodWithTimeout(taskID, appName string, timeout time.Duration) error {
	if taskID == "" {
		return fmt.Errorf("query rebuild task ID should not be empty")
	}
	t := time.Now()
	for {
		result, err := querySigma3PodRebuild(taskID)
		if err != nil {
			glog.Errorf("check rebuild pod api status err: %s", err.Error())
			return err
		}
		if result.Success && result.Finish {
			glog.Infof("rebuild pod task[%s] is finished", taskID)
			for _, successApp := range result.SuccessApps {
				if successApp == appName {
					return nil
				}
			}
			glog.Errorf("rebuild successful apps[%v] not include %v", result.SuccessApps, appName)
			return fmt.Errorf("rebuild successful apps[%v] not include %v", result.SuccessApps, appName)
		}
		glog.Infof("rebuild pod task[%s] is not finished!", taskID)
		if time.Since(t) >= timeout {
			glog.Errorf("query rebuild pod timeout, task id is %s", taskID)
			break
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout to get the query task[%s]", taskID)
}

// querySigma3PodRebuild query sigma3.1 pod rebuild result from task ID
func querySigma3PodRebuild(taskID string) (*RebuildResult, error) {
	url := fmt.Sprintf("http://daily.sigmaboss.alibaba-inc.com/api/sigma3/queryRebuildApi.htm?taskID=%s", taskID)
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Get error from queryRebuildApi api, error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result RebuildResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		glog.Errorf("parse json queryRebuildApi response error, %v", err)
		return nil, err
	}
	return &result, nil
}

// OpenStockBuildingForSigma3Pod sigmaboss open stock build, so that 3.1 pod can rollback to 2.0
func OpenStockBuildingForSigma3Pod(appName, site string) (string, error) {
	url := fmt.Sprintf("http://daily.sigmaboss.alibaba-inc.com/api/sigma3/rebuildApi.htm?env=TEST&appNames=%s&cellName=%s&operate=openStockBuilding&kubeCluster=sigma3_gray_test", appName, site)
	glog.Infof(url)
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Get error from openStockBuilding api, error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var task RebuildTask
	err = json.Unmarshal(respBody, &task)
	if err != nil {
		glog.Errorf("parse json openStockBuilding response error, %v", err)
		return "", err
	}
	if task.Success {
		return task.TaskID, nil
	}
	return "", nil
}
