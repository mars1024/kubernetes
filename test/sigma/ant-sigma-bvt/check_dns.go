package ant_sigma_bvt

import (
	"bytes"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

//checkDNSPolicy() check dnsPolicy and dnsConfig.
func checkDNSPolicy(f *framework.Framework, pod *v1.Pod) {
	//check dnsPolicy
	expectDNSPolicy := getDNSPolicy(pod)
	getPod, _ := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	framework.Logf("DNSPolicy:%v, expect:%v, podInfo:%#v", pod.Spec.DNSPolicy, expectDNSPolicy, DumpJson(getPod))
	Expect(pod.Spec.DNSPolicy).To(Equal(expectDNSPolicy), "Check DNSPolicy, unexpected dnsPolicy")
	//check resolve.conf
	compareDNSConfig(f, pod)
}

//compareDNSConfig() compare dnsConfig and resolve.conf
func compareDNSConfig(f *framework.Framework, pod *v1.Pod) {
	podDNSConfig := getPodDNSConfig(f, pod)
	if pod.Spec.DNSPolicy == v1.DNSNone {
		framework.Logf("pod DNSConfig:%#v, resolve: %#v", pod.Spec.DNSConfig, DumpJson(podDNSConfig))
		Expect(IsEqualSlice(pod.Spec.DNSConfig.Nameservers, podDNSConfig.Servers)).To(BeTrue(), "DNSNone, Unexpected DNSConfig nameServers")
		Expect(IsEqualSlice(pod.Spec.DNSConfig.Searches, podDNSConfig.Searches)).To(BeTrue(), "DNSNone, Unexpected DNSConfig searches")
		Expect(IsEqualSlice(getDNSOptions(pod), podDNSConfig.Options)).To(BeTrue(), "DNSNone, Unexpected DNSConfig options.")
	} else if pod.Spec.DNSPolicy == v1.DNSDefault {
		nodeDNSConfig := getNodeDNSConfig(pod)
		framework.Logf("pod DNSConfig:%#v, pod resolv: %#v, node resolv:%#v", pod.Spec.DNSConfig, DumpJson(podDNSConfig), DumpJson(nodeDNSConfig))
		compareNodePodResolvAndConfig(getDNSOptions(pod), nodeDNSConfig.Options, podDNSConfig.Options)
		compareNodePodResolvAndConfig(pod.Spec.DNSConfig.Searches, nodeDNSConfig.Searches, podDNSConfig.Searches)
		compareNodePodResolvAndConfig(pod.Spec.DNSConfig.Nameservers, nodeDNSConfig.Servers, podDNSConfig.Servers)
	}
}

//compareNodePodResolvAndConfig() compare node/pod resolve.conf and pod.Spec.DNSConfig
func compareNodePodResolvAndConfig(podConfig, nodeResolv, podResolv []string) {
	framework.Logf("PodSpec: %#v, podResolve:%#v, nodeResolve:%#v", podConfig, podResolv, nodeResolv)
	if len(podConfig) == 0 {
		Expect(IsEqualSlice(podResolv, nodeResolv)).To(BeTrue(), "DNS Default, Unexcted DNConfig options.")
	} else {
		Expect(IsSubslice(podResolv, podConfig)).To(BeTrue(), "pod.Spec.DNSConfig should be sub set of pod resolv.conf")
		Expect(IsSubslice(podResolv, nodeResolv)).To(BeTrue(), "nodeResolve should be sub set of pod resolv.conf")
	}
}

//getDNSOptions() get dns options.
func getDNSOptions(pod *v1.Pod) []string {
	options := make([]string, 0)
	for _, option := range pod.Spec.DNSConfig.Options {
		options = append(options, option.Name)
	}
	return options
}

//getDNSPolicy()
func getDNSPolicy(pod *v1.Pod) v1.DNSPolicy {
	dnsConfig := pod.Spec.DNSConfig
	if len(dnsConfig.Searches) != 0 && len(dnsConfig.Options) != 0 && len(dnsConfig.Nameservers) != 0 {
		return v1.DNSNone
	}
	return v1.DNSDefault
}

//getNodeDNSConfig() get node dnsConfig
func getNodeDNSConfig(pod *v1.Pod) *runtimeapi.DNSConfig {
	framework.Logf("Get Node dnsConfig, nodeName:%v, nodeIP:%v", pod.Spec.NodeName, pod.Status.HostIP)
	resp, err := util.ResponseFromStarAgentTask("cmd://cat /etc/resolv.conf", pod.Status.HostIP, pod.Spec.NodeName)
	Expect(err).To(BeNil(), "Get Node resolv.conf failed.")
	Expect(resp).NotTo(BeEmpty(), "Get Node resolv.conf is empty.")
	framework.Logf("Node %v resolv.conf: %v", pod.Spec.NodeName, resp)
	return parseResolveConfFile(resp)
}

//getPodDNSConfig() get node dnsConfig
func getPodDNSConfig(f *framework.Framework, pod *v1.Pod) *runtimeapi.DNSConfig {
	cmd := []string{"cat", "/etc/resolv.conf"}
	stdout, _, err := GetOptionsUseExec(f, pod, cmd)
	Expect(err).To(BeNil(), "get pod resolv.conf failed.")
	framework.Logf("Pod %v Resolv.conf: %v", pod.Name, stdout)
	Expect(stdout).NotTo(BeEmpty(), "Get Pod resolv.conf is empty.")
	return parseResolveConfFile(stdout)
}

//parseResolveConfFile() parse resolve.conf.
func parseResolveConfFile(content string) *runtimeapi.DNSConfig {
	hostDNS, hostSearch, hostOptions, err := parseResolvConf(bytes.NewReader([]byte(content)))
	Expect(err).To(BeNil(), "Parse ")
	return &runtimeapi.DNSConfig{
		Servers:  hostDNS,
		Searches: hostSearch,
		Options:  hostOptions,
	}
}
