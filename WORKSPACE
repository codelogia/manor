workspace(name = "manor")

load("//build:versions.bzl", "versions")

load("//build/macros:repositories.bzl", "setup_repositories")
setup_repositories()

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace") 
bazel_skylib_workspace()

load("//build/macros:binaries.bzl", "setup_binaries")
setup_binaries()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()
go_register_toolchains(version = versions["go"])
gazelle_dependencies()

load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")
container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")
container_deps()

load("//:repositories.bzl", "go_repositories")

# gazelle:repository_macro repositories.bzl%go_repositories
go_repositories()
