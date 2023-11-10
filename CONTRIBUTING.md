# Contributing Guide

Welcome! We gladly accept contributions and encourage you to get involved in making
 ClusterLink the best it can be! 💖

## Code of Conduct

The ClusterLink community is governed by our [Code of Conduct](./CODE_OF_CONDUCT.md)
 and we expect all contributors to abide by it.

## Ways to Contribute

We welcome many different types of contributions including:

* New features
* Builds, CI/CD enhancements
* Bug fixes
* Documentation
* Issue Triage
* Answering questions on Slack/Mailing List
* Web design
* Communications / Social Media / Blog Posts
* Release management

<!--
Not everything happens through a GitHub pull request. Please come to our
[meetings](TODO) or [contact us](TODO) and let's discuss how we can work
together. -->

<!--
### Come to Meetings

Absolutely everyone is welcome to come to any of our meetings. You never need an
invite to join us. In fact, we want you to join us, even if you don’t have
anything you feel like you want to contribute. Just being there is enough!

You can find out more about our meetings [here](TODO). You don’t have to turn on
your video. The first time you come, introducing yourself is more than enough.
Over time, we hope that you feel comfortable voicing your opinions, giving
feedback on others’ ideas, and even sharing your own ideas, and experiences.
-->

## Pull Request Workflow

We follow [GitHub's Standard Fork & Pull Request Workflow](https://gist.github.com/Chaser324/ce0505fbed06b947d962)

## Bug Reports

First, please [search the ClusterLink repository](https://github.com/clusterlink-net/clusterlink/issues)
 with different keywords to ensure your bug is not already reported.

If not, [open an issue](https://github.com/clusterlink-net/clusterlink/issues/new), providing as
 much details as possible so we can understand and reproduce the problematic behavior. It is easiest
 to pinpoint the root cause when you write clear, concise instructions to reproduce the behavior.
 The more detailed and specific you are, the faster we will be able to help you. Check out [How to
 Report Bugs Effectively](https://www.chiark.greenend.org.uk/~sgtatham/bugs.html).

Please be kind. :smile: Remember that ClusterLink is work in progress and comes at no cost to you.

## Minor Improvements and New Tests

Submit [pull requests](https://github.com/clusterlink-net/clusterlink/pulls) at any time. Make
 sure to write tests to assert your change is working properly and is thoroughly covered.

## New Features

As with bug reports, please [search](https://github.com/clusterlink-net/clusterlink/issues) with
 a variety of keywords to ensure your suggestion/proposal is new. Please also check for existing pull
 requests to see if someone is already working on this. We want to avoid duplication of effort.

If the proposal is new and no one has opened pull request yet, you may open either an issue or a
 pull request for discussion and feedback. If you are going to spend significant time implementing
 code for a pull request, best to open an issue first and get feedback before investing time and effort.

If possible, make a pull request as small as possible, or submit multiple pull request to complete a
 feature. Smaller means: easier to understand and review. This in turn means things can be merged
 faster.

## New Contributors

If you're new to ClusterLink, you are in the best position to give us feedback on areas of
 our project that we can improve, including:

* Problems found during setting up a new developer environment
* Gaps in our guides or documentation
* Bugs in our automation scripts

If something doesn't make sense, or doesn't work when you run it, please open a
 bug report and let us know!

We have [good first issues](https://github.com/clusterlink-net/clusterlink/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) for new contributors
 and [help wanted](https://github.com/clusterlink-net/clusterlink/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
 issues suitable for any contributor.
 "Help wanted" issues are suitable for someone who isn't a core maintainer and are good to move onto
 after your first pull request.

<!-- 
Sometimes there won’t be any issues with these labels. That’s ok! There is
likely still something for you to work on. If you want to contribute but you
don’t know where to start or can't find a suitable issue, you can ⚠️ **explain how people can ask for an issue to work on**.
-->

We have a [roadmap](./README.md#roadmap) that will give you a good idea of the larger
 features that we are working on right now. That may help you decide what you would
 like to work on after you have tackled an issue or two. If you have a big idea for
 ClusterLink, you can propose it by creating an issue and marking it `enhancement`.

## Ask for Help

All contributors might get stuck sometimes. The best way to reach us with a question
 when contributing is to ask on:

* The original GitHub issue
* The [developer mailing list](https://groups.google.com/g/clusterlink-dev/)
<!-- * Our [Slack channel](TODO missing link) -->

## Developer Certificate of Origin

Licensing is important to open source projects. It provides some assurances that
 the software will continue to be available based under the terms that the
 author(s) desired. As required by the CNCF's [charter](https://github.com/cncf/foundation/blob/master/charter.md#11-ip-policy),
 all new code contributions must be accompanied by a
 [Developer Certificate of Origin (DCO)](https://developercertificate.org/). ClusterLink uses
 the [DCO App](https://github.com/apps/dco) to enforce the DCO on pull requests.

You may use git option `-s` to append automatically to the `Sign-off-by` line to your commit messages:

```sh
git commit -s
```

Your sign-off must match the git user and email associated with the commit.

<!-- 
Developer workflow (e.g., environment set up, creating PRs, commit messages, etc)
are covered in a separate document
-->
