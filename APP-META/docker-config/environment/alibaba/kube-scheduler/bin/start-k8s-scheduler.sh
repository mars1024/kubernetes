#!/bin/bash

workDir=/home/admin/kube-scheduler
kubeCfgDir=/etc/kubernetes/kubeconfig

POLICY_CONFIG=""
cat $kubeCfgDir/scheduler.kubeconfig | grep server | grep "et2.api3"
if [ $? -eq 0 ]; then
    POLICY_CONFIG="--policy-config-file=$workDir/cfg/scheduler-policy-config.json"
fi

pidof kube-scheduler || {
$workDir/bin/kube-scheduler \
    $POLICY_CONFIG \
    --kubeconfig $kubeCfgDir/scheduler.kubeconfig \
    --v=4 >> $workDir/log/k8s-scheduler.log 2>&1 &
}
