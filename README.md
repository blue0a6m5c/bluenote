# BlueNote

BlueNote is an unofficial, early-stage fork of [WriteFreely](https://github.com/writefreely/writefreely), a minimalist and federated publishing platform. It is currently based on [WriteFreely v0.17.2](https://github.com/writefreely/writefreely/releases/tag/v0.17.2).

BlueNote adds a small set of features and fixes while aiming to remain compatible with upstream WriteFreely. It is independently maintained and is not affiliated with or endorsed by WriteFreely or [Musing Studio](https://musing.studio).

> [!CAUTION]
> BlueNote is in early development. There are no stable BlueNote releases or pre-built binaries yet. Review changes carefully before using it in production.

## BlueNote changes

Compared with the v0.17.2 upstream base, BlueNote currently includes:

- Configurable datetime-based post slugs generated from a post's saved publication time, with an IANA timezone setting and UTC as the default. The feature is disabled by default to preserve WriteFreely behavior. See [Datetime slugs](docs/datetime-slugs.md).
- UTC serialization for post creation and update timestamps emitted in ISO 8601 format.
- Plain-text `summary` values for ActivityPub `Article` objects.
- Correct termination of empty ActivityPub followers and following collection pages.

These changes have regression coverage so that future upstream updates can be evaluated against BlueNote's behavior.

## About WriteFreely

WriteFreely provides a focused writing experience for individual blogs and multi-user communities. It supports ActivityPub federation, multiple blogs per account, OAuth 2.0, hashtags, pinned pages, and SQLite or MySQL/MariaDB storage.

The upstream project provides its [documentation](https://writefreely.org/docs), [installation guide](https://writefreely.org/start), and [source repository](https://github.com/writefreely/writefreely). Those resources describe WriteFreely and may differ from BlueNote where this fork has changed behavior.

## Development

BlueNote requires Go 1.25.0 or later. Clone this repository to work with the BlueNote source:

```sh
git clone https://github.com/blue0a6m5c/bluenote.git
cd bluenote
```

The upstream [developer setup guide](https://writefreely.org/docs/latest/developer/setup) is a useful reference for the build process and shared development dependencies. Its `go install github.com/writefreely/writefreely/cmd/writefreely@latest` and `git clone https://github.com/writefreely/writefreely.git` commands fetch upstream WriteFreely rather than BlueNote, so do not use those commands when setting up a BlueNote checkout.

BlueNote-specific configuration is documented in this repository's [`docs`](docs) directory.

Contributions are welcome through this repository. Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening an issue or pull request. Report suspected vulnerabilities according to [SECURITY.md](SECURITY.md).

## Upstream and attribution

BlueNote is derived from WriteFreely. Upstream development is maintained at [writefreely/writefreely](https://github.com/writefreely/writefreely). The authors inherited from the upstream project are recorded in [AUTHORS.md](AUTHORS.md), and the Git history retains the full contribution record.

WriteFreely copyright © 2018-2026 [Musing Studio LLC](https://musing.studio) and contributing authors.

BlueNote modifications copyright © 2026 [blue0a6m5c](https://github.com/blue0a6m5c) and BlueNote contributors.

## License

BlueNote is free software licensed under the [GNU Affero General Public License, version 3](LICENSE), consistent with the upstream WriteFreely project.
