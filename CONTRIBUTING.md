# Contributing to External DNS STACKIT Webhook

Welcome and thank you for making it this far and considering contributing to external-dns-stackit-webhook.
We always appreciate any contributions by raising issues, improving the documentation, fixing bugs in the CLI or adding new features.

Before opening a PR please read through this document.
If you want to contribute but don't know how to start or have any questions feel free to reach out to us on [Github Discussions](https://github.com/stackitcloud/stackit-api-manager-cli/discussions). Answering any questions or discussions there is also a great way to contribute to the community.

## Process of making an addition

> Please keep in mind to open an issue whenever you plan to make an addition to features to discuss it before implementing it.

To contribute any code to this repository just do the following:

1. Make sure you have Go's latest version installed
2. Fork this repository
3. Run `make build` to make sure everything's setup correctly
4. Make your changes
   > Please follow the [seven rules of greate Git commit messages](https://chris.beams.io/posts/git-commit/#seven-rules)
   > and make sure to keep your commits clean and atomic.
   > Your PR won't be squashed before merging so the commits should tell a story.
   >
   > Optional: Sign-off on all Git commits by running `git commit -s`.
   > Take a look at the [Gihub Docs](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) for further information.
   >
   > Add documentation and tests for your addition if needed.
5. Run `make lint test` to ensure your code is ready to be merged
   > If any linting issues occur please fix them.
   > Using a nolint directive should only be used as a last resort.
6. Open a PR and make sure the CI pipelines succeed.
   > Your PR needs to have a semantic title, which can look like: `type(scope) Short Description`
   > All available `scopes` & `types` are defined in [semantic.yml](https://github.com/stackitcloud/stackit-api-manager-cli/blob/main/.github/semantic.yml)
   >
   > A example PR tile for adding a new feature for the CLI would looks like: `cli(feat) Add saving output to file`
7. Wait for one of the maintainers to review your code and react to the comments.
8. After approval merge the PR
9. Thank you for your contribution! :)
