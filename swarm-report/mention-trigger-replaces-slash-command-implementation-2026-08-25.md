# Implementation: mention-trigger-replaces-slash-command

Status: complete (E2E verified against the live local Mattermost test server)

## E2E verification (live local Mattermost server, http://localhost:8065)

Deployed via `MM_SERVICESETTINGS_SITEURL=http://localhost:8065 MM_ADMIN_TOKEN=<env-var-only, discarded> make deploy`. `pluginctl`'s local-mode socket path failed with "Plugins and/or plugin uploads have been disabled." despite `PluginSettings.EnableUploads`/`Enable` both reading `true` via `/api/v4/config` — worked around by uploading the built bundle directly via `POST /api/v4/plugins` with the admin bearer token (`curl -F plugin=@dist/... -F force=true`), which succeeded and left the plugin in state 2 (running), `dev.effective.math` v`0.0.0+935a688`. Root cause of the local-mode discrepancy not investigated further (out of scope — deploy succeeded via the REST path).

Posted 6 real messages as the `danil` test user into `town-square` (`7irnjkb3njgcmezn74tsr7uqwr` / `doejterqzib5ucmariadke88uy`) via the REST API and inspected the resulting channel post list (`GET /channels/{id}/posts`), confirming both (a) the user's own message is stored completely unmodified and (b) the bot's reply, authored by `math-bot` (`gwigs33wgfy3idi6com88c4i1r`, confirmed via `GET /users/username/math-bot`, `is_bot: true`), has `root_id` set to the triggering post's own id:

| # | Message posted | User's message preserved? | Bot reply (`root_id` = triggering post id) |
|---|---|---|---|
| 1 | `@math-bot 50% + 10` (leading mention) | yes, unmodified | `` `50% + 10` = `10.5` `` |
| 2 | `sin(90) @math-bot` (trailing mention) | yes, unmodified | `` `sin(90)` = `1` `` |
| 3 | `@math-bot` (bare mention) | yes, unmodified | `Mention me with a math expression, e.g. \`@math-bot 2 + 2\`.` |
| 4 | `@math-bot 2+` (malformed, leading mention) | yes, unmodified | `Could not parse expression.` (threaded, non-ephemeral, per the resolved error-delivery design) |
| 5 | `hey @math-bot what is 2+2` (mid-message mention) | yes, unmodified | **no reply posted** — confirms the Option B decision (mid-message mention is a silent no-op) |
| 6 | `@math-bot2 2+2` (mention as substring of a longer username) | yes, unmodified | **no reply posted** — confirms the boundary check rejects `@math-bot2` |

