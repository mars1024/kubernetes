// +build linux

package cni

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"strings"

	"github.com/golang/glog"
	"golang.org/x/exp/inotify"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/network"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/validation"
)

const (
	// confCNIServiceAddressName is name of cni service address in conf file
	confCNIServiceAddressName = "cniServiceAddress"
	// configMapCNIServiceAddress config map data key
	configMapCNIServiceAddress = "cni-service-addr"
	// configMapNameSpaceOfCNI config map nameSpace
	configMapNameSpaceOfCNI = "kube-system"
	// configMapNameOfCNI config map name
	configMapNameOfCNI = "sigma-slave-config"
)

func UpdateCNIServiceAddress(plugins []network.NetworkPlugin, networkPluginName string, kubeClient clientset.Interface) {
	if !strings.EqualFold(networkPluginName, CNIPluginName) {
		glog.Errorf("not support network plugin %s", networkPluginName)
		return
	}

	pluginMap := map[string]network.NetworkPlugin{}

	for _, plugin := range plugins {
		name := plugin.Name()
		if errs := validation.IsQualifiedName(name); len(errs) != 0 {
			glog.Errorf("network plugin has invalid name: %q: %s", name, strings.Join(errs, ";"))
			continue
		}

		if _, found := pluginMap[name]; found {
			glog.Errorf("network plugin %q was registered more than once", name)
			continue
		}
		pluginMap[name] = plugin
	}

	chosenPlugin := pluginMap[networkPluginName]
	if chosenPlugin != nil {
		cniPlugin, ok := chosenPlugin.(*cniNetworkPlugin)
		if !ok {
			glog.Errorf("chosen net work %s is not cni network plugin", chosenPlugin.Name())
			return
		}
		listWatchCNIServiceAddress(kubeClient, cniPlugin.confDir)
		go cniPlugin.updateCNINetWork()
	} else {
		glog.Errorf("Network plugin %q not found.", networkPluginName)
	}
}

// listWatchCNIServiceAddress  list and watch cni service address through api server, when cni service address
// add or update, update cniConf file.
func listWatchCNIServiceAddress(c clientset.Interface, confDir string) {
	glog.V(4).Info("get cni service address")
	configMapFIFO := cache.NewFIFO(cache.MetaNamespaceKeyFunc)
	fieldSelector := fields.Set{api.ObjectNameField: configMapNameOfCNI}.AsSelector()

	configMapLW := cache.NewListWatchFromClient(c.CoreV1().RESTClient(), "configmaps",
		configMapNameSpaceOfCNI, fieldSelector)
	r := cache.NewReflector(configMapLW, &v1.ConfigMap{}, configMapFIFO, 0)
	go r.Run(wait.NeverStop)

	popFunc := func() {
		_, err := configMapFIFO.Pop(func(obj interface{}) error {
			_, err := parseConfigMapFromFIFO(obj, confDir)
			return err
		})
		if err != nil {
			glog.Errorf("cni service pop err %v", err)
		}
	}
	go wait.Forever(popFunc, 0)
}

// parseConfigMapFromFIFO parse config map which pop from FIFO.
func parseConfigMapFromFIFO(obj interface{}, confDir string) (string, error) {
	configMap, ok := obj.(*v1.ConfigMap)
	if !ok {
		return "", fmt.Errorf("cni service convert to v1.ConfigMap failed %v", obj)
	}

	glog.V(2).Infof(" cni service data is %v", configMap.Data)
	if len(configMap.Data) == 0 {
		return "", fmt.Errorf("cni service data is empty")
	}

	cniAddress := configMap.Data[configMapCNIServiceAddress]
	glog.V(0).Infof(" cni service cni address is %q", cniAddress)
	if cniAddress != "" {
		if err := updateCNIConf(cniAddress, confDir); err != nil {
			return cniAddress, err
		}
		return cniAddress, nil
	}
	return "", fmt.Errorf("cni service data: %v not contain cni service address", configMap.Data)
}

// updateCNIConf update cni conf, add cni address.
func updateCNIConf(cniServiceAddress, confDir string) error {
	if cniServiceAddress == "" {
		return fmt.Errorf("cni service address is empty")
	}

	if confDir == "" {
		return fmt.Errorf("cni conf is empty")
	}

	network, err := getDefaultCNINetwork(confDir, nil)
	if err != nil {
		glog.Errorf("Unable to update cni config: %v", err)
		return err
	}

	rawList := make(map[string]interface{})
	// In the current scenario, network.NetWorkConfig.Plugins only contain one NetworkConfig.
	// network.NetworkConfig can't be nil, len(network.NetworkConfig.Plugins) can't be zero,
	// it ensured by getDefaultCNINetwork().
	if err := json.Unmarshal(network.NetworkConfig.Plugins[0].Bytes, &rawList); err != nil {
		glog.Errorf("update cni conf error parsing configuration list: %v", err)
		return err
	}
	rawList[confCNIServiceAddressName] = cniServiceAddress

	fileContext, err := json.Marshal(rawList)
	if err != nil {
		glog.Errorf("Marshal conf file context %+v err :%v", rawList, err)
		return err
	}

	err = ioutil.WriteFile(network.confFileName, fileContext, 0644)
	if err != nil {
		glog.Errorf("write cni conf file err: %v", err)
		return err
	}
	return nil
}

// updateCNINetWork watch conf dir, when file create or modify, update plugin network config.
func (plugin *cniNetworkPlugin) updateCNINetWork() {
	w, err := inotify.NewWatcher()
	if err != nil {
		glog.Fatalf("unable to create inotify for cni : %v", err)
	}
	defer w.Close()

	err = w.AddWatch(plugin.confDir, inotify.IN_CREATE|inotify.IN_CLOSE_WRITE)
	if err != nil {
		glog.Fatalf("unable to create inotify for path %q: %v", plugin.confDir, err)
	}

	for {
		select {
		case event := <-w.Event:
			glog.V(2).Infof("cni plugin :%s watch conf event (%+v)", plugin.Name(), event)
			plugin.syncNetworkConfig()
		case err = <-w.Error:
			glog.Errorf("error while watching %q: %v", plugin.confDir, err)
		}
	}
}
