# Plan: `/math` success reply as a dedicated bot post

**Status:** consilium-complete
**Date:** 2026-08-25
**Consilium:** architect (opus), skeptic (opus), researcher (sonnet), reviewer (sonnet) — 4/4 returned, no re-spawns needed.

## Task container

- **slug:** `math-bot-post-reply`
- **scope-fence:** `server/plugin.go` (OnActivate bot registration, avatar set), `server/command/command.go` (Handler struct, NewCommandHandler signature, executeMathCommand success path), `server/command/command_test.go` (test rewrite), `plugin.json` (min_server_version bump), `assets/math-bot-icon.png` + `assets/math-bot-icon.svg` (git-add, currently untracked), `.assistant/decisions.md` (new D-004 entry, append-only, written after implementation+E2E per D-003 precedent). Everything else out of scope.
- **definition-of-done:**
  1. Plugin registers a bot account (`math-bot` / "Math Bot") idempotently in `OnActivate`, fails activation loudly if bot creation itself fails, logs-and-continues if only the avatar set fails.
  2. Bot's avatar is set from `assets/math-bot-icon.png` via the bundle-path-read pattern.
  3. `/math` success path posts `` `<expr>` = `<result>` `` as a new post authored by the bot (`UserId` = bot id, `ChannelId`/`RootId` from `args`), and returns a non-nil, all-zero `model.CommandResponse{}` (not `nil`, not the old `Text`/`ResponseType`).
  4. If `CreatePost` fails, user gets an ephemeral error instead of silence.
  5. Error path (compile/runtime/etc. failures) is unchanged: ephemeral `CommandResponse`.
  6. `server/command/command_test.go` rewritten and green: constructor signature change reflected at all 4 call sites, `CreatePost` mock assertions on `UserId`/`ChannelId`/`RootId`/`Message`, response-emptiness assertion.
  7. `plugin.json` `min_server_version` bumped `6.2.1` → `7.1` (see researcher finding on `EnsureBotUser`'s documented floor).
  8. Assets committed (currently untracked).
  9. `make deploy` against the local test Mattermost server (`MM_SERVICESETTINGS_SITEURL=http://localhost:8065 MM_ADMIN_TOKEN=...`) succeeds; live `/math 50% + 10` (and 2-3 more cases) show a reply from "Math Bot" with the new avatar, not from the invoking user's identity.
  10. New `.assistant/decisions.md` D-004 entry recorded after verification, documenting the D-003 supersession and the open scope notes below.
- **verification-contract:** `go build ./...`, `go vet ./...`, `go test ./...` (or project's `make test`/`make check` if defined — check `Makefile`) all green; live E2E per DoD item 9.
- **inputs-needed:** `MM_ADMIN_TOKEN` env var for local deploy verification (already provided out-of-band to the orchestrator, never persisted to any file).
- **out-of-scope:** DM/group-DM channel special-casing (see Notes), un-swallowing the user's own slash-command text (confirmed platform-level, not fixable), bot cleanup on `OnDeactivate`, normalizing the echoed expression string (keep raw/trimmed, matching current behavior exactly).

## TL;DR

- 4/4 consilium reports returned, all substantive (no `consilium-found-nothing`).
- Severity counts (post-dedup): **7 HIGH**, **9 MEDIUM**, **5 LOW**, plus 14 researcher findings (fact-finding, not severity-tagged).
- Top 3 must-fix: (1) `pluginapi.Client.Bot.EnsureBot(bot, ProfileImagePath(...))` fails the *entire* bot-ensure call if avatar-set fails — must split into two calls to make avatar failure non-fatal as required. (2) Success path must return a non-nil, all-zero `&model.CommandResponse{}` — returning `nil` breaks the command (server falls through to a 404 "not found" path); returning the old `Text`/`ResponseType` double-posts. (3) `CreatePost` failure and empty `botUserID` need an explicit ephemeral-error fallback — an empty response on failure would silently swallow the result (ANTI-11, "no error masking").
- All HIGH findings resolved to concrete engineering decisions below (Decisions Locked). None left as an unresolved blocker requiring a stop-and-ask — see "Decisions made without further human input" for the reasoning on each.

## Decisions locked (feed directly into `/implementor`)

1. **Two-phase bot creation.** `client.Bot.EnsureBot(&model.Bot{Username: "math-bot", DisplayName: "Math Bot", Description: "Posts results for the /math slash command."})` with **no** `ProfileImagePath`/`ProfileImageBytes` option. If this call errors, `OnActivate` returns the wrapped error (fails activation) — consistent with this file's existing precedent of failing hard on `cluster.Schedule` error (server/plugin.go:61-63). Separately, read `assets/math-bot-icon.png` via `client.System.GetBundlePath()` + `os.ReadFile` (not `ioutil.ReadFile`, deprecated since Go 1.16, module is `go 1.25`) and call `client.User.SetProfileImage(botUserID, bytes.NewReader(data))`; on error, `client.Log.Warn` and continue — activation must not fail on avatar failure alone.
2. **Wiring order.** Insert bot creation between `p.client = pluginapi.NewClient(...)` and `p.commandClient = command.NewCommandHandler(...)` in `OnActivate` (server/plugin.go:47-51). Add `botUserID string` field to the `Plugin` struct (server/plugin.go:20-43). Change `command.NewCommandHandler(client *pluginapi.Client) Command` to `NewCommandHandler(client *pluginapi.Client, botUserID string) Command`; add matching `botUserID string` field to `Handler`.
3. **Success-path post.** `executeMathCommand` builds `&model.Post{UserId: c.botUserID, ChannelId: args.ChannelId, RootId: args.RootId, Message: fmt.Sprintf("`%s` = `%s`", expression, mathexpr.FormatResult(result))}` — `expression` stays the raw/trimmed string exactly as used today (do NOT switch to the normalized form; the current implementation already echoes raw/trimmed, and D-003 never surfaced the normalized string — switching would be an unrequested behavior change). Post via `c.client.Post.CreatePost(post)`. On error: `c.client.Log.Error(...)`, return `&model.CommandResponse{ResponseType: model.CommandResponseTypeEphemeral, Text: "Could not post the result."}`. On success: return `&model.CommandResponse{}` (non-nil, all zero fields — verified against server v8 `channels/app/command.go` that this reaches `CreateCommandPost`'s early-return guard and posts nothing extra, whereas `nil` triggers a 404 fallback).
4. **Error path.** Unchanged — ephemeral `CommandResponse`, same as today.
5. **RootId threading.** Always set `RootId: args.RootId` on the bot's post (empty string is a safe no-op for a bare channel-box invocation; non-empty preserves in-thread reply behavior the server already provides today for the `CommandResponseTypeInChannel` path — this is behavior-preservation, not new scope).
6. **`plugin.json`.** Bump `min_server_version` from `"6.2.1"` to `"7.1"` — the pinned `server/public@v0.1.21` vendored source documents `EnsureBotUser`'s minimum server version as 7.1 (pluginapi's own internal soft-check of 5.10.0 is an inconsistency within the same dependency version, not authoritative). No bot-related manifest field exists or is needed elsewhere in `model.Manifest`.
7. **No channel-membership workaround.** Do not call `AddChannelMember`/`AddUserToChannel` before `CreatePost` — confirmed via two independent source-code paths that plugin-authored `CreatePost` bypasses channel-membership checks entirely. Adding a join step would be unrequested scope and would emit spurious join system-messages.
8. **Tests.** Rewrite `server/command/command_test.go`: all 4 `NewCommandHandler(env.client)` call sites (lines 45, 68, 98, 113) get a fake bot id argument. `TestMathCommandSuccess` needs `env.api.On("CreatePost", mock.MatchedBy(...))` asserting `UserId`/`ChannelId`/`RootId`/`Message`, and must assert the returned `CommandResponse` is empty (not the old `Text`/`ResponseType`). `TestMathCommandErrors`/`TestUnknownCommand` only need the signature update. Do **not** add an `OnActivate` unit test — `EnsureBot` internally calls `ensureServerVersion` which panics on `plugintest.API`'s zero-value `GetServerVersion()`; covering it would need mocking `GetServerVersion`, `KVSetWithOptions`, `EnsureBotUser`, `GetBundlePath`, `SetProfileImage` — track as a pre-existing test-infra gap, not a blocker for this feature.
9. **Assets.** `git add assets/math-bot-icon.png assets/math-bot-icon.svg` as part of this feature's commit — both are currently untracked (verified via `git status --short assets/`); ship them as part of this change so the D-004 entry has a real commit to point at.

## Blockers (HIGH, requires_human — resolved below, none left open)

- **Bot avatar asset provenance** (reviewer, HIGH): assets exist on disk, untracked, with no prior `.assistant/decisions.md`/`open-questions.md` entry recording approval — the only "approval" is the direct instruction given to this task in the current session. **Resolution:** treat as approved (the instruction is explicit and unambiguous: "already designed and approved this session... do not regenerate"); fix the paper-trail gap by committing the files and recording provenance in the new D-004 entry. Not re-asking — re-litigating an explicit direct instruction would be busywork, not diligence.
- **D-003 reversal** (reviewer, HIGH): plan reverses D-003's deliberate `CommandResponseTypeInChannel` convention. **Resolution:** this reversal *is* the user's literal request (their complaint is exactly "don't like that the message gets replaced... bot replies instead") — proceeding is correct; D-004 will document the supersession with rationale, per INVARIANT §8 (append-only, never edit D-003).
- **User's own text still never appears** (skeptic, HIGH, premise-flaw): researcher settled this as a confirmed platform fact — Mattermost slash commands execute as a request, never post the raw command text as a channel message, for any plugin, ever (docs.mattermost.com/integrations-guide/slash-commands.html: "it will execute without posting a message"). **Resolution:** not ambiguous anymore, no need to ask — this was already covered by the task brief's own explicit instruction not to hack around it. Will surface as a plain fact in the final report, not a blocking question.
- **OnActivate fail-loud vs. log-and-continue for bot-ensure itself** (skeptic, HIGH): resolved as **fail loud** (Decision 1 above) — matches this file's own local precedent (`cluster.Schedule` error already fails activation), and is the more conservative choice for a Type-2 production plugin (no silent non-responsiveness).
- **`CreatePost`/empty-`botUserID` failure silently swallowing the result** (skeptic, HIGH): resolved via the ephemeral-fallback in Decision 3.

## Concerns (MEDIUM) — accepted as documented trade-offs, not blocking

- **DM/group-DM bot posting** (architect, skeptic): bot has no special-casing per channel type — it posts into whatever `ChannelId` the command ran in, same as today's `CommandResponseTypeInChannel` already does (today's post is just self-authored instead of bot-authored). Restricting by channel type would be unrequested complexity (AGENTS.md "don't do more than asked"). Documented as an accepted scope note for D-004, not a blocker.
- **Attribution loss** (skeptic): once the bot always authors the result, it's less immediately obvious *who* ran `/math` in a busy channel (today's post was self-authored, making the requester visually obvious). Direct consequence of the user's own request; documented as an accepted cost.
- **Username collision risk** (skeptic): mitigated by using the namespaced `math-bot` username rather than the more collision-prone bare `math`.
- **Bot account permanence** (skeptic): `EnsureBot` creates a persistent user row with no `OnDeactivate` cleanup — intentional (deleting it would orphan historical result posts); documented, not fixed.
- **Unread-badge/notification change** (skeptic): invoking user now gets a notification for their own result post since it's bot-authored, not self-authored. Documented, not mitigated (no clean API lever without adding complexity out of scope).

