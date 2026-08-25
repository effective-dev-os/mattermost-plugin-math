# Plan: mention-trigger-replaces-slash-command

Status: consilium-complete

## Task container

- slug: mention-trigger-replaces-slash-command
- scope-fence: `server/command/` (deleted), `server/plugin.go`, new `server/hooks.go` (+ `server/hooks_test.go`), `plugin.json`, `webapp/src/manifest.ts` (description mirror only), `Makefile` (`mock:` target removal), `go.mod`/`go.sum` (drop `go.uber.org/mock` if orphaned), `README.md`, `CLAUDE.md`, `.memory-bank/tech-details/stack.md`, `.assistant/decisions.md` (append D-005), `.assistant/open-questions.md` (append note to OQ-003, no reopen). Out of scope: `server/mathexpr/*` (reused as-is, D-002/D-003 unchanged), webapp bundle logic beyond the manifest description string, bot avatar/icon assets.
- definition-of-done:
  - `/math` slash command registration and dispatch fully removed, no dead code in `server/command/`.
  - `MessageHasBeenPosted` implemented on `*Plugin`, guards against self/plugin/webhook/system/bot re-entrancy, detects `@math-bot` mention per the extraction rule confirmed with the human (see Blockers), evaluates via unchanged `server/mathexpr`, replies threaded (`RootId = post.RootId` if set else `post.Id`).
  - Error path decided and implemented (see Blockers/Concerns — SendEphemeralPost reliability).
  - Table-driven unit tests cover mention-detection/extraction edge cases and hook guard chain; obsolete slash-command tests removed, not left dangling.
  - `plugin.json`, `README.md`, `CLAUDE.md`, `.memory-bank/tech-details/stack.md` no longer describe `/math <expression>` as the invocation syntax.
  - `go build ./...`, `go vet ./...`, `go test ./...` all pass.
  - Live E2E: deploy to local Mattermost, post real `@math-bot ...` messages, confirm user's message stays untouched and bot reply lands as a thread reply with correct content.
- verification-contract: `make dist` or `go build ./...` + `go vet ./...` + `go test ./... -v`; live deploy via `MM_SERVICESETTINGS_SITEURL=... MM_ADMIN_TOKEN=... make deploy` + manual channel messages per the task's E2E script.
- inputs-needed: `MM_ADMIN_TOKEN` (env var only, already available to orchestrator, never committed).
- out-of-scope: DM-only interaction (bot has no channel-independent trigger — accepted per consilium, see Notes), edited-message re-evaluation (`MessageHasBeenUpdated` not implemented), per-channel/team admin opt-out toggle, `UpdateEphemeralPost`-based streaming UX.

## TL;DR

- 4/4 consilium members returned (architect required one retry after an API-error truncation on the first attempt — retry succeeded).
- Total findings: 13 HIGH, 10 MEDIUM, 6 LOW (post-dedup), plus 26 confidence-flagged researcher findings.
- Top 3 must-fix items:
  1. **Genuine spec conflict**: the task brief's own examples require `@math-bot` mention detection anywhere in the message (start/middle/end — e.g. `what is @math-bot 2+2`), but architect + skeptic independently (both HIGH confidence) found this produces silently-wrong results (`sqrt @math-bot (4)` → evaluates to `2`, a result never intended) or hard parse failures (`2+2 @math-bot 3+3`). Consilium recommends leading-token-only extraction. **This overrides an explicit brief requirement — escalated to human, not resolved unilaterally.**
  2. `MessageHasBeenPosted` fires for every post server-wide (all channels/teams/DMs, no membership filter) — confirmed by both skeptic and researcher against server source. This is inherent to the hook the brief itself specified, not a new choice, but it's a real scope expansion over the old per-invocation `/math` command and is recorded as an accepted trade-off, not blocked.
  3. `p.API.SendEphemeralPost` is structurally callable from `MessageHasBeenPosted` (confirmed by researcher — no HTTP-request-cycle dependency) but is an unpersisted, unconfirmed websocket broadcast that silently drops for disconnected/offline users — a real reliability downgrade from the old `CommandResponse`-body ephemeral. Consilium (architect+skeptic) recommends threaded non-ephemeral error replies instead; the task brief itself pre-authorized this fallback ("if not cleanly usable... fall back... note the UX change"), so this is resolved without escalation.

