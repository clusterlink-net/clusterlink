---
title: Developers
description: Setting up a development environment and contributing code
weight: 24
---

This guide provides a quick start for developers wishing to contribute to ClusterLink.

## Setting up a development environment

Here are the key steps for setting up your developer environment, making a change and testing it:

1. Install required tools (you can either do this manually or use the project's
 [devcontainer specification][])
    - [Go][] version 1.20 or higher.
    - [Git][] command line.
    - We recommend using a [local development environment][]  such as kind/kubectl for
      local development and integration testing.
    - Additional development packages, such as `goimports` and `golangci-lint`. See the full list in
      [post-create.sh][].
1. Clone our repository with `git clone git@github.com:clusterlink-net/clusterlink.git`.
1. Run `make test-prereqs` and install any missing required development tools.
1. Run `make build` to ensure the code builds as expected. This will pull in all needed
 dependencies.

## Making code changes

- If you are planning on contributing back to the project, please carefully read the
 [contribution guide][].
- We follow [GitHub's Standard Fork & Pull Request Workflow][].

All contributed code should pass precommit checks such as linting and other tests. These
 are run automatically as part of the CI process on every pull request. You may wish to
 run these locally, before initiating a PR:

```sh
$ make precommit
$ make unit-tests tests-e2e-k8s
$ go test ./...
```

Output of the end-to-end tests is saved to `/tmp/clusterlink-k8s-tests`. In case
 of failures, you can also (re-)run individual tests by name:

```sh
$ go test -v ./tests/e2e/k8s -testify.m TestConnectivity
```

### Tests in CICD

All pull requests undergo automated testing before being merged. This includes, for example,
 linting, end-to-end tests and DCO validation. Logs in CICD default to `info` lavel, and
 can be increased to `debug` by setting environment variable `DEBUG=1`. You can also enable
 debug logging from the UI when re-running a CICD job, by selecting "enable debug logging".

## Release management

ClusterLink releases, including container images and binaries, are built based
 on version tags in github. Applying a tag that's prefixed by `-v` will automatically
 trigger a new release through the github [release][] action.

To aid in auto-generation of changelog from commits, please kindly mark all PR's
 with one or more of the following labels:

- `ignore-for-release`: PR should not be included in the changelog report.
 This label should not be used together with any other label in this list.
- `documentation`: PR is a documentation update.
- `bugfix`: PR is fixing a bug in existing code.
- `enhancement`: PR provides new or extended functionality.
- `breaking-change`: PR introduces a breaking change in user facing aspects
 (e.g., API or CLI). This label may be used in addition to other labels (e.g.,
 `bugfix` or `enhancement`).

[devcontainer specification]: https://github.com/clusterlink-net/clusterlink/tree/main/.devcontainer/dev
[Go]: https://go.dev/doc/install
[Git]: https://git-scm.com/downloads
[local development environment]: https://kubernetes.io/docs/tasks/tools/
[post-create.sh]: https://github.com/clusterlink-net/clusterlink/blob/main/.devcontainer/dev/post-create.sh
[contribution guide]: https://github.com/clusterlink-net/clusterlink/blob/main/CONTRIBUTING.md
[GitHub's Standard Fork & Pull Request Workflow]: https://gist.github.com/Chaser324/ce0505fbed06b947d962
[release]: https://github.com/clusterlink-net/clusterlink/blob/main/.github/workflows/release.yml
