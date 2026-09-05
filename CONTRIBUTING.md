# Contributing to BlueNote

Thank you for helping improve BlueNote. This project is an early-stage fork of [WriteFreely](https://github.com/writefreely/writefreely), so contributions should keep the fork maintainable and make intentional differences from upstream easy to identify.

Use [BlueNote issues](https://github.com/blue0a6m5c/bluenote/issues) for reproducible bugs and feature proposals. Before reporting a security issue, follow [SECURITY.md](SECURITY.md) and do not disclose vulnerability details in a public issue.

## Before starting

- Search existing issues and pull requests for related work.
- For a substantial feature or behavior change, open an issue first so its scope and upstream compatibility can be discussed.
- Check whether the behavior also occurs in upstream WriteFreely v0.17.2 when practical. Include that result in the issue or pull request.
- Keep each pull request focused on one complete change.

BlueNote does not currently require a contributor license agreement. Contributions submitted to this repository are made under the repository's [GNU AGPLv3 license](LICENSE). Contributions sent to upstream WriteFreely are governed by [WriteFreely's own contribution process](https://github.com/writefreely/writefreely/blob/develop/CONTRIBUTING.md), including any upstream CLA requirement.

## Branches and commits

Create work branches from `bluenote`, and open BlueNote pull requests with `bluenote` as the base branch. Use a short prefix that describes the change:

- `feat/` for features
- `fix/` for bug fixes
- `docs/` for documentation
- `test/` for test-only changes
- `chore/` for maintenance

Branches used to integrate upstream should use the `sync/` prefix. Do not mix upstream synchronization with unrelated BlueNote changes.

Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages, such as `feat: add an optional publishing mode`, `fix: stop an empty collection page`, or `docs: clarify local setup`.

## Code and documentation

- Follow the style already used in the surrounding code.
- Format changed Go files with `gofmt`.
- Avoid new dependencies unless they provide a clear benefit.
- Add focused tests for fixes and behavior changes.
- Document new configuration, migrations, and user-visible compatibility differences.
- Preserve default WriteFreely behavior unless a BlueNote difference is deliberate and documented.

## Testing

Run the tests relevant to your change. The focused BlueNote regression tests can be run with:

```sh
go test -mod=readonly ./config
go test -mod=readonly -vet=off -tags sqlite -run TestBlueNote -count=1 .
```

When your environment supports it, also run the broader suite:

```sh
go test -mod=readonly -vet=off -tags sqlite ./...
```

The upstream-derived suite may contain failures unrelated to a proposed change. Report any such failures with the package, test name, and output; do not hide or silently change them as part of unrelated work.

## Pull requests

In the pull request, explain the problem, the resulting behavior, and how you tested it. Call out configuration or database changes and any difference from upstream WriteFreely. Update documentation in the same pull request when users or administrators need to take action.
