git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.12.0",
)

git_repository(
    name = "bazel_gazelle",
    remote = "https://github.com/bazelbuild/bazel-gazelle.git",
    tag = "0.12.0",
)

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_rules_dependencies",
    "go_register_toolchains",
)

go_rules_dependencies()

go_register_toolchains()

load(
    "@bazel_gazelle//:deps.bzl",
    "gazelle_dependencies",
    "go_repository",
)

gazelle_dependencies()

go_repository(
    name = "com_github_aws_aws_sdk_go",
    commit = "v1.16.4",
    importpath = "github.com/aws/aws-sdk-go",
)

go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    tag = "v1.2.2",
)

go_repository(
    name = "com_github_go_ini_ini",
    commit = "32cf4f7e9c77f151e18b53067b55549b4d1d411d",
    importpath = "github.com/go-ini/ini",
)

go_repository(
    name = "com_github_pmezard_go_difflib",
    commit = "792786c7400a136282c1664665ae0a8db921c6c2",
    importpath = "github.com/pmezard/go-difflib",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    commit = "782f4967f2dc4564575ca782fe2d04090b5faca8",
    importpath = "github.com/davecgh/go-spew",
)

go_repository(
    name = "com_github_jmespath_go_jmespath",
    commit = "bd40a432e4c76585ef6b72d3fd96fb9b6dc7b68d",
    importpath = "github.com/jmespath/go-jmespath",
)