All 6 outcomes match the plan exactly. No infinite loop / self-reply observed (the bot's own reply posts never triggered a further reply). No unmodified user messages were altered by the plugin at any point — the original ask ("the user's message should NOT be swallowed") is directly satisfied.

## Layers executed

1. **Backend** (`server/`, `Makefile`, `go.mod`/`go.sum`, `plugin.json`) — sonnet exec agent, ~206s. Full diff spot-checked by orchestrator, independently re-verified (`gofmt -l`, `go build`, `go vet`, `go test -v`, all clean).
2. **Docs** (`README.md`, `CLAUDE.md`, `.memory-bank/tech-details/stack.md`) — haiku exec agent, ~49s. `README.md` and `stack.md` edits landed and were spot-checked (diff above). `CLAUDE.md` edit was blocked by the harness's `self-config-guard` PreToolUse hook (`.claude/hooks/self-config-guard.sh`) — by design, it denies `Edit`/`Write` on `CLAUDE.md`/`AGENTS.md`/`.claude/*` in any consuming install (`.harness-lock` present) to prevent governing-config drift as a side effect of a feature task. Not bypassed. See "Open issues" below.

## Files touched

- **Deleted:** `server/command/command.go`, `server/command/command_test.go`, `server/command/mocks/mock_commands.go` (and the now-empty `server/command/` tree).
- **New:** `server/hooks.go` (146 lines), `server/hooks_test.go` (149 lines).
- **Modified:** `server/plugin.go` (-24/+12 net, `ExecuteCommand`/`commandClient` removed, `botUsername` const added, bot `Description` updated), `Makefile` (-6, `mock:` target removed), `go.mod`/`go.sum` (`go.uber.org/mock` dropped via `go mod tidy`), `plugin.json` (description string), `README.md` (intro line + Command-package subsection removed), `.memory-bank/tech-details/stack.md` (Webapp bullet + full `/math` section rewritten to `@math-bot mention trigger`).
- **Not touched (out of scope, per plan):** `server/mathexpr/*`, `CLAUDE.md` (blocked, see above), `AGENTS.md` (confirmed by reviewer consilium pass to contain no `/math` reference — nothing to change), `webapp/src/manifest.ts` / `server/manifest.go` (auto-generated from `plugin.json` via `./build/bin/manifest apply`, regenerated but gitignored/untracked so no tracked diff).

## Per-agent verbatim summaries

### backend exec agent (sonnet)
Implemented `server/hooks.go`/`server/hooks_test.go` exactly as specified by the orchestrator (who had pre-verified every Mattermost API call against the pinned `server/public@v0.1.21` source). Found and fixed one test-side bug during verification: `hooks_test.go` as originally specified didn't stub `LogDebug` on `plugintest.API`, causing `TestMessageHasBeenPosted_EvaluationError` to panic on the unmocked call from `hooks.go`'s `p.client.Log.Debug("mention math evaluation failed", "error", err)`. Fixed by adding `api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()` to the shared `setupHookTest()` helper. No bugs found in the `extractMathMention`/`stripLeadingMention`/`stripTrailingMention` logic itself — all 10 table-driven cases passed as specified.

verify: `gofmt -l server/` (clean) · `go build ./server/...` (0) · `go vet ./server/...` (0) · `go test ./server/... -v` (0, all pass) · `go mod tidy && git diff go.mod go.sum` (clean, `go.uber.org/mock` removed).

### docs exec agent (haiku)
4 of 5 requested edits landed clean (README.md x2, stack.md x2). CLAUDE.md edit correctly self-reported as blocked by `self-config-guard`, did not attempt a workaround.

## Verify results (orchestrator, independent re-run after both layers)

```
$ gofmt -l server/ && go build ./server/... && go vet ./server/... && go test ./server/...
ok  	github.com/effective-dev-os/mattermost-plugin-math/server	(cached)
ok  	github.com/effective-dev-os/mattermost-plugin-math/server/mathexpr	(cached)
?   	github.com/effective-dev-os/mattermost-plugin-math/server/store/kvstore	[no test files]
```
All green, exit 0. Diffs for every changed file read and spot-checked directly by the orchestrator (not just trusted from agent self-report), per `AGENTS.md` "Trust but verify."

## Out-of-scope (declared, carried over from plan)

`server/mathexpr/*` reused unchanged; no `MessageHasBeenUpdated` (edit re-triggering); no DM-only (no-mention) interaction mode; no channel/team admin opt-out toggle; no streaming/progressive ephemeral UX.

## Open issues raised during implementation

1. **`CLAUDE.md` still reads `/math <expression>`** (line 3) — blocked by `self-config-guard`, requires a deliberate human edit or `/sync`, not fixable from within this task. Flagged to the user in the final report; not a silent gap.
2. D-005 decision entry (superseding D-003/D-004's trigger/transport portions) still needs to be written — that's `/post-feature`'s job, next step.
3. OQ-003 addendum (confirm `min_server_version` 7.1.0 still holds, no reopen) — also `/post-feature`'s job.
4. Live E2E deploy + manual `@math-bot` channel-message verification — done, see "E2E verification" section above. All 6 scenarios passed.

## Suggested commit message (draft only, not yet committed)

```
Replace /math slash command with @math-bot mention trigger

Slash commands are intercepted client-side by Mattermost and never posted
as a channel message, which made the /math response look like the user's
own message had been swallowed. Per explicit user direction, the trigger
is now a MessageHasBeenPosted hook that reacts to @math-bot mentioned as
the first or last token of a channel message, leaving the user's message
completely untouched and replying as a threaded post from the bot account.

- server/command/ removed entirely; server/hooks.go implements the hook,
  reusing server/mathexpr's evaluation pipeline unchanged (D-002/D-003).
- Mention detection is leading-or-trailing-token-only, not anywhere in the
  message: mid-message stripping was found to silently produce wrong
  results or unparseable expressions during /pre-feature review.
- Errors are now a threaded bot reply (not ephemeral) since
  MessageHasBeenPosted has no request-bound ephemeral guarantee.

Supersedes the trigger/transport portions of D-003/D-004; D-002/D-003's
normalization, percent, and log-base decisions are unchanged. See D-005.
```

## Next

`/post-feature mention-trigger-replaces-slash-command` — append D-005 to `.assistant/decisions.md`, append the OQ-003 addendum, close out. E2E verification is complete; ready for commit after `/post-feature`.
