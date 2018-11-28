package controller_test

import (
	"net"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	corev1 "k8s.io/api/core/v1"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"gitlab.alipay-inc.com/sigma/controller-manager/pkg/alipaymeta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"gitlab.alipay-inc.com/sigma/controller-manager/pkg/xvip"
	"strconv"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

func newService(namespace string, name string, portName []string, ports []int32, protocols []corev1.Protocol, ) *corev1.Service {
	var p []corev1.ServicePort
	for idx, pp := range ports {
		p = append(p, corev1.ServicePort{
			Name:       portName[idx],
			Port:       pp,
			Protocol:   protocols[idx],
			TargetPort: intstr.FromInt(int(pp)),
		})
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				alipayapis.ServiceProvisioner:            "xvip",
				alipayapis.AnnotationXvipAppGroup:        "robot_unithost",
				alipayapis.AnnotationXvipApplyUser:       "leowang.wangl",
				alipayapis.AnnotationXvipAppId:           "316263",
				alipayapis.AnnotationXvipBuType:          "cross_domain",
				alipayapis.AnnotationXvipHealthcheckType: "TCPCHECK",
				alipayapis.AnnotationXvipReqAvgSize:      "1024",
				alipayapis.AnnotationXvipQpsLimit:        "1024",
				alipayapis.AnnotationXvipLbName:          "SLB-CN-HANGZHOU-TEST-ZONE1.ET15",
				alipayapis.ServiceDNSRRZone:              "alipay.net",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: p,
		},
	}
}

func newEndpoints(namespace string, name string, addresses, notReadyAddresses []string) *corev1.Endpoints {
	var epAddr []corev1.EndpointAddress
	for _, addr := range addresses {
		epAddr = append(epAddr, corev1.EndpointAddress{
			IP: addr,
		})
	}
	var epNotReadyAddr []corev1.EndpointAddress
	for _, addr := range notReadyAddresses {
		epNotReadyAddr = append(epNotReadyAddr, corev1.EndpointAddress{
			IP: addr,
		})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses:         epAddr,
				NotReadyAddresses: epNotReadyAddr,
			},
		},
	}
}

func checkServiceStatus(client clientset.Interface, service *corev1.Service) wait.ConditionFunc {
	return func() (bool, error) {
		svc, err := client.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		framework.Logf("service[%s] load balancer ingress is %#v", service.Name, svc.Status.LoadBalancer.Ingress)
		if len(svc.Status.LoadBalancer.Ingress) > 0 && net.ParseIP(svc.Status.LoadBalancer.Ingress[0].IP) != nil {
			return true, nil
		}
		return false, nil
	}
}

func dump(obj interface{}) string {
	bytes, _ := json.Marshal(obj)
	return string(bytes)
}

var (
	e2eName     = "boss"
	e2eToken    = "xxxxxx"
	e2eEndpoint = "http://cloudnet-eu95-0.gz00b.stable.alipay.net"
	// docs: http://net-dev.alibaba.net/xvip/vip_management
)

