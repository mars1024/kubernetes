#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# A configuration for Kubemark cluster of sigma.

NUM_NODES=${KUBEMARK_NUM_NODES:-10}

EVENT_PD=${EVENT_PD:-false}

# Default Log level for all components in test clusters and variables to override it in specific components.
TEST_CLUSTER_LOG_LEVEL="${TEST_CLUSTER_LOG_LEVEL:---v=4}"
API_SERVER_TEST_LOG_LEVEL="${API_SERVER_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"
CONTROLLER_MANAGER_TEST_LOG_LEVEL="${CONTROLLER_MANAGER_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"
SCHEDULER_TEST_LOG_LEVEL="${SCHEDULER_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"

HOLLOW_KUBELET_TEST_LOG_LEVEL="${HOLLOW_KUBELET_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"
HOLLOW_PROXY_TEST_LOG_LEVEL="${HOLLOW_PROXY_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"

TEST_CLUSTER_DELETE_COLLECTION_WORKERS="${TEST_CLUSTER_DELETE_COLLECTION_WORKERS:---delete-collection-workers=16}"
TEST_CLUSTER_MAX_REQUESTS_INFLIGHT="${TEST_CLUSTER_MAX_REQUESTS_INFLIGHT:-}"
TEST_CLUSTER_RESYNC_PERIOD="${TEST_CLUSTER_RESYNC_PERIOD:-}"

# ContentType used by all components to communicate with apiserver.
TEST_CLUSTER_API_CONTENT_TYPE="${TEST_CLUSTER_API_CONTENT_TYPE:-}"

# Hollow-node components' test arguments.
HOLLOW_KUBELET_TEST_ARGS="${HOLLOW_KUBELET_TEST_ARGS:-} ${HOLLOW_KUBELET_TEST_LOG_LEVEL}"
HOLLOW_PROXY_TEST_ARGS="${HOLLOW_PROXY_TEST_ARGS:-} ${HOLLOW_PROXY_TEST_LOG_LEVEL}"

ALLOCATE_NODE_CIDRS=true

# Optional: Enable cluster autoscaler.
ENABLE_KUBEMARK_CLUSTER_AUTOSCALER="${ENABLE_KUBEMARK_CLUSTER_AUTOSCALER:-false}"
# When using Cluster Autoscaler, always start with one hollow-node replica.
# NUM_NODES should not be specified by the user. Instead we use
# NUM_NODES=KUBEMARK_AUTOSCALER_MAX_NODES. This gives other cluster components
# (e.g. kubemark master, Heapster) enough resources to handle maximum cluster size.
if [[ "${ENABLE_KUBEMARK_CLUSTER_AUTOSCALER}" == "true" ]]; then
  NUM_REPLICAS=1
  if [[ ! -z "$NUM_NODES" ]]; then
    echo "WARNING: Using Cluster Autoscaler, ignoring NUM_NODES parameter. Set KUBEMARK_AUTOSCALER_MAX_NODES to specify maximum size of the cluster."
  fi
fi
