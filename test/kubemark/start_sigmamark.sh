#!/bin/bash

export KUBECONFIG=${KUBECONFIG:?"kubeconfig for deploying hollow-kubelet node, must required"}
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

source "${KUBE_ROOT}/cluster/kubemark/sigma/config-default.sh"


###############################
# Setup for master.
###############################

MASTER_KUBECONFIG=${MASTER_KUBECONFIG:?"kubeconfig of sigma master, must required"}
MASTER_IP=${MASTER_IP:?"sigma master IP, must required"}

${KUBECTL} --kubeconfig=${MASTER_KUBECONFIG} apply -f ${RESOURCE_DIRECTORY}/manifests/addons/kubemark-rbac-bindings



###############################
# Setup for hollow-nodes.
###############################

# Create a docker image for hollow-node and upload it to the appropriate docker registry.
function create-and-upload-hollow-node-image {
    echo "create and upload hollow node image ..."
}

function create-sigma-hollow-node-resources {
    echo "create sigma hollow nodes ..."
    ${KUBECTL} --context=${KUBE_CONTEXT} create -f "${RESOURCE_DIRECTORY}/kubemark-ns.json"
    ${KUBECTL} --context=${KUBE_CONTEXT} create configmap "node-configmap" --namespace="kubemark" \
        --from-literal=content.type="${TEST_CLUSTER_API_CONTENT_TYPE}" \
        --from-file=kernel.monitor="${RESOURCE_DIRECTORY}/kernel-monitor.json"
    ${KUBECTL} --context=${KUBE_CONTEXT} create secret generic "kubeconfig" --type=Opaque --namespace="kubemark" \
        --from-file=kubelet.kubeconfig="${MASTER_KUBECONFIG}" \
        --from-file=kubeproxy.kubeconfig="${MASTER_KUBECONFIG}" \
        --from-file=heapster.kubeconfig="${MASTER_KUBECONFIG}" \
        --from-file=cluster_autoscaler.kubeconfig="${MASTER_KUBECONFIG}" \
        --from-file=npd.kubeconfig="${MASTER_KUBECONFIG}"


    # Create addon pods.
    mkdir -p "${RESOURCE_DIRECTORY}/addons"

    # Heapster.
    sed "s/{{MASTER_IP}}/${MASTER_IP}/g" "${RESOURCE_DIRECTORY}/heapster_template-sigma.json" > "${RESOURCE_DIRECTORY}/addons/heapster.json"
    metrics_mem_per_node=4
    metrics_mem=$((200 + ${metrics_mem_per_node}*${NUM_NODES}))
    sed -i'' -e "s/{{METRICS_MEM}}/${metrics_mem}/g" "${RESOURCE_DIRECTORY}/addons/heapster.json"
    metrics_cpu_per_node_numerator=${NUM_NODES}
    metrics_cpu_per_node_denominator=2
    metrics_cpu=$((80 + metrics_cpu_per_node_numerator / metrics_cpu_per_node_denominator))
    sed -i'' -e "s/{{METRICS_CPU}}/${metrics_cpu}/g" "${RESOURCE_DIRECTORY}/addons/heapster.json"
    eventer_mem_per_node=500
    eventer_mem=$((200 * 1024 + ${eventer_mem_per_node}*${NUM_NODES}))
    sed -i'' -e "s/{{EVENTER_MEM}}/${eventer_mem}/g" "${RESOURCE_DIRECTORY}/addons/heapster.json"

    ## Cluster Autoscaler.
    #if [[ "${ENABLE_KUBEMARK_CLUSTER_AUTOSCALER}" == "true" ]]; then
    #  echo "Setting up Cluster Autoscaler"
    #  KUBEMARK_AUTOSCALER_MIG_NAME="${KUBEMARK_AUTOSCALER_MIG_NAME:-${NODE_INSTANCE_PREFIX}-group}"
    #  KUBEMARK_AUTOSCALER_MIN_NODES="${KUBEMARK_AUTOSCALER_MIN_NODES:-0}"
    #  KUBEMARK_AUTOSCALER_MAX_NODES="${KUBEMARK_AUTOSCALER_MAX_NODES:-10}"
    #  NUM_NODES=${KUBEMARK_AUTOSCALER_MAX_NODES}
    #  echo "Setting maximum cluster size to ${NUM_NODES}."
    #  KUBEMARK_MIG_CONFIG="autoscaling.k8s.io/nodegroup: ${KUBEMARK_AUTOSCALER_MIG_NAME}"
    #  sed "s/{{master_ip}}/${MASTER_IP}/g" "${RESOURCE_DIRECTORY}/cluster-autoscaler_template.json" > "${RESOURCE_DIRECTORY}/addons/cluster-autoscaler.json"
    #  sed -i'' -e "s/{{kubemark_autoscaler_mig_name}}/${KUBEMARK_AUTOSCALER_MIG_NAME}/g" "${RESOURCE_DIRECTORY}/addons/cluster-autoscaler.json"
    #  sed -i'' -e "s/{{kubemark_autoscaler_min_nodes}}/${KUBEMARK_AUTOSCALER_MIN_NODES}/g" "${RESOURCE_DIRECTORY}/addons/cluster-autoscaler.json"
    #  sed -i'' -e "s/{{kubemark_autoscaler_max_nodes}}/${KUBEMARK_AUTOSCALER_MAX_NODES}/g" "${RESOURCE_DIRECTORY}/addons/cluster-autoscaler.json"
    #fi

    toleration=""
    affinity="{}"
    if [ "$TOLERATION_KEY" -a "$TOLERATION_VALUE" ]; then
        toleration=",{\"key\":\"$TOLERATION_KEY\",\"operator\":\"Equal\",\"effect\":\"NoSchedule\",\"value\":\"$TOLERATION_VALUE\"}"
        affinity="{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"$TOLERATION_KEY\",\"operator\":\"In\",\"values\":[\"$TOLERATION_VALUE\"]}]}]}}}"
    fi
    sed -i'' -e "s/{{TOLERATION}}/${toleration}/g" "${RESOURCE_DIRECTORY}/addons/heapster.json"
    sed -i'' -e "s/{{AFFINITY}}/${affinity}/g" "${RESOURCE_DIRECTORY}/addons/heapster.json"


    ${KUBECTL} --context=${KUBE_CONTEXT} create -f "${RESOURCE_DIRECTORY}/addons" --namespace="kubemark"

    # Create the replication controller for hollow-nodes.
    # We allow to override the NUM_REPLICAS when running Cluster Autoscaler.
    NUM_REPLICAS=${NUM_REPLICAS:-${NUM_NODES}}
    sed "s/{{numreplicas}}/${NUM_REPLICAS}/g" "${RESOURCE_DIRECTORY}/hollow-node-sigma_template.yaml" > "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    proxy_cpu=20
    if [ "${NUM_NODES}" -gt 1000 ]; then
      proxy_cpu=50
    fi
    proxy_mem_per_node=50
    proxy_mem=$((100 * 1024 + ${proxy_mem_per_node}*${NUM_NODES}))
    sed -i'' -e "s/{{HOLLOW_PROXY_CPU}}/${proxy_cpu}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{HOLLOW_PROXY_MEM}}/${proxy_mem}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{master_ip}}/${MASTER_IP}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{hollow_kubelet_params}}/${HOLLOW_KUBELET_TEST_ARGS}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{hollow_proxy_params}}/${HOLLOW_PROXY_TEST_ARGS}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{kubelet_verbosity_level}}/--v=2/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{kubeproxy_verbosity_level}}/--v=2/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s'{{kubemark_mig_config}}'${KUBEMARK_MIG_CONFIG:-}'g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"

    toleration=""
    affinity=""
    if [ "$TOLERATION_KEY" -a "$TOLERATION_VALUE" ]; then
        toleration="- {key: $TOLERATION_KEY, operator: Equal, value: $TOLERATION_VALUE, effect: NoSchedule}"
        affinity="affinity: {nodeAffinity: {requiredDuringSchedulingIgnoredDuringExecution: {nodeSelectorTerms: [{matchExpressions: [{key: $TOLERATION_KEY, operator: In, values: [$TOLERATION_VALUE]}]}]}}}"
        echo $toleration
    fi
    sed -i'' -e "s/{{TOLERATION}}/${toleration}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"
    sed -i'' -e "s/{{AFFINITY}}/${affinity}/g" "${RESOURCE_DIRECTORY}/hollow-node.yaml"

    ${KUBECTL} --context=${KUBE_CONTEXT} create -f "${RESOURCE_DIRECTORY}/hollow-node.yaml" --namespace="kubemark"
}

