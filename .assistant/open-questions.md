# Open Questions

> OQ-001 — set this project's primary stack and tooling versions explicitly.

## OQ-001 — RESOLVED: Go module path and plugin id renamed from starter-template

Module renamed to `github.com/effective-dev-os/mattermost-plugin-math`, plugin id to `dev.effective.math` (user-confirmed 2026-08-25). All starter-template references removed from go.mod, plugin.json, Makefile, .golangci.yml, server/*.go, README.md, public/hello.html; icon asset renamed to `assets/math-icon.svg`.

## OQ-002 — RESOLVED: Math expression evaluation approach

Decided via `/research` consilium + human sign-off: `github.com/expr-lang/expr` (>=v1.17.7) as the evaluation engine, plus a text-normalization pass for human notation (×/÷/x, unicode minus/dashes, comma decimals, implicit multiplication, trailing `%`). `sin`/`cos` accept degrees. Full rationale: `.assistant/decisions.md` D-002, `swarm-report/research-go-expr-library-2026-08-25.md`.

## OQ-003 — RESOLVED: does `pluginapi.Client.Bot.EnsureBot`/`EnsureBotUser` actually require server ≥7.1 at runtime?

Raised during `/pre-feature` for the bot-post feature: `github.com/mattermost/mattermost/server/public@v0.1.21`'s vendored `EnsureBotUser` doc comment states "Minimum server version: 7.1", while `pluginapi.BotService.EnsureBot`'s own internal soft version-check only requires 5.10.0 — an inconsistency within the same pinned dependency version. Resolved empirically, not by further doc archaeology: `plugin.json` was set to `"min_server_version": "7.1.0"` and the plugin deployed and activated successfully against the local test Mattermost server, with `EnsureBotUser` creating the `math-bot` account without error. Treat 7.1.0 as the safe declared floor going forward. Full detail: `.assistant/decisions.md` D-004, `swarm-report/math-bot-post-reply-implementation-2026-08-25.md`.
