#!/bin/bash
# Copyright 2016 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

readonly testinfra="$(dirname "${0}")/.."

### provider-env
export KUBERNETES_PROVIDER="gce"
export E2E_MIN_STARTUP_PODS="1"
export FAIL_ON_GCP_RESOURCE_LEAK="false"
export CLOUDSDK_CORE_PRINT_UNHANDLED_TRACEBACKS="1"

### project-env
# expected empty

### job-env
export E2E_NAME="kubemark-2000"
export PROJECT="kubernetes-scale"
export E2E_TEST="false"
export USE_KUBEMARK="true"
export KUBEMARK_TESTS="\[Feature:Performance\]"
export KUBEMARK_TEST_ARGS="--gather-resource-usage=true --garbage-collector-enabled=true --kube-api-content-type=application/vnd.kubernetes.protobuf"
# Increase throughput in Kubemark master components.
export KUBEMARK_MASTER_COMPONENTS_QPS_LIMITS="--kube-api-qps=100 --kube-api-burst=100"
# Increase limit for inflight requests in apiserver.
export TEST_CLUSTER_MAX_REQUESTS_INFLIGHT="--max-requests-inflight=1500"
# Increase throughput in Load test.
export LOAD_TEST_THROUGHPUT=25
# Override defaults to be independent from GCE defaults and set kubemark parameters
# We need 11 so that we won't hit max-pods limit (set to 100). TODO: do it in a nicer way.
export NUM_NODES="60"
export MASTER_SIZE="n1-standard-4"
# Note: can fit about ~10 hollow nodes per core so NUM_NODES x
# cores_per_node should be set accordingly.
export NODE_SIZE="n1-standard-8"
export KUBEMARK_MASTER_SIZE="n1-standard-32"
# Increase disk size to check if that helps for etcd latency.
export MASTER_DISK_SIZE="100GB"
export KUBEMARK_NUM_NODES="5000"
export KUBE_GCE_ZONE="us-central1-b"
# =========================================
# Configuration we are targetting in 1.6
export TEST_CLUSTER_STORAGE_CONTENT_TYPE="--storage-media-type=application/vnd.kubernetes.protobuf"
# The kubemark scripts build a Docker image
export JENKINS_ENABLE_DOCKER_IN_DOCKER="y"
export KUBE_NODE_OS_DISTRIBUTION="gci"

# TODO: revert after running experiments.
export EVENT_PD="true"
# TODO remove after #19188 is fixed
export CUSTOM_ADMISSION_PLUGINS="NamespaceLifecycle,LimitRanger,ResourceQuota"
# TODO: Reduce this once we have log rotation in Kubemark.
export KUBEMARK_MASTER_ROOT_DISK_SIZE="100GB"

### post-env

# Assume we're upping, testing, and downing a cluster
export E2E_UP="${E2E_UP:-true}"
export E2E_TEST="${E2E_TEST:-true}"
export E2E_DOWN="${E2E_DOWN:-true}"

# Skip gcloud update checking
export CLOUDSDK_COMPONENT_MANAGER_DISABLE_UPDATE_CHECK=true
# Use default component update behavior
export CLOUDSDK_EXPERIMENTAL_FAST_COMPONENT_UPDATE=false

# AWS variables
export KUBE_AWS_INSTANCE_PREFIX="${E2E_NAME}"

# GCE variables
export INSTANCE_PREFIX="${E2E_NAME}"
export KUBE_GCE_NETWORK="${E2E_NAME}"
export KUBE_GCE_INSTANCE_PREFIX="${E2E_NAME}"

# GKE variables
export CLUSTER_NAME="${E2E_NAME}"
export KUBE_GKE_NETWORK="${E2E_NAME}"

# Get golang into our PATH so we can run e2e.go
export PATH="${PATH}:/usr/local/go/bin"

### Runner
readonly runner="${testinfra}/jenkins/dockerized-e2e-runner.sh"
export DOCKER_TIMEOUT="620m"
export KUBEKINS_TIMEOUT="600m"
"${runner}"
