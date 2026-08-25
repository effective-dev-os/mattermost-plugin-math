# Implementation: `/math` success reply as a dedicated bot post

**Status:** complete
**Date:** 2026-08-25
**Plan:** `swarm-report/math-bot-post-reply-plan-2026-08-25.md`

## Layers executed

1. **backend** (single layer, one exec agent, sonnet) — `server/plugin.go`, `server/command/command.go`, `server/command/command_test.go`, `plugin.json`. ~176s.
2. **orchestrator fix** — `plugin.json` `min_server_version` corrected from `"7.1"` to `"7.1.0"` after live deploy revealed the manifest validator (`semver.Parse`, strict 3-part semver) rejected the 2-part value; the exec agent's own `go build`/`go vet`/`go test` pass couldn't catch this since manifest semver validation only runs server-side at plugin-install time, not at Go compile/test time.

## Files touched

- `server/plugin.go` — `botUserID` field on `Plugin`; `OnActivate` calls `p.client.Bot.EnsureBot` (fails activation on error) then non-fatally `setBotIcon()` (bundle-path read + `SetProfileImage`, warn-and-continue on failure); passes `p.botUserID` into `command.NewCommandHandler`.
- `server/command/command.go` — `botUserID` field on `Handler`; `NewCommandHandler(client, botUserID)` signature; `executeMathCommand` success path posts via `c.client.Post.CreatePost` as the bot, returns `&model.CommandResponse{}` on success or an ephemeral error on `CreatePost` failure. Error (evaluation-failure) path unchanged.
- `server/command/command_test.go` — all 4 `NewCommandHandler` call sites updated; `TestMathCommandSuccess` rewritten with `CreatePost` mock assertions (`UserId`/`ChannelId`/`RootId`/`Message`) and an empty-`CommandResponse` assertion; `TestMathCommandErrors`/`TestUnknownCommand`/`TestMathCommandRegistration` only got the signature fix.
- `plugin.json` — `min_server_version`: `"6.2.1"` → `"7.1.0"` (corrected from an invalid intermediate `"7.1"` after live-deploy failure).
- `assets/math-bot-icon.png`, `assets/math-bot-icon.svg` — staged (were untracked since before this feature).

## Verify results

- `go build ./...` — exit 0.
- `go vet ./...` — exit 0.
- `go test ./...` — exit 0, all packages `ok` (`server`, `server/command`, `server/mathexpr`; `build/pluginctl` cached-ok).
- `gofmt -l` on the 3 touched Go files — no output (clean).
- Re-run independently by the orchestrator (not just trusted from the exec agent) — same result.

## E2E verification (live local Mattermost server)

Deploy: `MM_LOCALSOCKETPATH=<forced-nonexistent> MM_SERVICESETTINGS_SITEURL=http://localhost:8065 MM_ADMIN_TOKEN=<redacted> make deploy` — required forcing the HTTP/token path because `pluginctl`'s default local-mode Unix-socket path hit a different (pre-existing, unrelated to this feature) `app.plugin.upload_disabled.app_error` that the HTTP+token path did not reproduce; not investigated further since the HTTP path is an equally valid, documented deploy mechanism and succeeded cleanly.

First deploy attempt failed: `failed to parse MinServerVersion` — root cause `"min_server_version": "7.1"` is not valid 3-part semver per `github.com/blang/semver`. Fixed to `"7.1.0"`, rebuilt, redeployed — succeeded, plugin shows `active` via `GET /api/v4/plugins`.

Verified via direct REST calls (`GET /api/v4/plugins`, `POST /api/v4/users/search`, `GET /api/v4/users/{id}/image`, `POST /api/v4/commands/execute`, `GET /api/v4/channels/{id}/posts`):

- Bot account `math-bot` exists: `is_bot: true`, `first_name: "Math Bot"`, `bot_description: "Posts results for the /math slash command."`.
- Bot avatar: `GET .../image` returns a 128x128 PNG (byte-for-byte the bundled `assets/math-bot-icon.png` via the bundle-path read).
- `/math 50% + 10` → HTTP response `text: ""`, `response_type: ""` (no double-post); channel post authored by the bot's exact user ID, `message: "`50% + 10` = `10.5`"`.
- `/math 2*(3+4)` → bot post `"`2*(3+4)` = `14`"`.
- `/math sin(90)` → bot post `"`sin(90)` = `1`"`.
- `/math 1/0` → HTTP response `response_type: "ephemeral"`, `text: "Result is not a finite number (e.g. division by zero)."`; no channel post created (confirmed via post listing — error path unchanged, no bot involvement).
- Browser-based visual screenshot was attempted but blocked (`Local hostnames are not allowed` — the available browser tool is sandboxed against `localhost`); REST-level evidence above is direct and conclusive (post author user ID matches the bot account ID exactly, avatar bytes confirmed, message content byte-exact) so this is not treated as a verification gap.

## Out-of-scope (declared, carried over from plan)

- DM/group-DM channel-type special-casing.
- Un-swallowing the user's own `/math ...` invocation text (confirmed platform-level, unfixable).
- Bot account cleanup on `OnDeactivate`/uninstall.
- Normalizing the echoed expression string (stays raw/trimmed).
- Any change to `server/mathexpr/*`.

## Open issues raised during implementation

- `plugin.json` `min_server_version` semver-format bug (`"7.1"` vs `"7.1.0"`) was not caught by any Go-level verify step (`go build`/`vet`/`test` don't touch manifest JSON semantics) — only surfaced via the live-deploy E2E step. Worth noting for future plan/verify-contract design: any `plugin.json` field with server-side format validation should get a manifest-lint or deploy-dry-run step, not just Go tooling.
- `pluginctl`'s local Unix-socket deploy path failed with `app.plugin.upload_disabled.app_error` on this test server while the HTTP+token path succeeded against the same server — pre-existing test-infra quirk, unrelated to this feature's code, not investigated further (out of scope).

## Suggested commit message

```
Post /math results from a dedicated bot account instead of the raw command response

Users reported that /math replies read as if their own message got replaced.
Register a "Math Bot" bot account (avatar from assets/math-bot-icon.png) and
post results as that bot instead of returning them via CommandResponse text,
so the reply has a clearly distinct identity. Error responses stay ephemeral.
```

## Next

`/post-feature math-bot-post-reply` — append D-004 to `.assistant/decisions.md`, close out scope notes, draft PR text. Not committed yet (human gate).
