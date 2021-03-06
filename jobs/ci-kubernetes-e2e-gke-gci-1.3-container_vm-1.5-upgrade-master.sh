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
export CLOUDSDK_API_ENDPOINT_OVERRIDES_CONTAINER="https://test-container.sandbox.googleapis.com/"
export CLOUDSDK_BUCKET="gs://cloud-sdk-testing/ci/staging"
export E2E_MIN_STARTUP_PODS="8"
export FAIL_ON_GCP_RESOURCE_LEAK="true"
export KUBERNETES_PROVIDER="gke"

### project-env
# expected empty

### job-env
readonly version_old='1.3'
readonly version_new='1.5'
readonly version_infix='g1-3-c1-5'
readonly image_old='gci'
readonly image_new='container_vm'


export E2E_OPT="--check_version_skew=false"
export E2E_UPGRADE_TEST="true"
export GINKGO_UPGRADE_TEST_ARGS="--ginkgo.focus=\[Feature:MasterUpgrade\] --upgrade-target=ci/latest-${version_new} --upgrade-image=${image_new}"
export JENKINS_PUBLISHED_SKEW_VERSION="ci/latest-${version_new}"
export JENKINS_PUBLISHED_VERSION="ci/latest-${version_old}"
[ "${version_new}" = "latest" ] && JENKINS_PUBLISHED_SKEW_VERSION="ci/latest"
[ "${version_old}" = "latest" ] && JENKINS_PUBLISHED_VERSION="ci/latest"
export KUBE_GKE_IMAGE_TYPE="${image_old}"
export PROJECT="gke-up-g1-3-c1-5-up-mas"
export ZONE="us-central1-c"

### version-env
# 1.3 doesn't support IAM, so we should use cert auth.
export CLOUDSDK_CONTAINER_USE_CLIENT_CERTIFICATE="true"
export ZONE="us-central1-a"

### post-env

# Assume we're upping, testing, and downing a cluster
export E2E_UP="${E2E_UP:-true}"
export E2E_TEST="${E2E_TEST:-true}"
export E2E_DOWN="${E2E_DOWN:-true}"

export E2E_NAME='bootstrap-e2e'

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
