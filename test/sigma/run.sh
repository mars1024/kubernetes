#!/bin/bash
set -e

export TEST_ROOT=${TEST_ROOT:?"MUST REQUIRED"}
export KUBECONFIG=${KUBECONFIG:-"/etc/kubernetes/kubelet.conf"}
export SIGMA_PAUSE_IMAGE=reg.docker.alibaba-inc.com/ali/os:7u2
export TESTER=${TESTER:-"jituan"}

cd $TEST_ROOT/test/sigma
export SWARM_IP=${SWARM_IP:?"MUST REQUIRED"}
export SWARM_PORT=${SWARM_PORT:-"8442"}
export SIGMA_SITE=${SIGMA_SITE:?"Please provide all sites of nodes in the test cluster, separated by ';'"}
export SIGMA_ETCD_ENDPOINTS=${SIGMA_ETCD_ENDPOINTS:?"Please provide sigma etcd endpoints"}
export TLS_DIR=$(pwd)/tlscert
export TEST_DATA_DIR=$(pwd)/testdata

cd $TEST_ROOT
tags=`git tag`
git tag -d $tags
make ginkgo
make WHAT=test/e2e/e2e.test
if which ginkgo ; then
    GINKGO=ginkgo
else
    if [[ $(uname -s |tr '[:upper:]' '[:lower:]') == "linux" ]]; then
        GINKGO="./_output/local/bin/linux/amd64/ginkgo"
    else
        echo "must have ginkgo in PATH"
        exit 1
    fi
fi

$GINKGO build  ./test/sigma
if (( $? != 0 )); then
	echo "build test failed"
	exit 1
fi

opt=$1
case $opt in
	build)
	echo "only build test binary"
	;;
	mix)
	$GINKGO -v -focus="\[sigma-2\.0\+3\.1\]" ./test/sigma/sigma.test
	;;
	all)
	$GINKGO -v -focus="\[sigma-.+\]" ./test/sigma/sigma.test
	;;
	mono)
	$GINKGO -v -focus="\[node-mono\]" ./test/sigma/sigma.test
	;;
	p0m0)
	$GINKGO -v -focus="\[p0m0\]" ./test/sigma/sigma.test
	;;
esac
