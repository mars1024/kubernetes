package ant_migrate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	dockertypes "github.com/docker/docker/api/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/test/e2e/framework"
	antsigma "k8s.io/kubernetes/test/sigma/ant-sigma-bvt"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

const (
	Etcd_key_tmpl_allocplans       = "/nodes/allocplans/%v/%v/%v"     //etcd:/nodes/allocplans/$site/$sn/$slotId
	Etcd_key_tmpl_allocplans_bak   = "/nodes/allocplans_bak/%v/%v/%v" //etcd:/nodes/allocplans/$site/$sn/$slotId
	Etcd_key_tmpl_instances_config = "/instances/config/%v/%v/%v"     //site, hostSn, instanceSn
	Etcd_key_tmpl_slotstates       = "/nodes/slotstates/%v/%v/%v"     //etcd:/nodes/slotstates/$site/$sn/$slotId
)

// ContainerLifeCycle() check container lifecycle after rebuild.
func ContainerLifeCycle(f *framework.Framework, pod *v1.Pod) {
	//stop pod
	By("Stop sigma3.1 pod.")
	err := antsigma.StopOrStartSigmaPod(f.ClientSet, pod, k8sapi.ContainerStateExited)
	Expect(err).To(BeNil(), "[Sigma3.1 LifeCycle] Stop sigma3.1 pod failed.")
	//start pod
	By("Start sigma3.1 pod.")
	err = antsigma.StopOrStartSigmaPod(f.ClientSet, pod, k8sapi.ContainerStateRunning)
	Expect(err).To(BeNil(), "[Sigma3.1 LifeCycle] Start sigma3.1 pod failed.")
	antsigma.CheckDNSPolicy(f, pod)
}

// CheckSigma20ResourceReomoved() check sigma20 reource removed from etcd.
func CheckSigma20ResourceReomoved(f *framework.Framework, pod *v1.Pod, site string) {
	// check 2.0 allocPlan/config/slot states is removed.
	nodeInfo := GetNodeInfo(f, pod.Spec.NodeName)
	framework.Logf("Node: %#v", DumpJson(nodeInfo))
	hostSn := nodeInfo.Labels[k8sapi.LabelNodeSN]
	Expect(hostSn).NotTo(BeEmpty(), "hostSn must be added in node labels:%v", nodeInfo.Labels)
	hostSn = swarm.GetNodeSN(hostSn)
	Expect(hostSn).NotTo(BeEmpty(), "query hostSN from armory failed, hostSn:%v", hostSn)
	slotId := pod.Labels["custom.k8s.alipay.com/slot-id"]
	Expect(slotId).NotTo(BeEmpty(), "SlotId must be specified in pod labels.")

	By("check 2.0 container's allocplan is removed from etcd.")
	allocplan := fmt.Sprintf(Etcd_key_tmpl_allocplans, site, hostSn, slotId)
	framework.Logf("check etcd allocplan key %s", allocplan)
	val, err := swarm.EtcdGet(allocplan)
	Expect(val).To(BeNil(), "2.0 container's allocPlan is not removed from etcd.")
	Expect(err).NotTo(HaveOccurred(), "2.0 container's etcd info is not removed")

	By("check 2.0 container's allocplan backup in etcd.")
	allocplan_bak := fmt.Sprintf(Etcd_key_tmpl_allocplans_bak, site, hostSn, slotId)
	framework.Logf("check etcd allocplan bak key %s", allocplan_bak)
	val, err = swarm.EtcdGet(allocplan_bak)
	Expect(val).NotTo(BeNil(), "2.0 container's allocPlan is not backup in etcd.")
	Expect(err).NotTo(HaveOccurred(), "2.0 container's allocplan is not backup.")

	By("check 2.0 contaner's config is removed.")
	config_key := fmt.Sprintf(Etcd_key_tmpl_instances_config, site, hostSn, pod.Name)
	framework.Logf("check etcd container config key %s", config_key)
	val, err = swarm.EtcdGet(config_key)
	Expect(val).To(BeNil(), "2.0 container's config is not removed from etcd.")
	Expect(err).NotTo(HaveOccurred(), "2.0 container's config info is not removed.")

	By("check 2.0 contaner's config is removed.")
	slotstates_key := fmt.Sprintf(Etcd_key_tmpl_slotstates, site, hostSn, pod.Name)
	framework.Logf("check etcd container slotstates key %s", slotstates_key)
	val, err = swarm.EtcdGet(config_key)
	Expect(val).To(BeNil(), "2.0 container's slotStates is not removed from etcd.")
	Expect(err).NotTo(HaveOccurred(), "2.0 container's slotStates info is not removed.")
}

