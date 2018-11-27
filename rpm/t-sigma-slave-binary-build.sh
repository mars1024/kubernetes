#!/bin/bash

# make code compile dir
ROOT_PATH="k8s.io/kubernetes/"
mkdir -p $1/../src/$ROOT_PATH

# delete git and init git to v1.10
cd $1  && git branch  && git status && git tag -d `git tag` &&  git tag v1.10 &&  git tag

# sync file
rsync -avz $1/*  $1/../src/$ROOT_PATH  --exclude=.svn --exclude=_output/

# init go env
export GOPATH=$1/../
export GOROOT=/usr/local/golang
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOROOT/bin:/Users/Tinker/go/bin:$GOBIN

# copy .git file and delete change
cd $1/../src/$ROOT_PATH/
cp -r $1/.git .
git status
git rev-parse --short  HEAD
git reset --hard HEAD
git clean -xdf
git rev-parse --short  HEAD
git status
git tag -d `git tag` &&  git tag v1.10 &&  git tag

# compile binary
bin="sigma-slave"
pwd && make WHAT=cmd/kubelet

# output binary  version
_output/bin/kubelet --version

# copy binary to packaging folder
cp _output/bin/kubelet  $1/rpm/${bin}

cd $1/rpm
rpm_create $2.spec -v $3 -r $4 -p /home/a/