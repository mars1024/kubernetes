#!/bin/bash
cd  $GOPATH/src/k8s.io/kubernetes
git tag -d `git tag`
git tag v1.12.1
make kube-apiserver
cp _output/bin/kube-apiserver ./APP-META/docker-config/environment/alibaba/kube-apiserver/bin/kube-apiserver