// CheckSigma31Resouce() check cpu/disk/mem/ip/volume/hostname/ip/userinfo
func CheckSigma31Resouce(f *framework.Framework, pod *v1.Pod, containerHostName string,
	containerJson *dockertypes.ContainerJSON, cpusetMode string) {
	if containerJson.Config == nil {
		framework.Logf("container config is nil, container:%#v", containerJson)
		Fail("Unexpected container info.")
	}
	labels := containerJson.Config.Labels
	Expect(labels).NotTo(BeNil(), "Unexpected container labels.")

	By("rebuild sigma3.1: check container ip.")
	oldIP := labels["ali.container_ip"]
	Expect(oldIP).NotTo(BeEmpty(), "simga2.0 container ip label is empty.")
	Expect(oldIP).To(Equal(pod.Status.PodIP), "sigma3.1 container ip addr should be same as sigma2.0 container")

	By("rebuild sigma3.1: check container hostname should not change")
	cmd := []string{"hostname"}
	framework.Logf("pod:%#v, cmd:%#v", DumpJson(pod), cmd)
	stdout, _, err := antsigma.RetryExec(f, pod, cmd, "check_hostname", 10, 2)
	framework.Logf("Exec %v hostName:%#v, stdout: %v, err:%v", pod.Name, containerHostName, stdout, err)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod hostname error")
	Expect(stdout).To(Equal(containerHostName), "3.1 pod hostname is not equal with 2.0 container")

	By("rebuild sigma3.1: check container memory should same as pod.")
	cmd = []string{"cat", "/proc/meminfo"}
	stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_mem", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "get 3.1 pod memory error")
	memory := antsigma.Atoi64(labels["ali.MemoryHardlimit"], 0)
	isEqual := antsigma.CompareMemory(memory, stdout)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod mem is not equal with input.")
	Expect(isEqual).To(BeTrue(), "check 3.1 pod mem is not equal with input.")

	if cpusetMode == CPUSetModeDefault {
		By("rebuild sigma3.1: check container cpu should same as pod.")
		cmd = []string{"cat", "/proc/cpuinfo"}
		stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_cpu", 10, 2)
		Expect(err).NotTo(HaveOccurred(), "get 3.1 pod cpu error")
		cpu := antsigma.Atoi64(labels["ali.CpuCount"], 0)
		isEqual = antsigma.CompareCPU(cpu, stdout)
		Expect(err).NotTo(HaveOccurred(), "check 3.1 pod cpu is not equal with input.")
		Expect(isEqual).To(BeTrue(), "check 3.1 pod cpu is not equal with input.")
	} else if cpusetMode == CPUSetModeShare {
		By("rebuild sigma3.1: check container cpu should same as pod.")
		cmd = []string{"cat", "/proc/cpuinfo"}
		stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_cpu", 10, 2)
		Expect(err).NotTo(HaveOccurred(), "get 3.1 pod cpu error")
		cpu := GetNodeCPU(f, pod)
		isEqual = antsigma.CompareCPU(cpu, stdout)
		Expect(err).NotTo(HaveOccurred(), "check 3.1 pod cpu is not equal with input.")
		Expect(isEqual).To(BeTrue(), "check 3.1 pod cpu is not equal with input.")
	}

	By("rebuild sigma3.1: check container diskSize should same as pod.")
	cmd = []string{"df", "-h"}
	stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_disk", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "get 3.1 pod diskSize error")
	disk := antsigma.Quota2Byte(labels["ali.DiskSize"])
	isEqual = antsigma.CompareDisk(disk, stdout)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod diskSize is not equal with input.")
	Expect(isEqual).To(BeTrue(), "check 3.1 pod diskSize is not equal with input.")

	By("rebuild sigma3.1: check container volumes and binds.")
	CheckSigma31Volumes(pod, containerJson)

	By("rebuild sigma3.1: check container network settings.")
	antsigma.CheckNewWorkSettings(f, pod)

	By("rebuild sigma3.1: check container armory info.")
	antsigma.CheckArmory(pod)

	By("rebuild sigma3.1: check container dnsConfig")
	antsigma.CheckDNSPolicy(f, pod)

	By("rebuild sigma3.1: check container extra host alias.")
	CheckHostAlias(f, pod)
}

