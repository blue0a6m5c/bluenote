# BlueNote version information

`internal/version/version.go` is the source of truth for the BlueNote version,
the WriteFreely upstream base, and the BlueNote source URL. Update the appropriate
constant there when preparing a BlueNote release or integrating a new upstream base.

BlueNote is based on WriteFreely v0.17.2 and is synchronized with upstream
`develop` through commit `049aa2a25af41d20c400f2f9794b285f7b2958ec`. The
plain upstream version remains `0.17.2` until WriteFreely publishes a newer
release; the synchronization commit records the additional development changes.

CLI output identifies both products and the build revision. The executable is still
named `writefreely`. Plain `go build` uses `unknown` for the revision. Makefile build
targets inject the short Git commit into `internal/version.Revision` using linker
flags; when Git or repository history is unavailable they also use `unknown`.
The revision identifies the base commit, not uncommitted changes.

The shared server-rendered branding template preserves the WriteFreely attribution
and shows the BlueNote version and source link. Admin pages label the BlueNote
version and WriteFreely base separately.

NodeInfo keeps `software.name = writefreely`, and its version is generated as
`<upstream-base>+bluenote.<bluenote-version>`. Outbound User-Agent strings use the
same compatibility version. NodeInfo source metadata points to BlueNote. The HTTP
Server header remains `WriteFreely`.

The compatibility version is an identification string, not a way to compare
BlueNote releases: SemVer ignores build metadata for ordering. Writer guide links,
upstream release notes, and the existing update checker continue to use the plain
WriteFreely base version. The checker only reports upstream availability; it does
not check BlueNote releases or install updates.

Phase 1 changes binary version embedding only. Existing release archive naming
still uses `GITREV` / `git describe`; release and Docker workflows and tag creation
are deferred. Do not interpret those legacy archive names as BlueNote versions.
