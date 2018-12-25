#!/bin/bash
cd  $GOPATH/src/k8s.io/kubernetes
git tag -d `git tag`
git tag v1.12.1
make kube-controller-manager
cp _output/bin/kube-controller-manager ./APP-META/docker-config/environment/alibaba/kube-controller-manager/bin/kube-controller-manager