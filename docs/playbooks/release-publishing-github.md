# Playbook: Publishing Releases to a Private GitHub Repo

This is the upstream half of the self-updating daemon playbook. It describes
how to build a versioned hestia binary and publish it as a private GitHub
Release that the [systemd daemon](self-update-github-systemd.md) will
auto-install.

## Versioning contract

The updater compares versions with **semantic versioning**. The running app's
version must be a semver and the release tag must be a semver. A `v` prefix is
allowed on both sides; any non-semver value (e.g. `dev`) silently never
updates.

- The **release tag** becomes the reported version (`v1.2.0`).
- The running binary's version is set at build time via `-ldflags "-X main.version=<tag>"`.
- The daemon picks up the latest **published, non-draft, non-prerelease**
  release whose tag is *newer* than its running version. Older tags are ignored.

Keep the tag and the baked version identical, e.g. both `v1.2.0`.

## Asset naming contract

The daemon finds its asset by **exact name match** against
`UPDATE_GITHUB_ASSET_PATTERN`, expanded with `{version}`, `{os}`, `{arch}`
and `{ext}`:

| Placeholder | Value |
|---|---|
| `{version}` | the release tag, e.g. `v1.2.0` |
| `{os}` | `linux` (the daemon host's GOOS) |
| `{arch}` | `amd64` (the daemon host's GOARCH) |
| `{ext}` | empty on Linux |

So the pattern `hestia-{version}-{os}-{arch}` expects an asset literally named
`hestia-v1.2.0-linux-amd64`. The asset name must match exactly — a suffix or a
`.tar.gz` wrapper will not be found.

Publish one asset per platform (`linux-amd64`, `linux-arm64`, …); each daemon
host resolves its own.

## Token setup

The GitHub provider authenticates with a PAT. Two options:

- **Fine-grained token (recommended):** repository access → *Selected
  repositories* → the private repo → permissions → **Contents: Read**. This
  grants read access to the release API and asset downloads, nothing else.
- **Classic PAT:** `repo` scope.

A token also lifts the anonymous API rate limit (5,000 requests/hour instead
of 60/hour per IP). Keep it in the daemon's `EnvironmentFile` with
`0600` permissions.

## Build

```bash
# from the repo root of your hestia app (here: the built-in test-server)
VERSION=v1.2.0
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-X main.version=${VERSION}" \
  -o hestia-${VERSION}-linux-amd64 ./cmd/test-server
```

`main.version` must exist in your `main` package (`var version = "dev"`) and be
wired into `hestia.Setup(SetupConfig{Version: version})` or `APP_VERSION`.

Sanity check before uploading:

```bash
./hestia-v1.2.0-linux-amd64 --version   # prints v1.2.0
```

## Publish

With `gh` authenticated to a repo your PAT can reach:

```bash
gh release create v1.2.0 \
  --repo <owner>/<private-repo> \
  --title "v1.2.0" \
  --notes "What changed in this release" \
  ./hestia-v1.2.0-linux-amd64
```

Do **not** mark the release as a draft or prerelease — `/releases/latest`
skips both.

## Verifying the pipeline before a fleet deploy

With one daemon running against the repo:

```bash
# status shows the running version and the staged one
curl -s http://localhost:PORT/api/system/updates/status/get
```

Publish `v1.1.0` (or trigger the check endpoint), confirm the status endpoint
reports `staged_version`, then apply.

## Rollback

Rollback is **monotonic**: the daemon only ever moves forward to the newest
tag, so re-publishing an *older* tag does nothing. To revert to old behavior
you must publish a **higher** version (e.g. `v1.2.1`) containing the reverted
code, or manually place an older binary and restart the unit.

## Integrity note

The GitHub provider downloads the asset over HTTPS but performs **no
checksum or signature verification** (unlike the RSA-signed update-server
provider). Trust rests on: HTTPS transport, the PAT's read-only scope, and the
repo being private. For integrity-hardened, centrally-controlled fleets, use
the signed update-server provider instead of GitHub Releases.

## Next

Now configure the daemon itself: [self-update-github-systemd.md](self-update-github-systemd.md).