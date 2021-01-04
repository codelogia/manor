# Copyright 2019-2020 Cloud Foundry Foundation, 2021 Codelogia
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

"""
The definitions and implementations of the Bazel rules for dealing with Helm.
"""

_helm_attr = attr.label(
    allow_single_file = True,
    cfg = "host",
    default = "@helm//:helm",
    executable = True,
)

def _package_impl(ctx):
    output_filename = "{}.tgz".format(ctx.attr.name)
    output_tgz = ctx.actions.declare_file(output_filename)
    outputs = [output_tgz]
    stamp_files = [ctx.info_file, ctx.version_file]
    args = ctx.actions.args()
    args.add("--helm-path", ctx.executable.helm.path)
    args.add("--package-dir", ctx.attr.package_dir)
    args.add("--output", output_tgz.path)
    args.add("--version", ctx.attr.version)
    args.add("--bazel-info-file", ctx.info_file.path)
    args.add("--bazel-version-file", ctx.version_file.path)
    ctx.actions.run(
        inputs = ctx.files.srcs + stamp_files,
        outputs = outputs,
        executable = ctx.executable.package_binary,
        arguments = [args],
        tools = [ctx.executable.helm],
    )
    return [DefaultInfo(files = depset(outputs))]

_package = rule(
    implementation = _package_impl,
    attrs = {
        "srcs": attr.label_list(
            mandatory = True,
        ),
        "package_dir": attr.string(
            mandatory = True,
        ),
        "version": attr.string(
            mandatory = True,
        ),
        "package_binary": attr.label(
            cfg = "host",
            default = "//build/rules/helm/package",
            executable = True,
        ),
        "helm": _helm_attr,
    },
)

def package(**kwargs):
    _package(
        package_dir = native.package_name(),
        **kwargs
    )

def _dependencies_impl(ctx):
    # Get the attribute absolute paths.
    helm = ctx.path(ctx.attr.helm)
    chart_yaml = ctx.path(ctx.attr.chart_yaml)
    requirements = ctx.path(ctx.attr.requirements)
    requirements_lock = ctx.path(ctx.attr.requirements_lock)

    # Symlink the required files into the cache.
    ctx.symlink(chart_yaml, "Chart.yaml")
    ctx.symlink(requirements, "requirements.yaml")
    ctx.symlink(requirements_lock, "requirements.lock")

    # Fetch the dependencies.
    ctx.execute([helm, "dep", "up"])

    # Create the workspace root BUILD.bazel.
    ctx.file("BUILD.bazel", 'package(default_visibility = ["//visibility:public"])\n')

    # Create the charts/BUILD.bazel exporting the fetched charts.
    charts_build = ctx.read(ctx.path(ctx.attr.charts_build_bazel))
    ctx.file("charts/BUILD.bazel", charts_build)

dependencies = repository_rule(
    _dependencies_impl,
    doc = """A repository rule for fetching and caching Helm dependencies.

    It creates a filegroup that exports all the files under the charts/ directory in the cache after
    `helm dep up` runs.
    """,
    attrs = {
        "chart_yaml": attr.label(
            allow_single_file = True,
            doc = "The Chart.yaml file containing the chart metadata.",
            mandatory = True,
        ),
        "helm": _helm_attr,
        "requirements": attr.label(
            allow_single_file = True,
            doc = "The requirements.yaml file containing the Helm dependencies.",
            mandatory = True,
        ),
        "requirements_lock": attr.label(
            allow_single_file = True,
            doc = "The requirements.lock file containing the locked Helm dependencies.",
            mandatory = True,
        ),
        "charts_build_bazel": attr.label(
            allow_single_file = True,
            default = "//build/rules/helm:dependencies_charts.BUILD.bazel",
            doc = "The BUILD.bazel file used to export the fetched sub-chart dependencies.",
        ),
    },
    local = True,
)
