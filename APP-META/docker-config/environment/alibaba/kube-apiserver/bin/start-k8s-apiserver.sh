#!/bin/bash

if [[ -z $CLUSTER_IP_RANGE ]]; then
    CLUSTER_IP_RANGE=192.168.0.0/16
fi

if [[ -z $CLUSTER_ETCD ]]; then
    echo "CLUSTER_ETCD is empty"
    exit 1
fi

certDir=/etc/kubernetes/pki
workDir=/home/admin/kube-apiserver

if [ $ETCD_USE_TLS == "true" ]; then
    etcdTlsOpts="--etcd-cafile=$certDir/etcd-ca.crt --etcd-certfile=$certDir/apiserver-etcd-client.crt --etcd-keyfile=$certDir/apiserver-etcd-client.key"
fi

if [[ -n $ETCD_SERVERS_OVERRIDES ]]; then
    etcdServersOverridesOpts="--etcd-servers-overrides=$ETCD_SERVERS_OVERRIDES"
fi

pidof kube-apiserver || {
$workDir/bin/kube-apiserver \
    --enable-admission-plugins=Initializers,NamespaceLifecycle,ServiceAccount,LimitRanger,DefaultStorageClass,DefaultTolerationSeconds,PodPreset,ResourceQuota,ValidatingAdmissionWebhook,MutatingAdmissionWebhook,AliPodLifeTimeHook,PodDeletionFlowControl,AliPodInjectionPreSchedule,NewAliPodInjectionPreSchedule,AliPodInjectionPostSchedule,ContainerState \
    --advertise-address=$(hostname -i) \
    --allow-privileged=true \
    --audit-policy-file=$workDir/cfg/audit.yaml --audit-log-path=$workDir/log/k8s-audit.log --audit-log-format=json --audit-log-maxsize=10240 --audit-log-maxbackup=1 \
    --authorization-mode=Node,RBAC \
    --bind-address=0.0.0.0 --secure-port=6443 \
    --client-ca-file=$certDir/ca.crt \
    --etcd-servers=$CLUSTER_ETCD --storage-backend=etcd3 \
    $etcdTlsOpts \
    $etcdServersOverridesOpts \
    --external-hostname=localhost \
    --feature-gates=AllAlpha=false,CSINodeInfo=true,CSIDriverRegistry=true,SelectorIndex=true \
    --insecure-bind-address=0.0.0.0 --insecure-port=8080 \
    --max-requests-inflight=3000 --max-mutating-requests-inflight=1000 \
    --request-timeout=300s \
    --requestheader-client-ca-file=$certDir/front-proxy-ca.crt \
    --requestheader-allowed-names=front-proxy-client \
    --requestheader-extra-headers-prefix=X-Remote-Extra- --requestheader-group-headers=X-Remote-Group --requestheader-username-headers=X-Remote-User \
    --proxy-client-cert-file=$certDir/front-proxy-client.crt --proxy-client-key-file=$certDir/front-proxy-client.key \
    --runtime-config=admissionregistration.k8s.io/v1alpha1,settings.k8s.io/v1alpha1=true \
    --service-account-key-file=$certDir/sa.pub \
    --service-cluster-ip-range=$CLUSTER_IP_RANGE \
    --tls-cert-file=$certDir/apiserver.crt --tls-private-key-file=$certDir/apiserver.key \
    --v=3 >> $workDir/log/k8s-apiserver.log 2>&1 &
}
