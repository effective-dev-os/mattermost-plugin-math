# Stack

## Detected

- **Language:** Go (`go.mod`, module `github.com/effective-dev-os/mattermost-plugin-math`), Go 1.25
- **Server framework:** Mattermost plugin API (`github.com/mattermost/mattermost/server/public`), router `github.com/gorilla/mux`
- **Expression evaluation:** `github.com/expr-lang/expr` (>=v1.17.7) — see `.assistant/decisions.md` D-002 for version-pin rationale (two distinct DoS advisories) and the required input-normalization pass for human notation (`×`/`x`, unicode minus, comma decimals, implicit multiplication, `%`). `sin`/`cos` accept degrees, converted to radians before calling `math.Sin`/`math.Cos`.
- **Webapp:** TypeScript + React (`webapp/package.json`), Webpack build, Jest for tests, ESLint (`@mattermost/eslint-plugin`), `tsc` for type-checking. No UI planned for v1 — `/math` is a server-only slash command.
- **Build:** `Makefile`, Node v16 / npm v8 pinned via `.nvmrc`
- **Plugin id:** `dev.effective.math`

## /math slash command — implementation details

- **Server-only slash command**, registered in `server/command/command.go` via `client.SlashCommand.Register` inside `NewCommandHandler` (called from `OnActivate` in `server/plugin.go`). No webapp-side registration or UI needed.
- **Expression evaluation:** new `server/mathexpr` package (zero Mattermost imports, pure functions), handles normalization (`×`/`÷`/unicode minus → ASCII, comma → decimal point, implicit multiplication, percent-rewrite), then `expr.Compile`/`Run`.
- **Input cap:** 1024 runes (via `utf8.RuneCountInString`), enforced before `expr.Compile`. Rejects input before normalization for accurate cap (not post-normalization, since some normalization steps emit extra characters).
- **Trig/log conventions:** `sin`/`cos` accept degrees (converted to radians before `math.Sin`/`math.Cos`), `log` is base-10 (`math.Log10`), both matching calculator user expectations per D-002 rationale.
- **Percent normalization:** every `%` occurrence is rewritten as `(operand/100)` regardless of position, removing expr-lang's integer-modulo operator from user-facing input. `50% + 10` → `(50/100) + 10` = 0.5. Pipeline order: length cap → unicode symbols → comma decimals → percent rewrite → implicit multiplication → character allowlist.
- **Success replies post as a dedicated bot** (`math-bot` / "Math Bot", avatar `assets/math-bot-icon.png`), not via the raw `CommandResponse` — see `.assistant/decisions.md` D-004 for full rationale. Bot is registered idempotently in `OnActivate` via `p.client.Bot.EnsureBot` (fails activation on error); avatar is set separately and non-fatally. `executeMathCommand` posts `` `<expr>` = `<result>` `` via `client.Post.CreatePost` as the bot and returns an empty `CommandResponse{}` on success, or an ephemeral error if `CreatePost` fails. Evaluation-error responses are unchanged: ephemeral `CommandResponse`, no bot involvement. `plugin.json`'s `min_server_version` is `7.1.0` (required by `EnsureBotUser`; must be full 3-part semver or the manifest fails to parse on deploy).