## Notes (LOW)

- README.md:170-188's `ioutil.ReadFile` + raw `p.API.SetProfileImage` example is stylistically inconsistent with this repo's existing 100%-`pluginapi.Client` convention (server/plugin.go, server/command/command.go never touch `p.API` directly) — use `client.System.GetBundlePath()` + `os.ReadFile` + `client.User.SetProfileImage(userID, io.Reader)` instead, per architect finding.
- No `plugin.json` manifest field exists for bot declaration in `model.Manifest` (v0.1.21) — confirmed by both direct struct inspection and official docs (2 independent sources, corroborated).

## Research findings (researcher, confidence-flagged)

- `pluginapi.Client.Bot.EnsureBot(bot *model.Bot, options ...pluginapi.EnsureBotOption) (string, error)` wraps raw `plugin.API.EnsureBotUser`. Options: `pluginapi.ProfileImagePath(path)`, `pluginapi.ProfileImageBytes(bytes)`. — source: github.com/mattermost/mattermost/blob/server/public/v0.1.21/server/public/pluginapi/bot.go#L141-L206, 2025-10-15, confidence: medium.
- Raw `EnsureBotUser` doc comment says "Minimum server version: 7.1" in the same pinned v0.1.21 source where `pluginapi`'s own soft-check requires only 5.10.0 — internal inconsistency, flagged for explicit resolution. — source: github.com/mattermost/mattermost/blob/server/public/v0.1.21/server/public/plugin/api.go#L1189-L1192 and .../pluginapi/bot.go#L161-L166, 2025-10-15, confidence: medium.
- `model.Manifest` (v0.1.21) has no bot-declaration field — corroborated by struct inspection + developers.mattermost.com/integrate/reference/bot-accounts/. confidence: corroborated.
- `plugin.API.CreatePost` bypasses channel-membership checks entirely (2 independent source-code paths: `channels/app/post.go` has no membership check; contrast with `channels/api4/command.go`'s session-layer `SessionHasPermissionToChannel` check, which plugin calls never traverse). confidence: medium. Caveat: file uploads by a bot DO require channel membership (official docs), irrelevant to plain-text replies.
- Slash-command text is never posted as a channel message by the platform, for any plugin — docs.mattermost.com/integrations-guide/slash-commands.html. confidence: medium.
- `model.CommandArgs.RootId` is client-controlled (decoded verbatim from the client's request body), not server-derived; inferred (not directly verified against mattermost-webapp source) to be empty for a bare `/math` typed in the main channel box. confidence: unverified for the "empty for bare commands" inference specifically — treat `RootId: args.RootId` as a safe pass-through regardless, since it degrades to a no-op if actually always empty in this context.

## Out-of-scope (declared)

- Un-swallowing the user's own `/math ...` invocation text — confirmed unfixable, platform-level, universal to all Mattermost slash commands.
- DM/group-DM channel-type special-casing.
- Bot account cleanup on `OnDeactivate`/uninstall.
- Normalizing the echoed expression string in the bot's reply (stays raw/trimmed, matching current behavior).
- Any change to `server/mathexpr` (evaluation/normalization/formatting logic untouched).

## Open questions raised (candidates for `.assistant/open-questions.md` if not resolved by E2E)

- Does the local test Mattermost server actually reject `EnsureBotUser` below server 7.1, or is the doc-comment stale? Resolve empirically during the `make deploy` E2E step rather than guessing further.

## Per-agent verbatim sections

### Architect (opus) — 12 findings, 6 HIGH / 3 MEDIUM / 3 LOW — see decisions above; full YAML in agent transcript, not reproduced verbatim here per terseness (all HIGH/MEDIUM items are reflected in Decisions Locked / Concerns / Notes sections above with attribution).

### Skeptic (opus) — 12 findings, 4 HIGH / 5 MEDIUM / 3 LOW — reflected above.

### Researcher (sonnet) — 14 findings, all fact-finding with confidence flags — reflected in Research findings above.

### Reviewer (sonnet) — 4 findings, 2 HIGH / 2 MEDIUM — reflected above.