# Wait until all hollow-nodes are running or there is a timeout.
function wait-for-hollow-nodes-to-run-or-timeout {
    echo -n "Waiting 30 min for all ${NUM_NODES} hollow-nodes to become Running"
    start=$(date +%s)
    nodes=$("${KUBECTL}" --kubeconfig="${MASTER_KUBECONFIG}" get node 2> /dev/null) || true
    ready=$(($(echo "${nodes}" | grep "hollow-node" | grep -v "NotReady" | wc -l) - 1))

    until [[ "${ready}" -ge "${NUM_REPLICAS}" ]]; do
    echo -n "."
    sleep 10
    now=$(date +%s)
    # Fail it if it already took more than 30 minutes.
    if [ $((now - start)) -gt 1800 ]; then
      echo ""
      echo "Timeout waiting for all hollow-nodes to become Running."
      # Try listing nodes again - if it fails it means that API server is not responding
      if "${KUBECTL}" --kubeconfig="${MASTER_KUBECONFIG}" get node &> /dev/null; then
        echo "Found only ${ready} ready hollow-nodes while waiting for ${NUM_NODES}."
      else
        echo "Got error while trying to list hollow-nodes. Probably API server is down."
      fi
      pods=$("${KUBECTL}" --context=${KUBE_CONTEXT} --kubeconfig="${KUBECONFIG}" get pods -l name=hollow-node --namespace=kubemark) || true
      running=$(($(echo "${pods}" | grep "Running" | wc -l)))
      echo "${running} hollow-nodes are reported as 'Running'"
      not_running=$(($(echo "${pods}" | grep -v "Running" | wc -l) - 1))
      echo "${not_running} hollow-nodes are reported as NOT 'Running'"
      echo $(echo "${pods}" | grep -v "Running")
      exit 1
    fi
    nodes=$("${KUBECTL}" --kubeconfig="${MASTER_KUBECONFIG}" get node 2> /dev/null) || true
    ready=$(($(echo "${nodes}" | grep "hollow-node" | grep -v "NotReady" | wc -l) - 0))
    done
    echo "Done!"
}

# Setup for hollow-nodes.
function start-hollow-nodes {
  echo "STARTING SETUP FOR HOLLOW-NODES"
  create-and-upload-hollow-node-image
  create-sigma-hollow-node-resources
  wait-for-hollow-nodes-to-run-or-timeout
}
start-hollow-nodes &

wait
echo ""
echo "Master IP: ${MASTER_IP}"
echo "Master KUBECONFIG: ${MASTER_KUBECONFIG}"