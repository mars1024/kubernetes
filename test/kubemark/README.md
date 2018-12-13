# Build kubemark image

```bash
cd $GOPATH/src/k8s.io/kubernetes/
git pull
git checkout develop-tmp-xiaodai
make kubemark
cp _output/bin/kubemark cluster/images/kubemark
cd cluster/images/kubemark
export REGISTRY="reg.docker.alibaba-inc.com/k8s-test"
export IMAGE_TAG="latest"
make build
docker push $REGISTRY/kubemark:$IMAGE_TAG
```

# Set up sigma-mark environment

At first, **KUBECONFIG** should be specified.

```bash
export KUBECONFIG=<KUBECONFIG_for_sigmalet_deployment>
export MASTER_KUBECONFIG=<KUBECONFIG_of_testing_master>
export MASTER_IP=<IP_or_DNS_of_testing_master>

```

## start sigma-mark environment

```bash

cd $GOPATH/src/k8s.io/kubernetes/test/kubemark
export KUBEMARK_NUM_NODES=10 # how many hollow nodes will be created
bash start_sigmamark.sh
```

if run start_sigmamark.sh successfully, you can check hollow node by:

```bash
kubectl get node --kubeconfig=${MASTER_KUBECONFIG} | grep hollow
```

## stop sigma-mark environment

```bash
cd $GOPATH/src/k8s.io/kubernetes/test/kubemark
bash stop_sigmamark.sh
```

# Run scalability/performance e2e test

```bash
export  KUBECONFIG=${MASTER_KUBECONFIG}
cd $GOPATH/src/k8s.io/kubernetes/
git pull
# now develop-tmp-xiaodai can be used for testing, since some codes are changed in this branch
git checkout develop-tmp-xiaodai
git tag -d `git tag`
make WHAT=test/e2e/e2e.test
make ginkgo
export KUBERNETES_CONFORMANCE_TEST=yes
export KUBERNETES_PROVIDER="skeleton"
go run ./hack/e2e.go --get=false -- --check-version-skew=false --test --test_args="--e2e-verify-service-account=false --dump-logs-on-failure=false --ginkgo.focus=\[Feature:Performance\]"
```