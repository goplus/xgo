# Contributing Guidelines

Thank you for your interest in contributing to the XGo project! Your efforts
help us build and improve this powerful language.

Please be aware that participation in the XGo project requires adherence to our
established [code of conduct](https://github.com/goplus/xgo/blob/main/CODE_OF_CONDUCT.md). Your involvement signifies your agreement to comply
with the guidelines set forth within these terms. We appreciate your cooperation
in fostering a respectful and inclusive environment for all contributors.

To ensure a smooth contribution process, this guide outlines the steps and best
practices for contributing to XGo. It's assumed that you have a foundational
knowledge of XGo and are comfortable using Git and GitHub for version control
and collaboration.

## Before contributing code

The XGo project welcomes code contributions. However, to ensure coordination,
please discuss any significant changes beforehand. It's recommended to signal
your intent to contribute via the issue tracker, either by filing a new issue or
claiming an existing one.

The issue tracker is your go-to, whether you've got a contribution in mind or
are looking for inspiration. Issues are organized to streamline the workflow.

Except for very trivial changes, all contributions should relate to an existing
issue. Feel free to open one and discuss your plans. This process allows
validation of design, prevents duplicate efforts, ensures alignment with
language and tool goals, and confirms design validity before coding.

## Making a code contribution

Let's say you want to fix a typo in the `README.md` file of the
`https://github.com/goplus/xgo.git` repository, changing "Github" to "GitHub",
and you wish to merge this contribution into the `main` branch of the upstream
repository.

### Step 1: Fork the upstream repository

First, you need to fork the upstream repository to your own username (or an
organization you own) by following [this guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo). Then you can start making your
contributions.

Let's say you've forked `https://github.com/goplus/xgo.git` to
`https://github.com/YOUR_USERNAME/gop.git`.

### Step 2: Clone the forked repository

Open your terminal, clone the repository you just forked, and change into the
cloned directory:

```
git clone https://github.com/YOUR_USERNAME/gop.git
cd gop
```

This directory will serve as the working directory in the following steps.

### Step 3: Create a new branch

We recommend that every contribution should have its own branch, and this branch
should be deleted after the contribution is merged into the target branch of the
upstream repository.

First, you need to sync your fork's `main` branch to keep it up-to-date with the
upstream's `main` branch by following [this guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/syncing-a-fork).

Then, you can create and switch to a branch for your contribution:

```
git checkout -b typo main
```

### Step 4: Make the changes

This step is where you'll spend most of your effort when contributing. It
typically takes up the majority of the time needed to make a contribution.

However, in our case, we just want to correct a typo in the `README.md` file,
changing "Github" to "GitHub". So, go ahead and make that change.

### Step 5: Commit and push

After making your changes, you need to commit them:

```
git add README.md
git commit -s -m 'README.md: correct typo "Github" to "GitHub"'
```

The commit message here is very straightforward, but it's recommended to read
the [Commit Messages](#commit-messages) section for crafting better commit messages.

Once you've committed your changes, you also need to push your commit to your
fork on GitHub:

```
git push --set-upstream origin typo
```

### Step 6: Open a pull request

Head to the upstream repository and open a pull request by following
[this guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request). Your pull request should clearly describe the change you're
proposing and why it's beneficial.

Typically, the title and description of the pull request you open will come from
the first commit message in the branch of your fork. You can modify them if
necessary.

### Step 7: Engage with feedback

Be open to feedback from upstream repository maintainers. They might request
further changes or provide insights that could enhance your contribution.

After making any further changes, simply repeat [Step 5](#step-5-commit-and-push). However, when
committing, it's recommended to use the `--amend` option. This ensures that all
your changes are contained within a single commit, maintaining a linear commit
history. For example:

```
git add README.md
git commit --amend
```

This isn't a strict requirement, though. You can create as many commits as you
like.

### Step 8: Do post-merge cleanup

As mentioned in [Step 3](#step-3-create-a-new-branch), we recommend dedicating a branch to each contribution.
Once a contribution's pull request is [squash and merged](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#squash-and-merge-your-commits), the branch should be
deleted:

```
git checkout main
git branch -d typo
git push origin -d typo
```

## Commit messages

Commit messages in XGo adhere to specific conventions outlined in this section.

A well-crafted commit message looks like this:

```
docs: revise contribution guidelines for clarity and inclusivity

As our project grows, we want to ensure that our community understands
how to contribute effectively and feels welcomed to do so. This update
to our contribution guidelines aims to clarify the process and encourage
more developers to get involved.

Key changes include:
  - Simplified the language to make the guidelines more accessible to
    non-native English speakers and newcomers to open source.
  - Included a step-by-step guide on setting up the development
    environment, submitting a pull request, and what to expect during
    the review process.
  - Added a new section on community standards and code of conduct to
    foster a respectful and inclusive community atmosphere.
  - Provided clear examples of desirable contributions, such as bug
    fixes, feature proposals, and documentation improvements.

Fixes #123

Signed-off-by: John Doe <john@example.com>
```

### First line

The first line of a commit message typically serves as a concise one-line
summary, beginning with the primary component it impacts (e.g., a specific
package). This summary should ideally be brief, with many Git interfaces
favoring a limit of approximately 72 characters. While XGo is not strict on the
length limit, it's a good guideline to follow.

Think of the summary as finishing the sentence "This commit modifies XGo to
_____." As such, it does not begin with a capital letter, form a complete
sentence, or diverge from succinctly stating the commit's effect.

After the summary, ensure to separate the main content (if any) with a blank
line.

### Main content

The rest of the commit message should elaborate, offering context and a clear
explanation of the commit. Employ complete sentences and proper punctuation.
Refrain from using HTML, Markdown, or any form of markup language.

Include relevant information, like benchmark data, if the commit impacts
performance.

Adhere to a line width of approximately 72 characters to accommodate Git viewing
tools, except when longer lines are necessary (such as for ASCII art, tables, or
extended URLs).

### Referencing issues

The special notation `Fixes #123` links the commit directly to issue 123 in the
issue tracker. Upon merging this commit, the issue tracker will automatically
mark the issue as resolved.

For commits that partially address an issue, use `Updates #123`. This notation
will not close the issue upon the commit's integration.

To reference multiple issues within a single commit, list them by appending each
issue number with a comma and a space, such as `Fixes #123, #456`. This indicates
that the commit resolves both referenced issues.

For cross-repository references, use the full `user/repo#number` syntax, such as
`Fixes goplus/xgo#123`.

### Signed-off-by

For contributions to be accepted, we mandate that a `Signed-off-by` line, in
accordance with the [Developer Certificate of Origin](https://developercertificate.org/), be included at the
conclusion of each commit message.
