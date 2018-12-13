#!/bin/bash

KUBECONFIG=${KUBECONFIG:?"kubeconfig for deploying hollow-kubelet node, must required"}
MASTER_KUBECONFIG=${MASTER_KUBECONFIG:?"kubeconfig of sigma master, must required"}
KUBE_CONTEXT="${KUBE_CONTEXT:-}"

TMP_ROOT="$(dirname "${BASH_SOURCE}")/../.."
KUBE_ROOT=$(readlink -e ${TMP_ROOT} 2> /dev/null || perl -MCwd -e 'print Cwd::abs_path shift' ${TMP_ROOT})
KUBEMARK_DIRECTORY="${KUBE_ROOT}/test/kubemark"
RESOURCE_DIRECTORY="${KUBEMARK_DIRECTORY}/resources"

# Check whether kubectl/ginkgo installed
if which kubectl ; then
    KUBECTL=`which kubectl`
else
    cd ${KUBE_ROOT}
    make all WHAT="cmd/kubectl" GOFLAGS=-v
    KUBECTL="${KUBE_ROOT}/_output/bin/kubectl"
    cd -
fi

${KUBECTL} --context=${KUBE_CONTEXT} --kubeconfig=${KUBECONFIG} delete -f "${RESOURCE_DIRECTORY}/addons" --namespace="kubemark" &> /dev/null || true
${KUBECTL} --context=${KUBE_CONTEXT} --kubeconfig=${KUBECONFIG} delete -f "${RESOURCE_DIRECTORY}/hollow-node.yaml" --namespace="kubemark" &> /dev/null || true
${KUBECTL} --context=${KUBE_CONTEXT} --kubeconfig=${KUBECONFIG} delete -f "${RESOURCE_DIRECTORY}/kubemark-ns.json" &> /dev/null || true

#rm -rf "${RESOURCE_DIRECTORY}/addons" \
#	"${RESOURCE_DIRECTORY}/kubeconfig.kubemark" \
#	"${RESOURCE_DIRECTORY}/hollow-node.yaml" \
#	"${RESOURCE_DIRECTORY}/kubemark-master-env.sh"  &> /dev/null || true

# remove all the hollow nodes
nodes=$(kubectl get node --kubeconfig=${MASTER_KUBECONFIG} | grep hollow-node | awk '{print $1}')
for n in ${nodes}; do
    echo "delete hollow node: "$n
	${KUBECTL} --kubeconfig=${MASTER_KUBECONFIG} delete node $n
done