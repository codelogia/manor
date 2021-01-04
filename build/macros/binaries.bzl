# Copyright 2021 Codelogia
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

"""The repositories.bzl module provides the setup_repositories macro that wraps the shared rules
dependencies between the project workspaces.
"""

load("@rules_binaries//:def.bzl", "binary")

def setup_binaries():
    binary(
        name = "helm",
        config = {
            "sha256": {
                "darwin":  "c33b7ee72b0006f23b33f5032b531dd609fff7b08a4324f9ba07722a4f3fec9a",
                "linux":   "cacde7768420dd41111a4630e047c231afa01f67e49cc0c6429563e024da4b98",
                "windows": "76ff3f8c21c9af5b80abdd87ec07629ad88dbfe6206decc4d3024f26398554b9",
            },
            "url": {
                "darwin":  "https://get.helm.sh/helm-v{version}-darwin-amd64.tar.gz",
                "linux":   "https://get.helm.sh/helm-v{version}-linux-amd64.tar.gz",
                "windows": "https://get.helm.sh/helm-v{version}-windows-amd64.zip",
            },
            "version": "3.4.2",
            "strip_prefix": {
                "darwin":  "darwin-amd64",
                "linux":   "linux-amd64",
                "windows": "windows-amd64",
            },
        },
    )
