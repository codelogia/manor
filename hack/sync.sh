#!/bin/sh

# Copyright 2020 Codelogia
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

set -o errexit -o nounset -o pipefail

project_root=$(bazel info workspace)

cd "${project_root}"

set -o xtrace

bazel run //:go_mod_tidy

bazel run //:gazelle -- update-repos \
    -from_file=go.mod \
    -to_macro=repositories.bzl%go_repositories \
    -prune=true \
    -build_file_proto_mode=disable_global

bazel run --run_under="cd '${project_root}/operator' && " \
    @io_k8s_sigs_controller_tools//cmd/controller-gen -- object paths="./..."

bazel run --run_under="cd '${project_root}/operator' && " \
    @io_k8s_sigs_controller_tools//cmd/controller-gen -- \
        crd:trivialVersions=false \
        rbac:roleName=manager-role \
        webhook \
        paths="./..." \
        output:crd:artifacts:config=config/crd

bazel run //:gazelle

bazel run //:go_fmt
