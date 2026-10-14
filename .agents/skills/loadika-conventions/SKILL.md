---
name: loadika-conventions
description: Coding conventions and architecture rules for the Loadika Desktop repository (Wails v3 + Go backend, Vue 3 + TypeScript-strict frontend organised with FSD). Use whenever writing, reviewing, or refactoring code in this repo to keep it readable, strictly typed, leak-free, and FSD-compliant.
---

# Loadika Desktop — conventions & architecture

Apply these rules to any code change in this repository. This skill is the short, enforceable
checklist; the full rationale lives in the docs.

The portable repo standard (stack, skeleton, boundaries, verification, FSD) is
[`PROJECT-BLUEPRINT.md`](../../../PROJECT-BLUEPRINT.md); agent rules are [`AGENTS.md`](../../../AGENTS.md).
This skill points to them — it does not restate them.

Before a non-trivial change, read the relevant doc:

- `docs/03-frontend-fsd.md` — FSD structure and import rules.
- `docs/06-conventions.md` — TS/Go style, naming, performance, leak rules.
- `docs/02-backend.md` — Go package layout, services, events.
- `docs/01-tech-stack.md` — pinned versions.

## Hard rules (must hold for every change)

### Architecture
- Frontend import direction is strictly downward: `app → pages → widgets → features → entities → shared`.
  Importing upward or reaching into another slice's internals is forbidden; use the slice's `index.ts`.
- Each FSD slice exposes a public API only via `index.ts`. No deep imports into `*/model`, `*/ui`, `*/api`.
- `shared` must not reference business entities.
- Go: services in `internal/services` are thin adapters (validate → call domain → return DTO).
  Business logic lives in domain packages (`download`, `deps`, `proxy`, `config`, `ytdlp`).
- Frontend never calls generated Wails bindings directly; go through `shared/api` wrappers.

### TypeScript / Vue
- Strict TS only. `any` is forbidden — use `unknown` and narrow.
- Use `import type` for type-only imports (`verbatimModuleSyntax`).
- `<script setup lang="ts">`; `defineProps`/`defineEmits` with type-only syntax.
- Prefer union types over `enum`.
- Derived values via `computed`, not `watch`.
- Large lists / heavy objects: `shallowRef` + `markRaw`.
- Every subscription (Wails events, listeners, timers) is unsubscribed in
  `onScopeDispose`/`onUnmounted`. No leaks.

### Go
- `gofmt`/`goimports` clean; `go vet` clean.
- Return errors, wrap with `%w`, check with `errors.Is/As`. No panics for expected failures.
- `context.Context` first arg, never stored in structs.
- No mutable globals; inject dependencies via constructors.
- Long-running work must honor context cancellation (yt-dlp processes, downloads).

### Naming
- Go: packages lowercase; types/methods PascalCase.
- TS: types PascalCase; functions/vars camelCase; Vue SFC files PascalCase.
- FSD folders kebab-case (e.g. `manage-proxy-pool`).
- Events: `domain:action` (e.g. `task:progress`). Event names are declared once in Go
  (`internal/events`) and mirrored in `shared/config/events.ts`.

## Before finishing
- `vue-tsc`/`tsc` and `go build`/`go vet` pass.
- No unhandled subscriptions or orphaned goroutines/processes.
- If behavior deviates from `docs/`, update the docs in the same change.
