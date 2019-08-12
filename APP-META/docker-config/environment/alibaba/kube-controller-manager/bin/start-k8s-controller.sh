#!/bin/bash

certDir=/etc/kubernetes/pki
kubeCfgDir=/etc/kubernetes/kubeconfig
workDir=/home/admin/kube-controller-manager

CONTROLLERS="*,-nodelifecycle"

if [[ ! -z $ENABLE_NODELIFECYCLE ]]; then
    CONTROLLERS="*"
fi

pidof kube-controller-manager || {
$workDir/bin/kube-controller-manager \
    --controllers="$CONTROLLERS" \
    --kubeconfig=$kubeCfgDir/controller-manager.kubeconfig \
    --kube-api-burst=300 --kube-api-qps=200 \
    --service-account-private-key-file=$certDir/sa.key \
    --use-service-account-credentials=true \
    --secure-port 0 \
    --v=4 >> $workDir/log/k8s-controller.log 2>&1 &
}
