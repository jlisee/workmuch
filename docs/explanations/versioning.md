# Versioning

WorkMuch release versions are derived from Git history:

```text
YYYYMMDD.SEQUENCE.DISTANCE+gCOMMIT
```

For example, `20260805.43.5+g69d002bbb362` contains:

- `YYYYMMDD`: the UTC committer date of the release commit's merge-base with
  the fetched origin default branch
- `SEQUENCE`: the first-parent commit count through that merge-base
- `DISTANCE`: the number of commits reachable from the release commit but not
  from the merge-base; this is zero when releasing the tip of the default branch
- `COMMIT`: the first 12 characters of the release commit hash

`cmd/workmuch-version` performs the calculation and requires complete,
non-shallow Git history. `release.sh` creates an annotated `v<version>` tag and
records the full merge-base hash in its `WorkMuch-Main-Base` metadata. The
release workflow validates both the tag and recorded base before building, so
the result remains reproducible after the default branch advances.

In source builds, `internal/buildinfo.Version` defaults to `dev`. GoReleaser
replaces it at link time with:

```text
-X workmuch-go/internal/buildinfo.Version=<version>
```

The `--version` command and tray About screen read this variable. GoReleaser
also uses the same version for Debian package metadata, filenames, and the
GitHub Release.
