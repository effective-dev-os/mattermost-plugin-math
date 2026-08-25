# Stack

## Detected

- **Language:** Go (`go.mod`, module `github.com/effective-dev-os/mattermost-plugin-math`), Go 1.25
- **Server framework:** Mattermost plugin API (`github.com/mattermost/mattermost/server/public`), router `github.com/gorilla/mux`
- **Expression evaluation:** `github.com/expr-lang/expr` (>=v1.17.7) — see `.assistant/decisions.md` D-002 for version-pin rationale (two distinct DoS advisories) and the required input-normalization pass for human notation (`×`/`x`, unicode minus, comma decimals, implicit multiplication, `%`). `sin`/`cos` accept degrees, converted to radians before calling `math.Sin`/`math.Cos`.
- **Webapp:** TypeScript + React (`webapp/package.json`), Webpack build, Jest for tests, ESLint (`@mattermost/eslint-plugin`), `tsc` for type-checking. No UI planned for v1 — `/math` is a server-only slash command.
- **Build:** `Makefile`, Node v16 / npm v8 pinned via `.nvmrc`
- **Plugin id:** `dev.effective.math`

## TODO

- Document server↔webapp slash-command wiring conventions once implemented