func CheckHostAlias(f *framework.Framework, pod *v1.Pod) {
	if len(pod.Spec.HostAliases) == 0 {
		return
	}
	By("rebuild sigma3.1: get container hosts.")
	cmd := []string{"cat", "/etc/hosts"}
	stdout, _, err := antsigma.RetryExec(f, pod, cmd, "check_hosts", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "get 3.1 pod hosts error")
	hosts := strings.Split(stdout, "\n")
	framework.Logf("Hosts:%#v, pod alias:%#v", hosts, pod.Spec.HostAliases)
	for _, hostAlias := range pod.Spec.HostAliases {
		for _, hostname := range hostAlias.Hostnames {
			str := fmt.Sprintf("%s %s", hostAlias.IP, hostname)
			framework.Logf("Alias:%v", str)
			Expect(antsigma.SliceContains(hosts, str, nil)).To(BeTrue(), "Check host alias failed.")
		}
	}
}

func CheckRebuildChanges(f *framework.Framework, pod *v1.Pod, publicKey string) {
	By("rebuild sigma3.1: check user account")
	cmd := []string{"cat", "/etc/passwd"}
	stdout, _, err := antsigma.RetryExec(f, pod, cmd, "check_user", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod user account error")
	if !strings.Contains(stdout, InjectUser) {
		framework.Logf("Exec %v cmd output: %s", stdout, pod.Name)
		Fail("sigma3test account is not passed to 3.1 pod")
	}

	By("rebuild sigma3.1: check user public key")
	cmd = []string{"cat", "/root/.ssh/id_rsa.pub"}
	stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_public_key", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod user public key error")
	framework.Logf("Exec %v cmd output: %s", stdout, pod.Name)
	Expect(strings.Trim(stdout, "\t\n\r ")).To(Equal(strings.Trim(publicKey, "\t\n\r ")), "check 3.1 pod user public key content failed.")

	By("rebuild sigma3.1: check volume dir")
	cmd = []string{"ls", fmt.Sprintf("%s/passwd", InjectDir)}
	stdout, _, err = antsigma.RetryExec(f, pod, cmd, "check_volume_info", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "check 3.1 pod volume info error")
	framework.Logf("Exec %v cmd output: %s", stdout, pod.Name)
	if strings.Contains(stdout, "No such file or directory") {
		framework.Logf("Volume File: %v", stdout)
		Fail("public key is not passed to 3.1 pod")
	}
}

func CheckSigma31Volumes(pod *v1.Pod, c *dockertypes.ContainerJSON) {
	podMounts := pod.Spec.Containers[0].VolumeMounts
	volumeMounts := make(map[string]v1.VolumeMount, len(podMounts))
	for idx := range podMounts {
		volumeMounts[podMounts[idx].MountPath] = podMounts[idx]
	}

	for _, mount := range c.Mounts {
		target := mount.Destination
		_, found := volumeMounts[target]
		if !found {
			framework.Logf("pod %v target %v, mounts:%#v", pod.Name, target, DumpJson(volumeMounts))
		}
		Expect(found).To(BeTrue(), "Unexpect volume mounts.")
	}
}

func splitBindRawSpec(raw string) ([]string, error) {
	if strings.Count(raw, ":") > 2 {
		return nil, fmt.Errorf("invalid bind spec: %v", raw)
	}

	arr := strings.SplitN(raw, ":", 3)
	if arr[0] == "" {
		return nil, fmt.Errorf("invalid bind spec: %v", raw)
	}
	return arr, nil
}

// CheckContainerStatus() check 2.0 container is stopped and 3.1 container is running.
func CheckContainerStatus(hostIP, container20ID, container31ID string) {
	By("check 2.0 container deleted")
	// log into slave node and check container status, container should be stopped
	runOutput := util.GetDockerPsOutput(hostIP, container20ID)
	framework.Logf("Container20Id:%v, hostIP:%v, outPut:%v", container20ID, hostIP, runOutput)
	if runOutput != "" {
		Fail("2.0 container should be deleted.")
	}

	By("check 3.1 container is up")
	runOutput = util.GetDockerPsOutput(hostIP, container31ID)
	framework.Logf("Container31Id:%v, hostIP:%v, outPut:%v", container31ID, hostIP, runOutput)
	if !strings.Contains(runOutput, "Up") {
		Fail("3.0 container status is not up, but we expect it should be that")
	}
}

func GetNodeInfo(f *framework.Framework, hostSn string) *v1.Node {
	nodeInfo, err := f.ClientSet.CoreV1().Nodes().Get(hostSn, metav1.GetOptions{})
	Expect(err).To(BeNil(), "get nodeInfo failed")
	return nodeInfo
}

