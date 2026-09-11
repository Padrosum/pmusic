# Security model

## Lua plugins and themes

pmusic Lua extensions are trusted code, not a sandbox. An enabled extension can
use the APIs exposed by pmusic and runs with the same user identity as pmusic.
Review a plugin before enabling it, keep the local Lua directory private, and
disable code whose origin you cannot verify.

## Remote store provenance

`pmusic sync` uses an application-versioned manifest. Each entry points to an
immutable repository commit, has a SHA-256 digest, and is downloaded with a
timeout and a one-megabyte size limit. pmusic installs a file only after the
digest matches, using an atomic replacement in the destination directory. A
failed, partial, oversized, or non-successful download leaves the installed
file unchanged. The command prints the release and source URL for inspection.

The manifest and application currently ship through the same repository and
release channel. Hashes detect corruption and unexpected content, but they are
not a cryptographic signature against compromise of that channel. Independent
signature verification and key distribution remain future work.

## Reviewing local extensions

Extensions live under the user configuration directory, normally
`~/.config/pmusic/lua/plugins` and `~/.config/pmusic/lua/themes`. Compare a
file's SHA-256 digest and source URL with the release manifest, then inspect the
Lua source before enabling it from the store screen.

## Optional GenLang catalog

`library.gl` is data, not code: GenLang documents cannot run commands or open
the network. pmusic loads them by running `genlang convert <absolute .gl>
--json` with a timeout and a JSON size cap, the same pattern used for
`yt-dlp`. The path is chosen by pmusic from the config or music directory; it
is not taken from command arguments. If `genlang` is missing, the overlay is
skipped and local playback continues.

`pmusic -s` also downloads catalog packages into `~/.config/pmusic/gl/` using
the same commit pin, size limit, SHA-256 digest, and atomic write as Lua
files. The active overlay remains `~/.config/pmusic/library.gl` (or the music
directory copy). Sync updates the package under `gl/` every time and seeds
`library.gl` only when that file is missing; an existing catalog is never
overwritten.

## Reporting vulnerabilities

Do not publish exploit details in a public issue before maintainers have had a
chance to respond. Use GitHub's private security advisory/reporting feature for
the canonical `Padrosum/pmusic` repository when available.
