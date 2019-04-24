package allocators

import (
	"fmt"
	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"reflect"
	"testing"
)

func TestCPUCntRef_Add(t *testing.T) {
	ref := CPUCntRef{}
	ref.Increase(1)
	ref.Increase(2)
	ref.Increase(3)
	ref.Increase(3)
	if !ref.CPUs().Equals(cpuset.NewCPUSet(1, 2, 3)) {
		t.Error("failed to Increase")
	}

	ref.Decrease(1)
	if !ref.CPUs().Equals(cpuset.NewCPUSet(2, 3)) {
		t.Error("failed to Decrease")

	}
}

func TestCPUCntRef_CPUs(t *testing.T) {
	data := []int{
		1, 2, 3, 3, 4,
	}
	ref := CPUCntRef{}
	for idx := range data {
		ref.Increase(data[idx])
	}
	expected := []int{1, 2, 3, 4}
	ret := reflect.DeepEqual(ref.CPUs().ToSlice(), expected)
	if !ret {
		t.Error(ret)
	}
}

func TestCPUCntRef_LeastUsedCPUSet(t *testing.T) {
	data := []int{
		0, 0, 1, 90, 3, 3, 5, 5, 1, 9, 11, 13, 15, 9, 90,
	}
	ref := CPUCntRef{}
	for idx := range data {
		ref.Increase(data[idx])
	}
	ret := ref.LeastUsedCPUSet(nil, 2)
	if !ret.IsSubsetOf(cpuset.NewCPUSet(13, 11, 15)) {
		t.Errorf("should be sub set [11, 13, 15], actual %s", ret.String())
	}
	t.Log(fmt.Sprintf("result :%+v", ret.String()))
	ret = ref.LeastUsedCPUSet(nil, 0)
	if !ret.IsEmpty() {
		t.Errorf("should not return any cpuset, actual %s", ret.String())
	}

}

var (
	LocalInfo = `{"cpuInfos":[{"cpu":0,"core":0,"socket":0},{"cpu":1,"core":1,"socket":0},{"cpu":2,"core":2,"socket":0},{"cpu":3,"core":3,"socket":0},{"cpu":4,"core":4,"socket":0},{"cpu":5,"core":5,"socket":0},{"cpu":6,"core":6,"socket":0},{"cpu":7,"core":7,"socket":0},{"cpu":8,"core":8,"socket":0},{"cpu":9,"core":9,"socket":0},{"cpu":10,"core":10,"socket":0},{"cpu":11,"core":11,"socket":0},{"cpu":12,"core":12,"socket":0},{"cpu":13,"core":13,"socket":0},{"cpu":14,"core":14,"socket":0},{"cpu":15,"core":15,"socket":0},{"cpu":16,"core":0,"socket":1},{"cpu":17,"core":1,"socket":1},{"cpu":18,"core":2,"socket":1},{"cpu":19,"core":3,"socket":1},{"cpu":20,"core":4,"socket":1},{"cpu":21,"core":5,"socket":1},{"cpu":22,"core":6,"socket":1},{"cpu":23,"core":7,"socket":1},{"cpu":24,"core":8,"socket":1},{"cpu":25,"core":9,"socket":1},{"cpu":26,"core":10,"socket":1},{"cpu":27,"core":11,"socket":1},{"cpu":28,"core":12,"socket":1},{"cpu":29,"core":13,"socket":1},{"cpu":30,"core":14,"socket":1},{"cpu":31,"core":15,"socket":1},{"cpu":32,"core":0,"socket":0},{"cpu":33,"core":1,"socket":0},{"cpu":34,"core":2,"socket":0},{"cpu":35,"core":3,"socket":0},{"cpu":36,"core":4,"socket":0},{"cpu":37,"core":5,"socket":0},{"cpu":38,"core":6,"socket":0},{"cpu":39,"core":7,"socket":0},{"cpu":40,"core":8,"socket":0},{"cpu":41,"core":9,"socket":0},{"cpu":42,"core":10,"socket":0},{"cpu":43,"core":11,"socket":0},{"cpu":44,"core":12,"socket":0},{"cpu":45,"core":13,"socket":0},{"cpu":46,"core":14,"socket":0},{"cpu":47,"core":15,"socket":0},{"cpu":48,"core":0,"socket":1},{"cpu":49,"core":1,"socket":1},{"cpu":50,"core":2,"socket":1},{"cpu":51,"core":3,"socket":1},{"cpu":52,"core":4,"socket":1},{"cpu":53,"core":5,"socket":1},{"cpu":54,"core":6,"socket":1},{"cpu":55,"core":7,"socket":1},{"cpu":56,"core":8,"socket":1},{"cpu":57,"core":9,"socket":1},{"cpu":58,"core":10,"socket":1},{"cpu":59,"core":11,"socket":1},{"cpu":60,"core":12,"socket":1},{"cpu":61,"core":13,"socket":1},{"cpu":62,"core":14,"socket":1},{"cpu":63,"core":15,"socket":1}],"diskInfos":[{"device":"/dev/sdh1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata8","diskType":"unknown"},{"device":"/dev/sda1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata1","diskType":"unknown"},{"device":"/dev/sdi1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata9","diskType":"unknown"},{"device":"/dev/sdb1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata2","diskType":"unknown"},{"device":"/dev/sdj3","filesystemType":"ext4","size":52710469632,"mountPoint":"/","diskType":"unknown"},{"device":"/dev/sdd1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata4","diskType":"unknown"},{"device":"/dev/sdg1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata7","diskType":"unknown"},{"device":"/dev/sdm1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata12","diskType":"unknown"},{"device":"/dev/sdl1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata11","diskType":"unknown"},{"device":"/dev/sde1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata5","diskType":"unknown"},{"device":"/dev/dfa1","filesystemType":"ext4","size":3149634568192,"mountPoint":"/home/t4","diskType":"unknown","isGraphDisk":true},{"device":"/dev/sdc1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata3","diskType":"unknown"},{"device":"tmpfs","filesystemType":"tmpfs","size":134893674496,"mountPoint":"/dev/shm","diskType":"unknown"},{"device":"/dev/sdj5","filesystemType":"ext4","size":180134858752,"mountPoint":"/home","diskType":"unknown"},{"device":"/dev/sdf1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata6","diskType":"unknown"},{"device":"/dev/sdj2","filesystemType":"ext4","size":1023303680,"mountPoint":"/boot","diskType":"unknown"},{"device":"/dev/sdk1","filesystemType":"ext4","size":5953724899328,"mountPoint":"/data/sata10","diskType":"unknown"}]}`
)