## Blockers (HIGH, requires_human)

1. **Mention-extraction scope** (architect HIGH/high, skeptic HIGH/high) — leading-token-only vs. the brief's originally-specified anywhere-in-message. Escalated to human, **RESOLVED 2026-08-25: Option B — leading-or-trailing only.** `@math-bot` must be the first OR last token of the trimmed message. `@math-bot 2+2` and `2+2 @math-bot` both trigger; `what is @math-bot 2+2` (mid-message) does NOT trigger — treated as a no-op (silent, no mention match), not an error. Rationale (architect+skeptic repro): stripping a mid-message mention produces silently-wrong results (`sqrt @math-bot (4)` → strips to `sqrt  (4)` → evaluates to `2`, a result never intended) or parse failures (`2+2 @math-bot 3+3` → `2+2  3+3`) — narrowing to the two unambiguous anchor positions eliminates both failure modes while still covering the "@bot at the start" and "@bot tacked on at the end" idioms. To be recorded verbatim in `.assistant/decisions.md` D-005.
2. **New decision entry required** (reviewer HIGH/high, architect MEDIUM/high) — a `D-005` append-only entry superseding the trigger/transport portions of D-003/D-004 is mandatory before/at `/post-feature`; D-002/D-003's eval/normalization/percent/log-base decisions stay untouched. Not a blocker to implementation start, but a hard gate before commit/close-out.
3. **Server-wide hook scope** (skeptic HIGH/high, researcher corroborated) — `MessageHasBeenPosted` has no channel/team/membership filter; this is inherent to the hook mechanism the brief specified, not an implementation choice. Recorded as an accepted trade-off in D-005, not blocking, but surfaced to the human for awareness since it's a real expansion of what messages the plugin process observes.
4. **Raw expression/message-text logging** (skeptic HIGH/high) — the old `c.client.Log.Debug("math command evaluation failed", "expression", expression, ...)` pattern, if carried into the hook, would write private-channel/DM message text into server logs on every failed evaluation (now the common case since incidental mentions are possible). Resolution is unambiguous (drop the field or log length/hash only) — implemented directly, not escalated.

## Concerns (MEDIUM)

- Empty-remainder-after-mention (bare `@math-bot`): architect recommends silence; brief originally specified a usage-hint reply. Resolved in favor of the brief's explicit instruction (usage-hint reply) unless the human's answer to Blocker 1 changes the extraction model enough to warrant revisiting.
- `server/command/` deletion cascades: `Makefile` `mock:` target (mockgen against `server/command Command`), `go.uber.org/mock` dependency (orphaned, drop via `go mod tidy`), `server/command/mocks/`.
- `ExecuteCommand`, `commandClient` field, `net/http` import in `server/plugin.go` all become dead once `/math` dispatch is removed.
- Test coverage migration: `mathErrorMessage`'s 6-branch + default mapping and post-shape assertions must be relocated to `server/hooks_test.go`, not just deleted with the rest of `command_test.go`.
- Bot `Description` field (`"Posts results for the /math slash command."`) propagates to already-installed servers on next `EnsureBot` activation — must be updated as part of this change, not left stale.
- No official Mattermost mention-parsing helper exists in `server/public@v0.1.21` — hand-rolled matcher required; official reference plugins (`mattermost-plugin-demo`, `mattermost-plugin-nps`) confirm hand-rolling is idiomatic, but the demo's own `strings.Contains` example is looser than needed (matches `@math-bot2`) — do not copy it verbatim.
- `pluginapi.PostService.ShouldProcessMessage` (official anti-loop helper, already in the pinned dependency) has a gotcha: its default bot-ID resolution reads a KV key only populated by `pluginapi.Client.Bot.EnsureBot` (which this plugin already uses per D-004) — safe to rely on the default, but pass `pluginapi.BotID(p.botUserID)` explicitly rather than trusting the KV-lookup path, per researcher's finding.

## Notes (LOW)

