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

"""The go.bzl module provides the go_vet rule for running Go within the project directory.
"""

go_vet_doc = """
The go_vet rule wraps the Go binary provided by the Go toolchain, which is provided by rules_go, and
runs the vet subcommand. It runs under the BUILD_WORKSPACE_DIRECTORY environment variable in the
original source tree appended with the package name, effectively running from the directory where
the BUILD.bazel file is located.
"""

def _go_vet_impl(ctx):
    return _impl(ctx, "vet", ["./..."])

go_vet = rule(
    _go_vet_impl,
    doc = go_vet_doc,
    executable = True,
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
)

go_fmt_doc = """
The go_fmt rule wraps the Go binary provided by the Go toolchain, which is provided by rules_go, and
runs the fmt subcommand. It runs under the BUILD_WORKSPACE_DIRECTORY environment variable in the
original source tree appended with the package name, effectively running from the directory where
the BUILD.bazel file is located.
"""

def _go_fmt_impl(ctx):
    return _impl(ctx, "fmt", ["./..."])

go_fmt = rule(
    _go_fmt_impl,
    doc = go_fmt_doc,
    executable = True,
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
)

go_mod_tidy_doc = """
The go_mod_tidy rule wraps the Go binary provided by the Go toolchain, which is provided by
rules_go, and runs the mod tidy subcommand. It runs under the BUILD_WORKSPACE_DIRECTORY environment
variable in the original source tree appended with the package name, effectively running from the
directory where the BUILD.bazel file is located.
"""

def _go_mod_tidy_impl(ctx):
    return _impl(ctx, "mod", ["tidy"])

go_mod_tidy = rule(
    _go_mod_tidy_impl,
    doc = go_mod_tidy_doc,
    executable = True,
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
)

def _impl(ctx, go_cmd, args = []):
    go = ctx.toolchains["@io_bazel_rules_go//go:toolchain"].sdk.go
    executable = ctx.actions.declare_file(ctx.attr.name)
    contents = [
        "pwd=$(pwd)",
        "cd \"${BUILD_WORKSPACE_DIRECTORY}/%s\"" % ctx.label.package,
        ("\"${pwd}/%s\" %s " % (go.path, go_cmd)) + " ".join(args),
    ]
    ctx.actions.write(executable, "\n".join(contents), is_executable = True)
    return [DefaultInfo(
        executable = executable,
        runfiles = ctx.runfiles(files = [go]),
    )]
