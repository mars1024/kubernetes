#!/bin/bash

export KUBECONFIG=/Users/zizhuoy/MyCode/src/gitlab.alipay-inc.com/sigma/dev-cluster-certs/biz/sigma-eu95/admin.kubeconfig.yaml
export ALIPAY_ADAPTER=sigma-adapter.sigma-eu95.svc.alipay.net:8442
export ARMORY_USER=vulcan
export ARMORY_KEY=19dJYkWxOYjZJ2K6RgOpTw==
export CMDB_URL=http://vulcanboss.test.alipay.net
export CMDB_USER=local-test
export CMDB_TOKEN=test-token
export YOUR_WORKSPACE=`pwd`
export ALIPAY_CERT_PATH=${YOUR_WORKSPACE}/test/sigma/tlscert/sigma-bvt
export TEST_DATA_DIR=${YOUR_WORKSPACE}/test/sigma/testdata
export ENABLEOVERQUOTA=true