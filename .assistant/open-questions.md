# Open Questions

> OQ-001 — set this project's primary stack and tooling versions explicitly.

## OQ-001 — RESOLVED: Go module path and plugin id renamed from starter-template

Module renamed to `github.com/effective-dev-os/mattermost-plugin-math`, plugin id to `dev.effective.math` (user-confirmed 2026-08-25). All starter-template references removed from go.mod, plugin.json, Makefile, .golangci.yml, server/*.go, README.md, public/hello.html; icon asset renamed to `assets/math-icon.svg`.

## OQ-002 — RESOLVED: Math expression evaluation approach

Decided via `/research` consilium + human sign-off: `github.com/expr-lang/expr` (>=v1.17.7) as the evaluation engine, plus a text-normalization pass for human notation (×/÷/x, unicode minus/dashes, comma decimals, implicit multiplication, trailing `%`). `sin`/`cos` accept degrees. Full rationale: `.assistant/decisions.md` D-002, `swarm-report/research-go-expr-library-2026-08-25.md`.