var _ = Describe("[ant][sigma-alipay-controller][xvip]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	/*
	测试 ip：
	  10.150.200.102
	  10.150.200.106
	 */
	xvipClient, _ := xvip.New(e2eEndpoint, e2eName, e2eToken)
	It("[sigma-alipay-controller][xvip][smoke] create a service with xvip provisioner, should register in xnet", func() {
		By("create a endpint and a service with xvip not created")

		name := "xvip-controller-e2e-" + time.Now().Format("20160607123450")
		ep := newEndpoints(f.Namespace.Name, name, []string{"10.150.200.102"}, nil)
		svc := newService(f.Namespace.Name, name,
			[]string{"http", "https"},
			[]int32{80, 443},
			[]corev1.Protocol{corev1.ProtocolTCP, corev1.ProtocolTCP},
		)

		endpoint, err := f.ClientSet.CoreV1().Endpoints(f.Namespace.Name).Create(ep)
		Expect(err).NotTo(HaveOccurred(), "create endpoint err")
		glog.Infof("create endpoint config:%v", endpoint)

		defer func() {
			f.ClientSet.CoreV1().Endpoints(f.Namespace.Name).Delete(name, &metav1.DeleteOptions{})
		}()

		service, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(svc)
		Expect(err).NotTo(HaveOccurred(), "create service err")
		glog.Infof("create service config:%v", service)

		defer func() {
			f.ClientSet.CoreV1().Services(f.Namespace.Name).Delete(name, &metav1.DeleteOptions{})
		}()

		endpoint.Subsets = []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "10.150.200.102",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "http",
						Port:     80,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:     "https",
						Port:     443,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}
		endpoint, err = f.ClientSet.CoreV1().Endpoints(endpoint.Namespace).Update(endpoint)
		Expect(err).NotTo(HaveOccurred(), "update endpoint err")
		glog.Infof("update endpoint config:%v", endpoint)

		By("wait until service have load balance IP")
		err = wait.PollImmediate(time.Second*3, time.Minute*3, checkServiceStatus(f.ClientSet, service))
		Expect(err).NotTo(HaveOccurred(), "service xvip is not working")

		var vip string
		service, err = f.ClientSet.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get service err")
		if len(service.Status.LoadBalancer.Ingress) > 0 && net.ParseIP(service.Status.LoadBalancer.Ingress[0].IP) != nil {
			vip = service.Status.LoadBalancer.Ingress[0].IP
		}
		Expect(vip).ShouldNot(BeEmpty(), "vip not found")

		getService, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Get(service.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getService.Status.LoadBalancer.Ingress).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getService.Finalizers).Should(ContainElement(ContainSubstring(alipaymeta.XvipFinalizer)),
			"XvipFinalizer should be register into service.finalizers")

		spec := &xvip.XVIPSpec{
			AppGroup: "robot_unithost",
			//ApplyUser:       "leowang.wangl",
			//AppId:           "316263",
			VipBuType:       "cross_domain",
			Ip:              vip,
			Port:            0,
			Protocol:        corev1.ProtocolTCP,
			HealthcheckType: "TCPCHECK",
			ReqAvgSize:      1024 * 1024,
			QpsLimit:        1024,
		}
		if vip != "" {
			By("check service dns has been registered")

			By("check xnet info")
			specs, err := xvipClient.GetRsInfo(spec)
			Expect(err).Should(BeNil(), "get xvip should be success")
			Expect(specs).Should(HaveLen(2), "should found with 2 xvip specs")
			for _, s := range specs {
				framework.Logf("service[%s] one load balancer spec is %s", service.Name, dump(s))
				Expect(s.Ip).Should(Equal(vip), "vip should be same")
				Expect([]string{"80", "443"}).Should(ContainElement(strconv.Itoa(int(s.Port))), "port should be in 80/443")
				Expect(s.RealServerList).Should(HaveLen(1), "real server mismatch")
				framework.Logf("xvip port: %d, real server list %s", s.Port, dump(s.RealServerList))
				if len(s.RealServerList) > 0 {
					rs := s.RealServerList[0]
					Expect(rs.Ip).Should(Equal("10.150.200.102"), "real server ip should be 10.150.200.102")
					Expect(rs.Status).Should(Equal(xvip.StatusEnable), "real server ip should be enable")
				}
			}

			By("check xnet info when add pod")
			Expect(endpoint.Subsets).Should(HaveLen(1), "endpoint len must be 1")
			newRsIp := "10.150.200.106"
			if len(endpoint.Subsets) == 1 {
				endpoint.Subsets[0].Addresses = append(endpoint.Subsets[0].Addresses,
					corev1.EndpointAddress{
						IP: newRsIp,
					})
			}
			endpoint, err = f.ClientSet.CoreV1().Endpoints(endpoint.Namespace).Update(endpoint)
			Expect(err).NotTo(HaveOccurred(), "update endpoint err")
			if err == nil {
				By("wait until xvip have a new real server")
				err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
					specs, err := xvipClient.GetRsInfo(spec)
					if err != nil {
						return false, nil
					}
					var count int
					for _, s := range specs {
						framework.Logf("xvip port: %d, real server list %s", s.Port, dump(s.RealServerList))
						for _, rs := range s.RealServerList {
							if rs.Ip == newRsIp && rs.Status == xvip.StatusEnable {
								count ++
							}
						}
					}
					if count == 2 {
						return true, nil
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred(), "service xvip when add a new real server is not working")
			}

			By("check xnet info when disable pod")

			endpoint.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "10.150.200.102",
						},
					},
					NotReadyAddresses: []corev1.EndpointAddress{
						{
							IP: newRsIp,
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "http",
							Port:     80,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "https",
							Port:     443,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			endpoint, err = f.ClientSet.CoreV1().Endpoints(endpoint.Namespace).Update(endpoint)
			Expect(err).NotTo(HaveOccurred(), "update endpoint err")
			if err == nil {
				By("wait until the new real server has been disabled ")
				err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
					specs, err := xvipClient.GetRsInfo(spec)
					if err != nil {
						return false, nil
					}
					var count int
					for _, s := range specs {
						framework.Logf("xvip port: %d, real server list %s", s.Port, dump(s.RealServerList))
						for _, rs := range s.RealServerList {
							if rs.Ip == newRsIp && rs.Status == xvip.StatusDisable {
								count ++
							}
						}
					}
					if count == 2 {
						return true, nil
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred(), "service xvip when add a new real server is not working")
			}

			By("check xnet info when remove pod")
			endpoint.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "10.150.200.102",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "http",
							Port:     80,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "https",
							Port:     443,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			endpoint, err = f.ClientSet.CoreV1().Endpoints(endpoint.Namespace).Update(endpoint)
			Expect(err).NotTo(HaveOccurred(), "update endpoint err")
			if err == nil {
				By("wait until the new real server has been deleted ")
				err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
					specs, err := xvipClient.GetRsInfo(spec)
					if err != nil {
						return false, nil
					}
					var count int
					for _, s := range specs {
						framework.Logf("xvip port: %d, real server list %s", s.Port, dump(s.RealServerList))
						for _, rs := range s.RealServerList {
							if rs.Ip == newRsIp {
								count ++
							}
						}
					}
					if count == 0 {
						return true, nil
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred(), "service xvip when add a new real server is not working")
			}

			By("check xnet info when add port")
			service, err = f.ClientSet.CoreV1().Services(service.Namespace).Get(service.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "get service err")
			service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
				Name:       "socks",
				Port:       1080,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(1080)),
			})
			service, err = f.ClientSet.CoreV1().Services(service.Namespace).Update(service)
			Expect(err).NotTo(HaveOccurred(), "update service err")
			if err == nil {
				By("wait until the xvip with new port has been created ")
				err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
					specs, err := xvipClient.GetRsInfo(spec)
					if err != nil {
						return false, nil
					}

					framework.Logf("xvip specs %s",dump(specs))
					for _, s := range specs {
						if s.Port == 1080 && s.Ip == vip {
							return true, nil
						}
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred(), "service xvip when add new port is not working")
			}

			By("check xnet info when delete port")
			service.Spec.Ports = service.Spec.Ports[0:2]
			service, err = f.ClientSet.CoreV1().Services(service.Namespace).Update(service)
			Expect(err).NotTo(HaveOccurred(), "update service err")
			if err == nil {
				By("wait until the xvip with a port has been deleted ")
				err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
					specs, err := xvipClient.GetRsInfo(spec)
					if err != nil {
						return false, nil
					}
					var found bool
					framework.Logf("xvip specs %s",dump(specs))
					for _, s := range specs {
						if s.Port == 1080 && s.Ip == vip {
							found = true
						}
					}
					if !found {
						return true, nil
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred(), "service xvip when delete a port is not working")
			}
		}

		By("delete a service should success")
		err = f.ClientSet.CoreV1().Services(f.Namespace.Name).Delete(name, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "delete service should succeed")

		if vip != "" {
			err = wait.PollImmediate(time.Second*3, time.Minute*3, func() (bool, error) {
				specs, err := xvipClient.GetRsInfo(spec)
				framework.Logf("xvip specs: %s", dump(specs))
				if err != nil {
					return false, nil
				}
				if specs == nil {
					return true, nil
				}
				return false, nil
			})
			Expect(err).NotTo(HaveOccurred(), "delete xvip is not working")
		}
	})
})
