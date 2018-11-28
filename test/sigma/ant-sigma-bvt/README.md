## 前置条件
* 需要有sigma的测试环境，获取kubeconfig
* 需要有sigma-adapter的地址
* 需要指定客户端证书的路径
* 3.1检查armory信息，需要传入armory-user/armory-key

### 运行
用例可以在开发同学的本地笔记本中运行，需要指定的环境变量如下
```bash
export KUBECONFIG=<从sigma测试环境获取的kubeconfig所在的路径>
export ALIPAY_CERT_PATH=<证书所在的目录路径>
export ALIPAY_ADAPTER=<adapter的地址>
export ARMORY_USER=<armory user信息>
export ARMORY_KEY=<armory key信息>
export YOUR_WORKSPACE=`pwd`
export TEST_DATA_DIR=${YOUR_WORKSPACE}/test/sigma/testdata
export ENABLEOVERQUOTA=true
```
按照自己的环境修改export.sh文件（都有默认值，需要更新的是`KUBECONFIG`和`APLIPATY_ADAPTER`两个ENV，如果`ENABLEOVERQUOTA`是false的需要更新下）
可以通过追加的方式
```
source ./test/sigma/ant-sigma-bvt/export.sh
```

编译测试：由于使用了社区的e2e framework代码，因此在执行sigma用例前，需要编译社区的e2e framework
```bash
cd ${GOPATH}/src/k8s.io/kubernetes
git tag -d `git tag`
make WHAT=test/e2e/e2e.test
```

执行测试：
如果只想执行某个用例，可以在__*-focus*__参数中设置用例名称
需要指定的环境变量在export.sh中指定环境变量的值，然后运行该脚本即可
```bash
cd ${GOPATH}/src/k8s.io/kubernetes
export KUBECONFIG=<kubeconfig_file_path>
export YOUR_WORKSPACE=`pwd`
ginkgo -v -focus="\[sigma-alipay-bvt\]\[adapter\]" test/sigma/ -- --test-data-dir=${YOUR_WORKSPACE}/test/sigma/testdata
```

#### 参数说明：
```bash
--kubeconfig kubeconfig配置路径
--test-data-dir 一些测试文件所在的目录
--sigma-pause-image 指定自定义的 pause 镜像名称（比如 `reg.docker.alibaba-inc.com/ali/os:7u2`）,可以避免默认 pause 镜像拉取错误的问题
--alipay-cert-path 指定adapter客户端证书的路径
--alipay-adapter-addr 指定adapter的访问地址
--armory-user 指定armory的user
--armory-key 指定armory的key信息
```

