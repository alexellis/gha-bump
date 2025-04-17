Bump GitHub Actions Workflow versions
=============================================

Why is this tool needed?

Maintaining GitHub Actions across many different repositories can make for a tedious and repetitive task. Dependabot and other tooling do not always work as expected, or send PRs when new versions of workflows are published i.e. the 4-5 required by any project using Docker.

It's an unnecessary burden to memorise which version of docker/login-action is the latest, instead, use gha-bump. It'll perform a no-op if you have nothing to change, otherwise you'll see the changes in your staged files. You could even use this as a pre-commit hook.

Usage:

```bash
go get github.com/alexellis/gha-bump

# Upgrade and write changes, do it quietly
gha-bump --write ./.github/workflows/build.yaml

# Print changes, do not write
gha-bump --verbose --write false \
  ./.github/workflows/build.yaml
```

Before/after:

```diff
  - name: Checkout code
-   uses: actions/checkout@v2
+   uses: actions/checkout@v4
```

Caveats:

* Does not modify the `master` tag if used for an action - `actions/checkout@master`
* Does not work with actions which have been pinned with a SHA - `actions/checkout@sha1234567890`
* Does not work when a version does not have a `v` prefix - `alexellis/upload-assets@0.10.0`

## License

MIT

No warranty of any kind is included, use at your own risk.
