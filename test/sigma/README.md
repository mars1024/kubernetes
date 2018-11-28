## 前置条件
* 需要有sigma的测试环境，获取kubeconfig
* 测试环境的搭建可以参考[minisigma](https://yuque.antfin-inc.com/sigma.pouch/sigma3.x/hnt4ak)，或者找测试同学获取

### 运行
用例可以在开发同学的本地笔记本中运行，只需要指定KUBECONFIG环境变量即可
```bash
export KUBECONFIG=<从sigma测试环境获取的kubeconfig所在的路径>
```

编译测试：由于使用了社区的e2e framework代码，因此在执行sigma用例前，需要编译社区的e2e framework
```bash
cd ${GOPATH}/src/k8s.io/kubernetes
git tag -d `git tag`
make WHAT=test/e2e/e2e.test
```

执行测试：
如果只想执行某个用例，可以在__*-focus*__参数中设置用例名称
```bash
cd ${GOPATH}/src/k8s.io/kubernetes
export KUBECONFIG=<kubeconfig_file_path>
export YOUR_WORKSPACE=`pwd`
ginkgo -v -focus="\[sigma-kubelet\]" test/sigma/ -- --test-data-dir=${YOUR_WORKSPACE}/test/sigma/testdata
```

#### 参数说明：
```bash
--kubeconfig kubeconfig配置路径
--test-data-dir 一些测试文件所在的目录
--sigma-pause-image 指定自定义的 pause 镜像名称（比如 `reg.docker.alibaba-inc.com/ali/os:7u2`）,可以避免默认 pause 镜像拉取错误的问题
```

#### 运行所有sigma e2e的用例
```bash
cd ${GOPATH}/src/k8s.io/kubernetes/test/sigma
sh -x run.sh all
```

#### 运行sigma 2.0+3.1的用例
```bash
cd ${GOPATH}/src/k8s.io/kubernetes/test/sigma
sh -x run.sh mix
```