- `plugin.json`'s `min_server_version` needs no bump — `MessageHasBeenPosted` and `ShouldProcessMessage` both document a 5.2 floor, well under the existing 7.1.0 (owned by `EnsureBot`, per D-004/OQ-003). Confirmed independently by architect and researcher.
- No `plugin.json` capability/manifest key exists for this hook — implementing the Go method is the entire registration (researcher, medium confidence, corroborated by architect).
- `RootId` logic: must be `post.RootId` if non-empty else `post.Id` — an earlier assumption that `post.Id` unconditionally was safe is wrong; the server rejects a reply whose root is itself a reply (400, confirmed against pinned server source by both architect and skeptic).
- Bot posting into channels/DMs it isn't a member of is unchanged-in-kind from D-004 (plugin-authored `CreatePost` already bypassed membership checks for `/math`) — not new risk, just extended to a new trigger surface.
- DM-only interaction (messaging `@math-bot` directly in its own DM channel with no literal mention text) does not work under a mention-based trigger — accepted as out-of-scope, matches the task brief's own examples which are all channel messages with an explicit mention.
- Message edits do not re-trigger `MessageHasBeenPosted` (that's `MessageWillBeUpdated`/`MessageHasBeenUpdated`) — accepted as a documented scope note, not implemented.

## Research findings (researcher, all sourced/dated 2026-08-25 unless noted)

- `MessageHasBeenPosted` doc-annotated "Minimum server version: 5.2" (`server/public@v0.1.21/plugin/hooks.go`). confidence: high.
- Hook registration is pure Go-method-presence via RPC `Implemented()` reflection — no manifest key. confidence: medium.
- Hook fires via `a.Srv().Go(...RunMultiHook...)` post-commit, no channel/team/membership filter anywhere in the call path. confidence: medium.
- Hook contract explicitly warns it fires for the plugin's own posts too. confidence: high.
- `rpost.ForPlugin()` clears `Metadata` before handing the post to the hook (MM-51090, merged 2023-03-15). confidence: high.
- Mattermost core maintainer (GitHub issue #21573, 2022-12-01): out-of-channel mentions are invisible to notifications but plugin hooks (`MessageWillBePosted`/`MessageHasBeenPosted`) still fire — corroborated by the RunMultiHook call-site finding. confidence: corroborated.
- Out-of-channel @mentions never auto-join the mentioned account; bots are filtered out of the out-of-channel-mention ephemeral prompt entirely (`FilterWithoutBots`, issue #11329 closed 2019-07-18). confidence: high.
- `PluginAPI.CreatePost` → `CreatePostMissingChannel`, no membership check at that layer; stamps `PostPropsFromPlugin="true"` on every plugin-created post. confidence: medium.
- No mention-parsing helper exists in `server/public@v0.1.21` (grepped `AtMention`/`ContainsMention`/`MentionPattern`/`PossibleAt` — zero hits). confidence: medium.
- Server's own (unexported, unimportable) mention regex: `\B@[[:alnum:]][[:alnum:]\.\-_:]*` (`channels/app/command.go`) — matches trailing `.`/`-`/`_`/`:`, must be trimmed. confidence: medium.
- `mattermost-plugin-demo` hand-rolls via `strings.Contains(post.Message, "@"+username)` — looser than ideal (matches `@math-bot2`). confidence: medium.
- `pluginapi.PostService.ShouldProcessMessage` is the official anti-loop helper (min version 5.2, already in pinned dep) — covers self-ID, system messages, webhook posts, other-bot authorship; gotcha on default bot-ID KV resolution (`mmi_botid`, only populated by `pluginapi.Client.Bot.EnsureBot`). confidence: high / medium.
- Legacy `plugin.Helpers.ShouldProcessMessage` was removed in v6.0 — only the `pluginapi` package version exists now. confidence: corroborated.
- Reference plugins (`mattermost-plugin-demo`, `mattermost-plugin-nps`) both guard with `post.UserId == p.botID` first, then `IsSystemMessage()`, then other-bot/webhook checks. confidence: medium.
- `SendEphemeralPost` structurally callable from `MessageHasBeenPosted` (no HTTP-request-cycle dependency, `api.ctx` is plugin-scoped not per-request) — confidence: medium. But delivery is a pure unpersisted websocket broadcast with no error return, silently dropped if the target has no live session at that instant — confidence: medium, and this caveat applies equally to `ExecuteCommand`, not hook-specific.
- No corroborated GitHub issue found describing an ordering/timing race for ephemeral posts from async hooks — absence of evidence, not evidence of absence (unverified).
- `UpdateEphemeralPost`/`DeleteEphemeralPost` are explicitly marked EXPERIMENTAL — avoid for PROJECT_TYPE 2. confidence: medium.

## Out-of-scope (declared)

- `server/mathexpr/*` — reused unchanged, D-002/D-003 baseline stands.
- Channel/team admin opt-out or scoping toggle for the mention hook.
- `MessageHasBeenUpdated` (edit re-triggering).
- DM-only (no-mention) interaction mode.
- Streaming/progressive ephemeral UX via `UpdateEphemeralPost` (experimental API).

## Open questions raised

- Should OQ-003 (resolved, `min_server_version` floor) get an appended note confirming 7.1.0 still holds under the new hook (no reopen, just an addendum)? — reviewer, LOW, non-blocking.

## Per-agent verbatim sections

### architect (retry — first attempt failed with an API error mid-run, re-spawned with a tighter, more concise prompt)

See findings integrated above; full YAML preserved in this session's task-notification history (module-boundary: delete `server/command/` entirely, no repurposing; fold `mathErrorMessage` into `server/hooks.go`; leading-token-only extraction; guard-chain ordering cheap-checks-first; RootId fallback rule; threaded non-ephemeral errors; silence-on-bare-mention; no mention-parsing helper in the SDK — hand-roll a prefix+boundary check, not regexp; Makefile mockgen target removal + `go mod tidy`; dead-code removal in `server/plugin.go`; test relocation plan; plugin.json/README description updates; no min_server_version bump needed; bot Description string update; accept server-wide/no-membership posting as unchanged-in-kind from D-004; recommends a written ADR).

### skeptic

See findings integrated above; full YAML preserved in this session's task-notification history (scope-creep flag on removing `/math` without a fresh, dedicated human sign-off distinct from D-004's already-consumed quote; server-wide hook scope with no membership filter, sourced directly against pinned server code; raw-expression logging privacy regression; guard chain must cover `from_plugin`/`from_webhook`/`from_bot`/system messages, not just `UserId==botUserID`; RootId-must-not-be-post.Id-unconditionally with server-side 400 evidence; substring-mention false positives on `@math-bot2` etc.; mid-message-strip ambiguity with concrete before/after examples; ephemeral-from-hook best-effort/unpersisted delivery caveat; dead DM path; orphaned test coverage on `server/command` deletion; min_server_version correctly left unchanged; RPC-volume/no-admin-toggle cost note).

### researcher

See findings integrated above; full YAML preserved in this session's task-notification history (26 sourced, dated, confidence-flagged findings covering hook registration mechanics, server-wide firing scope, out-of-channel-mention behavior, absence of an official mention-parser, `ShouldProcessMessage` as the correct anti-loop primitive with its KV-based bot-ID gotcha, `SendEphemeralPost` structural-callability-vs-delivery-reliability distinction, and two official reference-plugin guard-chain examples).

### reviewer

See findings integrated above; full YAML preserved in this session's task-notification history (D-005 requirement against INVARIANT §7/§8; `post.RootId`-vs-`post.Id` factual correction against D-004's pattern; ANTI-4 human-approval requirement flagged MEDIUM; bot-self-reentrancy guard requirement; error-transport-undefined-in-proposal flag against D-003's six-branch `mathErrorMessage`; ten-location doc-staleness inventory including `AGENTS.md` correction — task brief's claim that `AGENTS.md` mentions `/math` is factually wrong, it does not, `CLAUDE.md`/`plugin.json` do instead; append-only-log discipline reminder; server-wide scope flagged as undocumented trade-off, not an ANTI-9 violation; bot `Description` staleness; `Makefile` mockgen breakage; OQ-003 addendum suggestion, no reopen).
