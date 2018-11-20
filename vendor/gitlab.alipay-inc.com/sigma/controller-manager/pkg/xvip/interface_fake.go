package xvip

import (
	"fmt"
	"k8s.io/api/core/v1"
)

func NewFake() Client {
	return &fakeInterface{}
}

type fakeInterface struct {
	data    XVIPSpecList
	ipCount int
}

func (f *fakeInterface) AddVIP(spec *XVIPSpec) (ip string, err error) {
	f.ipCount += 1
	if spec.Ip == "" {
		spec.Ip = fmt.Sprintf("8.8.8.%d", f.ipCount)
	}
	f.data = append(f.data, spec)
	return spec.Ip, nil
}

func (f *fakeInterface) DeleteVIP(spec *XVIPSpec) error {
	for idx, s := range f.data {
		if s.Ip == spec.Ip && s.Port == spec.Port && s.Protocol == spec.Protocol {
			if idx+1 < len(f.data) {
				f.data = append(f.data[:idx], f.data[idx+1:]...)
				return nil
			} else {
				f.data = f.data[:idx]
				return nil
			}
		}
	}
	return fmt.Errorf("xvip spec not found: %#v", spec)
}

func (f *fakeInterface) AddRealServer(spec *XVIPSpec, rss ...*RealServer) error {
	spec = findXvipSpec(spec.Ip, spec.Port, spec.Protocol, f.data)
	spec.RealServerList = append(spec.RealServerList, rss...)
	return nil
}

func (f *fakeInterface) DeleteRealServer(spec *XVIPSpec, rss ...*RealServer) error {
	spec = findXvipSpec(spec.Ip, spec.Port, spec.Protocol, f.data)
	var newRss RealServerList
	for _, rs := range spec.RealServerList {
		r := findRealServer(rs.Ip, rs.Port, rss)
		if r == nil {
			newRss = append(newRss, rs)
		}
	}
	spec.RealServerList = newRss
	return nil
}

func (f *fakeInterface) EnableRealServer(spec *XVIPSpec, rss ...*RealServer) error {
	spec = findXvipSpec(spec.Ip, spec.Port, spec.Protocol, f.data)
	for _, rs := range spec.RealServerList {
		r := findRealServer(rs.Ip, rs.Port, rss)
		if r != nil {
			r.Status = StatusEnable
		}
	}
	return nil
}

func (f *fakeInterface) DisableRealServer(spec *XVIPSpec, rss ...*RealServer) error {
	spec = findXvipSpec(spec.Ip, spec.Port, spec.Protocol, f.data)
	for _, rs := range spec.RealServerList {
		r := findRealServer(rs.Ip, rs.Port, rss)
		if r != nil {
			r.Status = StatusDisable
		}
	}
	return nil
}

func (f *fakeInterface) GetTaskInfo(string) (*TaskInfo, error) {
	return nil, fmt.Errorf("not implement yet")
}

func (f *fakeInterface) GetRsInfo(spec *XVIPSpec) (XVIPSpecList, error) {
	return findXvipSpecList(spec.Ip, spec.Port, spec.Protocol, f.data), nil
}

func findXvipSpec(vip string, port int32, protocol v1.Protocol, specs XVIPSpecList) *XVIPSpec {
	for _, spec := range specs {
		if spec.Ip == vip && spec.Port == port && spec.Protocol == protocol {
			return spec
		}
	}
	return nil
}

func findXvipSpecList(vip string, port int32, protocol v1.Protocol, specs XVIPSpecList) XVIPSpecList {
	var xspecs XVIPSpecList
	for _, spec := range specs {
		if spec.Ip == vip && (spec.Port == port || port == 0) && (spec.Protocol == protocol || protocol == "") {
			xspecs = append(xspecs, spec)
		}
	}
	return xspecs
}

func findRealServer(ip string, port int32, list RealServerList) *RealServer {
	for _, rs := range list {
		if rs.Ip == ip && rs.Port == port {
			return rs
		}
	}
	return nil
}
