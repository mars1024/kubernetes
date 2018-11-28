package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

// LoadServiceFromFile create a service object from file
func LoadServiceFromFile(file string) (*v1.Service, error) {
	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var service *v1.Service
	err = json.Unmarshal(fileContent, &service)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// DeleteService delete Service by using k8s api, and check whether Service is really deleted within the timeout.
func DeleteService(client clientset.Interface, service *v1.Service) error {
	err := client.CoreV1().Services(service.Namespace).Delete(service.Name, metav1.NewDeleteOptions(0))
	if err != nil {
		return err
	}
	timeout := 3 * time.Minute
	t := time.Now()
	for {
		_, err := client.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), "not found") {
			framework.Logf("Service %s has been removed", service.Name)
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("Gave up waiting for Service %s is removed after %v seconds",
				service.Name, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether Service %s is removed", service.Name)
		time.Sleep(5 * time.Second)
	}
}

// GetVipserverServiceUrl returns the vipserver service url.
func GetVipserverServiceUrl(domain string, port string) string {
	return "http://" + domain + ":" + port
}

func WaitUntilServiceDeleted(client clientset.Interface, namespace, name string) {
	client.CoreV1().Services(namespace).Delete(name, nil)
	for i := 1; i <= 60; i++ {
		_, err := client.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				framework.Logf("delete svc %v/%v failed, err: %v", namespace, name, err)
			}
			break
		} else {
			framework.Logf("wait unitl svc %v/%v deleted", namespace, name)
		}
		time.Sleep(10 * time.Second)
	}
}