// GetSigmaContainerInfo() get quotaId/adminUID/cpusets in sigma container.
func GetSigma20ContainerInfo(hostIP, containerID string) (string, string, string) {
	By("inspect container info.")
	containerJson, err := swarm.InspectContainer(containerID)
	Expect(err).NotTo(HaveOccurred(), "inspect 2.0 container failed.")
	Expect(containerJson).NotTo(BeNil(), "inspect 2.0 container failed, nil value.")

	By("get the container QuotaID")
	Expect(len(containerJson.Config.Labels)).NotTo(BeZero(), "container label is null")
	containerQuotaID := containerJson.Config.Labels["QuotaId"]
	Expect(containerQuotaID).NotTo(BeEmpty(), "container quotaId is empty!")

	By("get the container ali_admin_uid")
	Expect(len(containerJson.Config.Env)).NotTo(BeZero(), "container env is null")
	containerAdminUID := swarm.GetEnv(containerJson.Config.Env, "ali_admin_uid")
	Expect(containerAdminUID).NotTo(BeEmpty(), "container admin uid is empty!")

	By("get the container cpu set")
	containerCpuSets := containerJson.HostConfig.CpusetCpus
	Expect(containerCpuSets).NotTo(BeEmpty(), "container cpuset is empty!")

	return containerQuotaID, containerAdminUID, containerCpuSets
}

// GetSigmaContainerInfo() get quotaId/adminUID/cpusets in sigma container.
func GetSigmaContainerInfo(hostIP, containerID string) (string, string, string) {
	By("sigma2.0/3.1: check container QuotaID remain")
	quotaCmd := `bash -c "docker inspect ` + containerID + ` | grep -w QuotaId | bash -c 'cut -d':' -f2'"`
	container31QuotaID := util.GetContainerInfoWithStarAgent(hostIP, quotaCmd)

	By("sigma2.0/3.1: check container ali_admin_uid remain")
	adminCmd := `bash -c "docker inspect ` + containerID + ` | grep -w ali_admin_uid | bash -c 'cut -d',' -f1'"`
	container31AdminUID := util.GetContainerInfoWithStarAgent(hostIP, adminCmd)

	By("sigma2.0/3.1: check container cpu set remain")
	cpuSetCmd := `bash -c "docker inspect ` + containerID + ` | grep -w CpusetCpus | bash -c 'cut -d',' -f1'"`
	container31CpuSets := util.GetContainerInfoWithStarAgent(hostIP, cpuSetCmd)
	framework.Logf("Container %v, quotaId:%v, adminId:%v, cpusets:%v", containerID, container31QuotaID, container31AdminUID, container31CpuSets)
	return container31QuotaID, container31AdminUID, container31CpuSets
}

func GetNodeCPU(f *framework.Framework, pod *v1.Pod) int64 {
	pod, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	Expect(err).To(BeNil(), "Get pod info failed.")
	node, err := f.ClientSet.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	Expect(err).To(BeNil(), "Get node info failed.")
	sharePool, ok := GetNodeSharePool(node)
	Expect(ok).To(BeTrue(), "Get node share pool info failed.")
	return int64(len(sharePool))
}

func GetNodeSharePool(node *v1.Node) ([]int32, bool) {
	sharePoolstr, ok := node.Annotations[k8sapi.AnnotationNodeCPUSharePool]
	if !ok {
		return nil, false
	}
	sharePool := &k8sapi.CPUSharePool{}
	err := json.Unmarshal([]byte(sharePoolstr), sharePool)
	if err != nil {
		framework.Logf("Unmarshal share pool %v failed:%v", sharePoolstr, err)
		return nil, false
	}
	return sharePool.CPUIDs, true
}

func DumpJson(v interface{}) string {
	str, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return String(str)
}

// ToString convert slice to string without mem copy.
func String(b []byte) (s string) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pstring.Data = pbytes.Data
	pstring.Len = pbytes.Len
	return
}

var upgradeEnv = []v1.EnvVar{
	{
		Name:  "SIGMA3_UPGRADE_TEST",
		Value: "test",
	},
}

var upgradeEnv2 = []v1.EnvVar{
	{
		Name:  "SIGMA3_UPGRADE_TEST2",
		Value: "test2",
	},
}

func GetNodeName(f *framework.Framework) string {
	Expect(f).NotTo(BeNil(), "init framework failed.")
	nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	Expect(err).NotTo(HaveOccurred(), "List nodes failed.")
	Expect(len(nodes.Items)).NotTo(BeZero(), "No nodes in cluster.")
	return nodes.Items[0].Name
}