func TestCPUPool_Initialize(t *testing.T) {

	nodeInfo, err := makeNodeInfo()
	if err != nil {
		t.Errorf(err.Error())
	}
	pool := NewCPUPool(nodeInfo)
	top := pool.Topology()
	if top.NumCPUs != 64 {
		t.Errorf("incorrect NumCPUs, expected 64 actual %d", top.NumCPUs)
	}
}

func TestGetNonExclusiveCPUSet(t *testing.T) {
	nodeInfo, err := makeNodeInfo()
	if err != nil {
		t.Errorf(err.Error())
	}
	pool := NewCPUPool(nodeInfo)
	ret := pool.GetNonExclusiveCPUSet().Size()
	if ret != 64 {
		t.Errorf("cpuset size should be 64, actual %d", ret)
	}
}

func TestGetAllocatedCPUShare(t *testing.T) {
	pod := makePod("1000m", "2")
	pod2 := makePod("2000m", "2")
	nodeInfo, err := makeNodeInfo(pod, pod2)
	if err != nil {
		t.Errorf(err.Error())
	}
	pool := NewCPUPool(nodeInfo)
	ret := pool.GetAllocatedCPUShare()
	if ret != 3000 {
		t.Errorf("CPUShare should be 3000, actual %d", ret)
	}
	nodeInfo, err = makeNodeInfo()
	pool = NewCPUPool(nodeInfo)

	overRatio := 1.0
	reti := int(float64(pool.GetAllocatedCPUShare()+int64(overRatio*float64(1000)-1)) / (overRatio * 1000))
	if reti != 0 {
		t.Errorf("CPUNums should be 0, actual %d", reti)
	}
	nonExclusivePoolSize := 64
	r := nonExclusivePoolSize - int(float64(pool.GetAllocatedCPUShare()+int64(overRatio*float64(1000)-1))/(overRatio*1000))
	glog.V(3).Infof("[DEBUG] %d", r)
}

func TestCPUShareOccupiedCPUs(t *testing.T) {
	pod := makePod("1000m", "2")
	pod2 := makePod("2000m", "2")
	nodeInfo, _ := makeNodeInfo(pod, pod2)
	pool := NewCPUPool(nodeInfo)

	ratio := pool.NodeOverRatio()
	if ratio != 1 {
		t.Errorf("over ratio should be 1, acutal: %f", ratio)
	}
	ret := pool.CPUShareOccupiedCPUs()
	if ret != 3 {
		t.Errorf("should be rounded up %d, acutal: %d", 3, ret)
	}

	/// With over ratio
	nodeInfo.Node().Labels = make(map[string]string, 0)
	nodeInfo.Node().Labels[sigmak8sapi.LabelCPUOverQuota] = "2.0"
	pool = NewCPUPool(nodeInfo)

	ratio = pool.NodeOverRatio()
	if ratio != 2 {
		t.Errorf("over ratio should be 1, acutal: %f", ratio)
	}
	ret = pool.CPUShareOccupiedCPUs()
	if ret != 2 {
		t.Errorf("should be rounded up %d, acutal: %d", 2, ret)
	}
}

func makeNodeInfo(pods ...*v1.Pod) (*schedulercache.NodeInfo, error) {
	nodeInfo := schedulercache.NewNodeInfo(pods...)
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				sigmak8sapi.AnnotationLocalInfo: LocalInfo,
			},
		},
	}
	err := nodeInfo.SetNode(node)
	return nodeInfo, err

}
