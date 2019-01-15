#!/bin/bash
cd  $GOPATH/src/k8s.io/kubernetes
rm -rf _output/
git tag -d `git tag`
git tag v1.12.1
make kube-scheduler
cp _output/bin/kube-scheduler ./APP-META/docker-config/environment/alibaba/kube-scheduler/bin/kube-scheduler
