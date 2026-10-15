# Squash merges without losing breaking changes

[Português](squash-merges.md) · go-release-manager best-practices guide

This guide explains why a squash merge can hide a breaking change, how to prevent it, and what to do when it has already happened.

## Contents

- [The problem in one minute](#the-problem-in-one-minute)
- [What GitHub keeps in each case](#what-github-keeps-in-each-case)
- [Best practices](#best-practices)
- [Step-by-step setup](#step-by-step-setup)
- [How to write the PR](#how-to-write-the-pr)
- [If the break was already lost](#if-the-break-was-already-lost)
- [Limits](#limits)
- [FAQ](#faq)

## The problem in one minute

go-release-manager computes the version from the commit messages on `main`. A breaking change bumps the major when the message has a `!` in the header (`feat!: ...`) or a `BREAKING CHANGE: ...` footer in the last paragraph.

A squash merge turns the PR commits into **a single commit**, and GitHub builds its message. If the breaking-change notice lived only in the PR description, or in a commit in the middle, it may never reach `main`:

```text
PR "feat: drop --legacy"            → commit on main: "feat: drop --legacy (#42)"
  description: "BREAKING CHANGE: ..."   (the description was left out)
  computed version: minor             ← should be major
```

Nobody notices until the release ships with the wrong version, and users upgrade without knowing something broke.

## What GitHub keeps in each case

Setting in *Settings → General → Pull Requests → Allow squash merging → Default commit message*:

| Setting | PR with several commits | Single-commit PR |
|---|---|---|
| **Default message** (GitHub's default) | PR title + list of the commit messages | The commit message (the PR title is ignored) |
| **Pull request title** | Only the PR title | Only the PR title |
| **Pull request title and description** (recommended) | PR title + description | PR title + description |

Consequences:

- **The title always counts.** If the title has `!`, the break survives in every setting. It is the simplest and safest rule.
- **Only with "title and description"** does a `BREAKING CHANGE:` footer written in the description reach `main`.
- **With GitHub's default, a single-commit PR** uses the commit message: editing the PR title changes nothing.
- **Whoever clicks "Squash and merge" can edit the message** right there and delete the notice. Only the post-merge check catches that.

## Best practices

1. **Every break goes in the title with `!`:** `feat!: drop --legacy`, `fix(api)!: rename the status field`. This works in every setting.
2. **Explain the break in a footer at the end of the description:** `BREAKING CHANGE: --legacy was removed; use --mode=classic`. It becomes the migration note.
3. **Set squash to "Pull request title and description"**, so the description reaches `main`.
4. **Run `check-pr` on every PR and make it required:** it blocks the merge when a break is announced somewhere but missing from the title, and suggests the right title.
5. **Keep the post-merge safety net:** the post-merge workflow opens an issue if the break was lost anyway.
6. **Before releasing, check with `--explain`:** `go-release-manager create --dry-run --explain` shows which commits decided the increment.
7. **Write `BREAKING CHANGE:` exactly like that:** uppercase, singular, with a colon. `BREAKING CHANGES:` or `Breaking change:` do not count (`check-pr` warns).
8. **Prefer squash over an unconventional merge commit:** a merge commit such as `Merge pull request #42` does not follow Conventional Commits and is ignored by the calculation.

## Step-by-step setup

1. **GitHub:** *Settings → General → Pull Requests*, check **Allow squash merging** and pick **Default to pull request title and description**.
2. **PR check:** create `.github/workflows/pr-check.yml`:

   ```yaml
   name: PR check
   on:
     pull_request:
       types: [opened, edited, synchronize, reopened]
   permissions:
     contents: read
     pull-requests: read
   jobs:
     check-pr:
       runs-on: ubuntu-latest
       steps:
         - uses: fabiodrneles/go-release-manager/check-pr@v1.3.0
   ```

   The `edited` event re-runs the check when someone fixes the title.
3. **Make the check required:** *Settings → Branches* (or *Rules → Rulesets*), protection for `main`, *Require status checks to pass*, and pick **check-pr**.
4. **Safety net:** copy this repository's [post-merge-check.yml](../.github/workflows/post-merge-check.yml) and replace `uses: ./check-pr` with `version: source` by `uses: fabiodrneles/go-release-manager/check-pr@v1.3.0` (the `setup-go` step is no longer needed).
5. **PR template:** add to `.github/pull_request_template.md` the question "Is this a breaking change? If so, `!` in the title and `BREAKING CHANGE:` at the end of the description".

## How to write the PR

Good: the break is in the title and explained at the end of the description.

```text
Title:     feat!: drop --legacy

Description:
Remove the --legacy flag, deprecated since v1.4.

BREAKING CHANGE: --legacy was removed; use --mode=classic instead.
```

Bad: the title has no `!`. `check-pr` fails it and suggests `feat!: drop --legacy`.

```text
Title:     feat: drop --legacy

Description:
BREAKING CHANGE: --legacy was removed.
```

Bad: the footer is not in the last paragraph. `check-pr` warns, because the rule only reads the end of the message.

```text
BREAKING CHANGE: --legacy was removed.

Also updated the docs.
```

To check locally before opening the PR:

```text
go-release-manager check-pr --title "feat: drop --legacy" --body-file pr.md
```

## If the break was already lost

If the merge landed without the break (the safety net opens an issue), fix it **before the release**, in one of these ways:

- **Fix-up commit:** `git commit --allow-empty -m "feat!: drop --legacy" -m "BREAKING CHANGE: --legacy was removed"` on `main` (through a PR, if `main` is protected). The next calculation sees the break.
- **Forced version:** `go-release-manager create --release-as v2.0.0` (or the Action's `release-as` input) creates the tag with the right major.

If the release already shipped with the wrong version, publish the correct major version right away and note in the previous version's CHANGELOG that it contained a break.

## Limits

- **Declared breaks only:** the tool reads `!` and `BREAKING CHANGE:`. A break nobody announced needs a comparison of the public API between versions, which is not done yet.
- **Repository settings:** choosing the squash message and making the check required are steps for whoever administers the repository.
- **Direct pushes to `main`:** without a PR there is nothing to check beforehand; the normal calculation from the commits applies.

## FAQ

**Is a `!` in the title enough?** For the version, yes: the title always reaches `main` in a squash. The `BREAKING CHANGE:` footer explains the break and goes into the notes.

**I use merge commits, not squash. Do I need this?** The PR commits land as they are, so a break in a commit reaches `main`. `check-pr` is still useful to keep titles conventional.

**Does `check-pr` fail PRs without a break?** Only if the title is not a Conventional Commit or uses an unknown type. Warnings (misplaced footer, single-commit PR) do not fail.

**Does it work on macOS and Windows runners?** Yes. If the job fails in *Set up job* with "Failed to download archive", it is a GitHub instability downloading actions: re-run the job.
