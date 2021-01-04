# Copyright 2020-2021 Codelogia
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

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file")

def setup_repositories():
    http_archive(
        name = "bazel_skylib",
        urls = [
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
        ],
        sha256 = "1c531376ac7e5a180e0237938a2536de0c54d93f5c278634818e0efc952dd56c",
    )

    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "7904dbecbaffd068651916dce77ff3437679f9d20e1a7956bff43826e7645fcc",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
        ],
    )

    http_archive(
        name = "bazel_gazelle",
        sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        ],
    )

    http_archive(
        name = "io_bazel_rules_docker",
        sha256 = "1698624e878b0607052ae6131aa216d45ebb63871ec497f26c67455b34119c80",
        strip_prefix = "rules_docker-0.15.0",
        urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.15.0/rules_docker-v0.15.0.tar.gz"],
    )

    http_archive(
        name = "com_google_protobuf",
        sha256 = "1c744a6a1f2c901e68c5521bc275e22bdc66256eeb605c2781923365b7087e5f",
        strip_prefix = "protobuf-3.13.0",
        urls = ["https://github.com/protocolbuffers/protobuf/archive/v3.13.0.zip"],
    )

    http_archive(
        name = "rules_binaries",
        sha256 = "26212909741ffd6cb5b4f48cc35f8eec21497582b64b2ea51fe88c0048a1ec53",
        strip_prefix = "rules_binaries-0.1.0",
        urls = ["https://github.com/codelogia/rules_binaries/archive/v0.1.0.tar.gz"],
    )

    http_file(
        name = "dumb_init",
        sha256 = "cd7ab5513d20f4b985012d264fbdee60ccd2ea528b94b4b9308c6c6d26f8ba90",
        urls = ["https://github.com/Yelp/dumb-init/releases/download/v1.2.4/dumb-init_1.2.4_x86_64"],
        downloaded_file_path = "dumb-init",
    )

    http_archive(
        name = "pack",
        sha256 = "fd55dc3b566d38076e08129a56f7462e6f1d7eae0d16a0d97838b258a75fcfac",
        urls = ["https://github.com/buildpacks/pack/releases/download/v0.15.1/pack-v0.15.1-linux.tgz"],
        build_file = "//build:pack.BUILD",
    )

    http_archive(
        name = "docker",
        sha256 = "8790f3b94ee07ca69a9fdbd1310cbffc729af0a07e5bf9f34a79df1e13d2e50e",
        urls = ["https://download.docker.com/linux/static/stable/x86_64/docker-20.10.1.tgz"],
        build_file = "//build:docker.BUILD",
        strip_prefix = "docker",
    )
