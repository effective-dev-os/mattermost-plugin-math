# Open Questions

> OQ-001 — set this project's primary stack and tooling versions explicitly.

## OQ-001 — RESOLVED: Go module path and plugin id renamed from starter-template

Module renamed to `github.com/effective-dev-os/mattermost-plugin-math`, plugin id to `dev.effective.math` (user-confirmed 2026-08-25). All starter-template references removed from go.mod, plugin.json, Makefile, .golangci.yml, server/*.go, README.md, public/hello.html; icon asset renamed to `assets/math-icon.svg`.

## OQ-002 — Math expression evaluation approach unconfirmed

No implementation exists yet. Need to decide: custom expression parser vs. existing Go library, supported operators/functions, error handling for malformed input, and output formatting rules (e.g. `x` vs `*` in the rendered result, per the vision example).
