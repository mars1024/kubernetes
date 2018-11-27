package kubelet

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

// GetPodHostNameTemplate get pod hostname template.
func GetPodHostNameTemplate(pod *v1.Pod) (string, error) {
	if pod == nil {
		return "", fmt.Errorf("pod is nil")
	}
	if len(pod.Annotations) == 0 {
		return "", fmt.Errorf("pod %s annotation len is zero", format.Pod(pod))
	}

	hostnameTemplate, ok := pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate]
	if !ok {
		return "", fmt.Errorf("pod %s has no annotation :%s",
			format.Pod(pod), sigmak8sapi.AnnotationPodHostNameTemplate)
	}
	glog.V(4).Infof("pod %s has annotation : %s,value is %s",
		format.Pod(pod), sigmak8sapi.AnnotationPodHostNameTemplate, hostnameTemplate)

	return hostnameTemplate, nil
}

// GeneratePodHostNameAndDomainByHostNameTemplate creates a hostname and domain name for a pod according to hostnameTemplate.
func GeneratePodHostNameAndDomainByHostNameTemplate(pod *v1.Pod, podIP string) (string, string, bool, error) {
	// step 1: get hostname template
	hostNameTemplate, err := GetPodHostNameTemplate(pod)
	if err != nil {
		glog.Warning(err.Error())
		return "", "", false, nil
	}
	if hostNameTemplate == "" {
		return "", "", false, fmt.Errorf("pod annotation contain %s, but it value hostname template is empty",
			sigmak8sapi.AnnotationPodHostNameTemplate)
	}

	// step 2：parse podIP, convert 1.1.1.1 to 001001001001
	ipAddress, err := ParseIPToString(podIP)
	if err != nil {
		return "", "", false, err
	}

	// step 3: parse hostname template
	// extension other field
	execTemplateData := &map[string]interface{}{
		"IpAddress": ipAddress,
	}
	buf := &bytes.Buffer{}
	err = template.Must(template.New("HostName template").Parse(hostNameTemplate)).Execute(buf, execTemplateData)
	if err != nil {
		return "", "", false, fmt.Errorf("generate hostname according to hostname template %s err :%v", hostNameTemplate, err)
	}

	// step 4: split to hostname and host domain
	hostDomain := ""
	hostNameAndDomain := strings.SplitN(buf.String(), ".", 2)
	hostName := hostNameAndDomain[0]
	if len(hostNameAndDomain) == 2 {
		hostDomain = hostNameAndDomain[1]
	}

	// step 5：verify hostname and host domain
	if len(hostName) > 0 {
		if msgs := utilvalidation.IsDNS1123Label(hostName); len(msgs) != 0 {
			return "", "", false, fmt.Errorf("pod Hostname %q is not a valid DNS label: %s", hostName, strings.Join(msgs, ";"))
		}
	}
	if len(hostDomain) > utilvalidation.DNS1123LabelMaxLength {
		return "", "", false, fmt.Errorf("pod host domain %q is not a valid DNS label: %s", hostDomain,
			utilvalidation.MaxLenError(utilvalidation.DNS1123LabelMaxLength))
	}

	return hostName, hostDomain, true, nil
}

// ParseIPToString parse ip to string
// 1.1.1.1 => 001001001001
//TODO consider ipv6
func ParseIPToString(podIP string) (string, error) {
	if nil == net.ParseIP(podIP) {
		return "", fmt.Errorf("ip :%s is not a valid textual representation of an IP address", podIP)
	}
	podIPArray := strings.Split(podIP, ".")
	if len(podIPArray) < 4 {
		return "", fmt.Errorf("ip: %s is invalid", podIP)
	}
	return fmt.Sprintf("%03s%03s%03s%03s", podIPArray[0], podIPArray[1], podIPArray[2], podIPArray[3]), nil
}

// PodHaveCNIAllocatedFinalizer judge pod have cni allocated finalizer.
func PodHaveCNIAllocatedFinalizer(pod *v1.Pod) bool {
	if pod == nil {
		glog.V(4).Info("pod is nil")

	}
	if len(pod.Finalizers) == 0 {
		glog.V(4).Infof("pod %s finalizer len is zero", format.Pod(pod))
		return false
	}

	for _, finalizer := range pod.Finalizers {
		if strings.EqualFold(finalizer, sigmak8sapi.FinalizerPodCNIAllocated) {
			glog.V(4).Infof("pod %s have finalizer : %s", format.Pod(pod), sigmak8sapi.FinalizerPodCNIAllocated)
			return true
		}
	}
	return false
}

// filterCgroupShouldPreservePods returns the given pods which the status manager
// does not consider failed or succeeded and container should not contain cni allocated finalizer
func (kl *Kubelet) filterCgroupShouldPreservePods(pods []*v1.Pod) []*v1.Pod {
	var filteredPods []*v1.Pod
	for _, p := range pods {
		if kl.podIsTerminated(p) && !PodHaveCNIAllocatedFinalizer(p) {
			continue
		}
		filteredPods = append(filteredPods, p)
	}
	return filteredPods
}
