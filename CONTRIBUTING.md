# Contributing Guide

<!--  TODO Add TOC
* [New Contributor Guide](#contributing-guide)
  * [Ways to Contribute](#ways-to-contribute)
  * [Find an Issue](#find-an-issue)
  * [Ask for Help](#ask-for-help)
  * [Pull Request Lifecycle](#pull-request-lifecycle)
  * [Development Environment Setup](#development-environment-setup)
  * [Sign Your Commits](#sign-your-commits)
  * [Pull Request Checklist](#pull-request-checklist)
-->

## How to Help

Welcome! We are glad that you want to contribute to our project! 💖
 We welcome your contributions and participation! If you aren't sure what to expect, we've
 outlined the contribution process for ClusterLink below, so you feel more comfortable
 with how things will go.

If this is your first contribution to ClusterLink, there's a [tutorial](TODO missing link)
 that walks you through how to set up your developer environment, make a change and
 test it. Contributions are accepted via a GitHub Pull Request. The process is
 documented in detail [below](#the-life-of-pi-oops-the-life-of-a-pr)

## Code of Conduct

The ClusterLink community is governed by our [Code of Conduct](./CODE_OF_CONDUCT.md)
 and we expect all contributors to abide by it.

## Ways to Contribute

There are more ways to help than code contribution. If you have experience in marketing,
 content creation, technical writing, project management, community management, or other
 areas we might not have considered - please reach out!

We welcome many different types of contributions including:

* New features
* Builds, CI/CD
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

[Instructions](https://contribute.cncf.io/maintainers/github/templates/required/contributing/#come-to-meetings)

Absolutely everyone is welcome to come to any of our meetings. You never need an
invite to join us. In fact, we want you to join us, even if you don’t have
anything you feel like you want to contribute. Just being there is enough!

You can find out more about our meetings [here](TODO). You don’t have to turn on
your video. The first time you come, introducing yourself is more than enough.
Over time, we hope that you feel comfortable voicing your opinions, giving
feedback on others’ ideas, and even sharing your own ideas, and experiences.
-->

## Find an Issue

If you're new to ClusterLink, you are in the best position to give us feedback on areas of
our project that we can improve, including:

* Problems found during setting up a new developer environment
* Gaps in our guides or documentation
* Bugs in our automation scripts

If anything doesn't make sense, or doesn't work when you run it, please open a
bug report and let us know!

We have [good first issues](https://github.com/clusterlink-net/clusterlink/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) for new contributors
and [help wanted](https://github.com/clusterlink-net/clusterlink/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) issues suitable for any contributor.
 "Help wanted" issues are suitable for someone who isn't a core maintainer and are
 good to move onto after your first pull request.

<!-- 
If your project doesn’t always have issues labeled and ready to find, and you are willing 
to help find suitable issues, let new contributors know how to ask for something to work 
on.  
Sometimes there won’t be any issues with these labels. That’s ok! There is
likely still something for you to work on. If you want to contribute but you
don’t know where to start or can't find a suitable issue, you can ⚠️ **explain how people 
can ask for an issue to work on**.
-->

Once you see an issue that you'd like to work on, please post a comment saying
that you want to work on it. Something like "I want to work on this" is fine.

We have a [roadmap](TODO missing link) that will give you a good idea of the larger
 features that we are working on right now. That may help you decide what you would
 like to work on after you have tackled an issue or two. If you have a big idea for
 ClusterLink, you can propose it by creating an issue and marking it `enhancement`.

## Asking for Help

All contributors might get stuck sometimes. The best way to reach us with a question
 when contributing is to ask on:

* The original GitHub issue
* The [developer mailing list](TODO missing link)
* Our [Slack channel](TODO missing link)

<!-- t them to the proper communication channel but also provide links to relevant
documentation such as a contributing tutorial, troubleshooting guides, etc.

If you are a project that regularly gets contributors who are also new to git or the 
programming language, considering linking to where they can get help for non-project 
related questions, such as [ohshitgit](https://ohshitgit.com/), CNCF Slack channels, or 
community forum like the Gophers or Kubernetes Slack.
-->

## Development Environment Set-up

We have a [tutorial](TODO missing link) that walks you through setting up your developer
 environment, making a change and testing it.

Here are the key steps, if you run into trouble, the tutorial has more details:

1. Install Go version 1.20 or higher.
1. Clone our repository with `git clone git@github.com:clusterlink-net/clusterlink.git`  
1. Run `make build` to ensure the code builds fine. This will pull in all needed
 dependencies.
1. If you are planning on contributing back to the project, you'll need to fork this repository
  and clone your fork instead. If you want to synchronize your fork with the main `clsuterlink`
  repository, we recommend that you add an additional git remote with
  `git remote add upstream git@github.com:clusterlink-net/clusterlink.git`

## Sign Your Commits / DCO

Licensing is important to open source projects. It provides some assurances that
the software will continue to be available based under the terms that the
author(s) desired. We require that contributors sign off on commits submitted to
our project's repositories. The [Developer Certificate of Origin
(DCO)](https://probot.github.io/apps/dco/) is a way to certify that you wrote and
have the right to contribute the code you are submitting to the project.

You sign-off by adding the following to each of your commit messages. Your sign-off must
match the git user and email associated with the commit.

```txt
    This is my commit message

    Signed-off-by: Your Name <your.name@example.com>
```

Git has a `-s` command line option to do this automatically:

```sh
git commit -s -m 'This is my commit message'
```

If you forgot to do this and have not yet pushed your changes to the remote
repository, you can amend your commit with the sign-off by running

```sh
git commit --amend -s 
```

The PRs DCO check will fail if any of the included commits are not signed or not
 signed correctly. The DCO failure page has more information on fixing these after
 code has been pushed.

## Pull Request (PR) Lifecycle

### Scope of a PR

<!-- 
What kind of pull requests do you prefer: small scope, incremental value or feature complete?
-->
When you are ready to start on a unit of work, such as fixing a bug or implementing a
 feature, create a branch on your fork. Each branch (and ultimately each PR) should represent a
 logical unit of work. If you are doing two different tasks like fixing a bug and
 refactoring - please do these on different branches and in different PRs. While this
 is a bit of a hassle for you, it makes your changes much easier to review and faster
 to accept.
 Additionally, a change history with focused commits is easier to work with in the future,
 test failures are easier to localize and interaction between changes is easier to see.

### When to Open a PR

While it's OK to submit a PR directly for problems such as typos or other things where
the motivation/problem is unambiguous, most PRs should have an associated GitHub issue.

If there isn't an issue for your PR, please make an issue first and explain the problem
 or motivation for the change you are proposing. When the solution isn't straightforward,
 then also outline your proposed solution. Your PR will go smoother if the problem and
 solution are agreed upon before you spend time implementing it.

### Which Branch to Use

Unless the issue specifically mentions a branch, please create your feature branch from `main`.

For example:

```sh
# Make sure you have the most recent changes to clusterlink-net/clusterlink main
git checkout main
git fetch upstream main
git rebase upstream/main

# Create a branch based on main named MY_FEATURE_BRANCH
git checkout -b MY_FEATURE_BRANCH main
```

### The Life of Pi (oops! The Life of a PR)

1. Start on a [unit of work](#scope-of-a-pr) by updating from upstream and then creating
 a branch off of it. Please use a descriptive name (e.g., `fix-issue-17` or `lb-policy-support`). Please do **not** create dependent branches and always branch from
 an up-to-date `upstream/main`.

1. You can create a draft or WIP pull request at any time. You may also
 mark the PR as not ready by assigning the label `do-not-merge/wip` to it.
 Reviewers will ignore it mostly unless you mention someone and ask for help.
 Feel free to open one and use the pull request to see if the CI passes.
 Once you are ready for a review, remove the WIP or click "Ready for Review" and
 leave a comment that it's ready for review.
 Give the PR a descriptive title, that would be appropriate as a commit message once
 your work is merged. Include additional information, as appropriate, in the PR
 description paragraph to explain what you have done.
 If your PR fixes any issues, the description should refer to it (e.g., "fixes #NNN") or
 you can associate it using "linked issues". Either of these links your PR to the issue
 and automatically closes the issue when the PR is accepted.
 Do **not** use the PR comments to document the code as they will most likely never be
 seen after the PR is merged. Please **do** make your code self-explanatory. If there is
 information that is needed by a programmer reading the source code, you should put it in
 a code comment. This also applies to answering questions from reviewers: it is best done
 by clarifying the code or add documentation, rather than in the PR comment thread.

1. A reviewer will assign themselves to the pull request. If you don't see anyone
 assigned after 3 business days, you can leave a comment asking for a review, or ping in
 slack. Sometimes we have busy days, sick days, weekends and vacations, so a little
 patience is appreciated! 🙇‍♀️

1. The reviewer will leave feedback.

* `nits`: These are suggestions that you may decide to incorporate into your pull
 request or not without further comment.
* Requests for change in the PR contents. These require resolution before the PR is
 merged.
* It is okay to clarify if you are being told to make a change or if it is a suggestion.

If you agree with a code review comment and do what it suggests, don't respond in the
 GitHub code review system. Simply resolve the conversation. Respond if more discussion
 is needed, such as asking a follow-up question or explaining why you disagree with the
 suggestion. If the reviewer is asking a question, then usually the best way to answer
 it is by improving the code or documentation. Answering it only in the code review
 will not help future programmers after the PR is merged.

Respond to the feedback by making changes in your working copy, committing them, and
 periodically pushing them to GitHub when the tests pass locally. As soon as you
 receive feedback, you can start working on it. The reviewer should assign the code
 review back to you, but they might forget, so don't wait for that.

After you have made the changes (in new commits please!), leave a comment asking the
 the reviewer to take another look. If 3 business days go by with no review, it is okay
 to bump. With the exception of rebasing discussed below, please do **not** force a push
 with `git push -f` as it might cause loss of code review comments and context associated
 with previous commits.

When the PR is ready for merging, you may need to rebase your PR on top of the current codebase.
This can be done via the PR's GitHub Web page or with the CLI as either `git pull --rebase`
or `git fetch` and `git rebase` from `upstream/main`.
If there are any conflicts please fix them locally before committing and pushing to the PR branch.
If you're unsure on how to resolve a conflict, please ask the reviewer for guidance or help.
This `git push` is the only instance where you are expected to use `-f` or `--force`. Note that
it might "litter" your commit with unrelated commits done on `main` since the branch was created
and thus make reviewing more difficult. We therefore strive to [make PR small and reviews quick](#how-to-get-your-prs-reviewed-quickly),
to avoid too much divergence and merge conflict.

<!--
The repository owner can prevent incorrect pull request merges. In the repository settings, in the “Merge button” section, disable “Allow merge commits” and “Allow rebase merging”. You might also want to enable “Automatically delete head branches”.
-->

After you have addressed all the review feedback, explicitly request a re-review.
 Do not assume that person will know when you are done. There are many ways to request
 a re-review:

* Add the reviewer to the PR on Github - this works even if that person has reviewed the
 pull request before.
* Assign the pull request to that person, using the “Assignees” list.
* Write a comment in the conversation in the GitHub pull request.

1. When a pull request has been approved, the reviewer will squash and merge your
 commits. If you prefer to rebase your own commits, at any time leave a comment on the
 pull request to let them know that.

1. At this point your changes are available to be included in the next release of
 ClusterLink! After your first pull request is merged, you will be invited to the
 Contributors team (TODO create Github team) which you may choose to accept (or not).
 Joining the team lets you have issues in GitHub assigned to you.

### Follow-on PR

A follow-on PR is a pull request that finishes up suggestions from a previous PR.

When the core of your changes are good, and it won't hurt to do more of the changes
 later, our preference is to merge early, and keep working on it in a subsequent PR.
 This allows us to start testing out the changes early on, and more importantly helps us
 avoid pull requests to rely on other pull requests as other developers can immediately
 start building their work on top of yours.

### How to Get Your PRs Reviewed Quickly

🚧 If you aren't done yet, create a draft pull request or put WIP in the title so that
 reviewers wait for you to finish before commenting.

1️⃣ Limit your pull request to a single task. Don't tackle multiple unrelated things,
 especially refactoring. If you need large refactoring for your change, chat with a
 maintainer first, then do it in a separate PR first without any functionality changes.

🎳 Group related changes into separate commits to make it easier to review.

😅 Make requested changes in new commits. Please don't amend or rebase commits that we
 have already reviewed.

🚀 We encourage follow-on PRs and a reviewer may let you know in their comment if it is
 okay for their suggestion to be done in a follow-on PR. You can decide to make the
 change in the current PR immediately, or agree to tackle it in a reasonable amount of
 time in a subsequent pull request. If you can't get to it soon, please create an issue
 and link to it from the pull request comment so that we don't collectively forget.

### PR Checklist

When you submit your pull request, or you push new commits to it, our automated
 systems will run some checks on your new code. We require that your pull request
 passes these checks, but we also have more criteria than just that before we can
 accept and merge it. We recommend that you check the following things locally
 before you submit your code:

<!-- ⚠️ **Create a checklist that authors should use before submitting a pull request** 
Being done requires at least the following:

Testing: You have written tests for your feature or bug fix. All the tests pass, both 
locally and on continuous integration.
Documentation: You have documented each procedure that you added or modified, and you 
have updated the user manual if appropriate.
Completeness: Any change you make is because you discovered a problem. Look for other 
places that the problem might manifest, such as in code with a similar specification or 
implementation. Fix them all at once rather than leaving some to be discovered later.

It passes tests: run the following command to run all of the tests locally: make build test lint
 Impacted code has new or updated tests
 Documentation created/updated
 All tests succeed when run by the CI build on a pull request before it is merged
-->

### Reviewing a PR

This section is for maintainers who are reviewing and merging a pull request. While
 it is currently incomplete, it does contain a few tips.

* If a submitted PR is not likely to be accepted (e.g., complexity, quality, or scope), please
 let the submitter know so explaining the reasoning (and perhaps what needs to be done in order
 to get it accepted in different form) and promptly close the PR. There is no point in wasting
 anyone's time.
* Clearly communicate your availability and expected review schedule, when possible.
* Clearly indicate to the submitter which code change comments are suggestions and which are
 required.
* Whenever possible, use `Start a review` instead of adding individual comments. This minimizes review
 emails sent and lets the submitter see the complete feedback at once instead of in piecemeal fashion.
* Mark files as viewed to keep track of your progress. GitHub can use that to only show modified
 files on subsequent reviews.
* Use filtering (e.g., commits since last review) to better focus on the changes made. The PR's
 `Files` tab will always contain the PR in its entirety.
