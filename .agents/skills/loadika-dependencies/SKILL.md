---
name: loadika-dependencies
description: Rules for implementing or changing external dependency management (yt-dlp, ffmpeg/ffprobe, deno JS runtime) and the proxy pool in Loadika Desktop. Use when touching internal/deps, internal/ytdlp, internal/proxy, internal/download, or the related frontend slices.
---

# Loadika Desktop — dependencies & proxy pool

Use this skill when working on how the app obtains and drives `yt-dlp`, `ffmpeg`/`ffprobe`, the JS
runtime (`deno`/`node`), or the proxy pool. Full details: `docs/04-dependencies.md`,
`docs/05-proxy-pool.md`.

## Dependency management (internal/deps)

- **Never bundle external binaries in the installer.** The installer stays minimal.
  Strategy is **system-first**: detect in `PATH` and standard locations; otherwise download
  on demand (lazily) or when the user explicitly asks.
- Every dependency implements the common `Dependency` interface
  (`Detect`/`Latest`/`Install`/`Update`/`Path`) — new binaries get install/update for free.
- Respect the per-dependency source policy: `auto` (system → managed), `managed`, `system`,
  `force-download`.
- **Atomic installs only**: download to temp → verify checksum → extract → `rename` into place →
  update `manifest.json`. Never leave a half-installed dependency. `chmod +x` on unix.
- Always verify **SHA-256** from the upstream sums file before activating.
- Verify progress is reported via `dep:progress` / `dep:state`, throttled.

### Source specifics (do not guess)
- `yt-dlp`: GitHub `yt-dlp/yt-dlp` standalone binaries (`yt-dlp.exe`, `yt-dlp_linux`,
  `yt-dlp_linux_aarch64`, `yt-dlp_macos`), verify with `SHA2-256SUMS`. Channels: `stable`
  (default), `nightly` (recommended), `master`. Prefer own updater; `--update-to` is a fallback.
  Never use the `.zip` "unpackaged" variants (no self-update).
- `deno` (JS runtime for `yt-dlp-ejs`): GitHub `denoland/deno` release zips per OS/arch, verify
  with `*.sha256sum`. Integrate with `--js-runtimes deno:<path>`. If a system runtime exists,
  prefer it; pass `--no-js-runtimes` before enabling a lower-priority runtime (`node`).
- `ffmpeg`/`ffprobe`: **lazy** — only needed for merge/post-processing. Sources differ per OS:
  Windows/Linux → `yt-dlp/FFmpeg-Builds`; macOS → `ffmpeg-static`/osxexperts. Install ffmpeg and
  ffprobe together. Integrate via `--ffmpeg-location <dir>`; if using system, pass nothing.

## Driving yt-dlp (internal/ytdlp)

- Parse machine-readable output, never human stdout: use `--newline` + `--progress-template`
  (JSON), `--print`/`-J`, `--no-color`.
- Argument building is pure and testable (`args.go`).
- Every run is bound to a `context.Context`; cancellation sends a graceful signal, then kill
  after a timeout. Terminate child processes on app shutdown.
- Throttle progress events (~4–10 Hz) and emit only on meaningful change.

## Proxy pool (internal/proxy)

- A proxy is "available" iff `enabled && health != 'fail' && cooldownUntil <= now`.
- Strategies: `round-robin`, `random`, `least-used`, `lowest-latency`, `sticky`.
- Failures: increment `failCount`; past threshold set `health='fail'` with exponential cooldown.
- **Classify errors**: network/timeout/proxy errors (connect refused/timed out/407/502/503) →
  blame proxy, rotate and retry with a new `--proxy`; logical errors (deleted/private/wrong format)
  → do not rotate, fail the task. Classification is a pure, tested function.
- Reset `cooldownUntil` on startup so proxies don't get stuck.
- Never log/expose proxy credentials; mask them in UI and strip them from error text.

## Safety
- Do not hardcode secrets. Redact credentials before logging or showing errors.
- When adding a new external source, document it in `docs/04-dependencies.md` and prefer
  checksum-verified downloads over git/insecure endpoints.
