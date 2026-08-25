# /math slash command — implementation report

Status: complete

## Layers executed

- Single `backend` layer (Go only, no webapp changes), one exec agent (subagent_type `backend` unavailable in this runtime — ran as `general-purpose` carrying the `backend` role contract inline). ~15 min wall time.
- Orchestrator verification pass: re-ran `go build`/`go vet`/`go test` independently, read every changed/new file, deployed to a live local Mattermost instance, invoked `/math` for 6 real cases via the REST API.

## Files touched

| Path | Action | Notes |
|---|---|---|
| `server/mathexpr/errors.go` | created | 6 sentinel errors |
| `server/mathexpr/normalize.go` | created | 400 lines — fixed-order pipeline, hand-written rune scanners (RE2 has no lookaround/backreferences) |
| `server/mathexpr/normalize_test.go` | created | table-driven, per-substep coverage |
| `server/mathexpr/eval.go` | created | 57 lines — `expr.Compile`/`Run` wiring, degrees conversion, log base-10 |
| `server/mathexpr/eval_test.go` | created | full required test matrix |
| `server/mathexpr/format.go` | created | 22 lines — epsilon-snap formatting |
| `server/mathexpr/format_test.go` | created | sin/cos degree edge cases |
| `server/command/command.go` | modified | `hello` removed entirely; `/math` registered; sentinel→fixed-string error mapping |
| `server/command/command_test.go` | modified | `TestHelloCommand` replaced with math command tests |
| `server/command/mocks/mock_commands.go` | modified | regenerated via mockgen after `Command` interface narrowed to `Handle`-only |
| `go.mod` / `go.sum` | modified | `github.com/expr-lang/expr v1.17.8` added (satisfies D-002's `>=v1.17.7`) |

## Verify results

| Command | Result |
|---|---|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `go test ./... -v` | pass, 0 failures (`server`, `server/command`, `server/mathexpr`, `build/pluginctl`) |
| `golangci-lint run ./...` (v2.9.0) | pass, 0 issues |

Re-verified independently by the orchestrator (not just trusting the exec agent's report): `go build ./... && go vet ./... && go test ./...` — all green.

## Manual E2E (live server — per ANTI-11, unit tests passing is not "done")

Deployed to the coordinator-provided local Mattermost instance (`http://localhost:8065`) via `make deploy` with `MM_SERVICESETTINGS_SITEURL`, `MM_ADMIN_TOKEN` as local env vars (never written to any file). Note: the default local-mode Unix socket (`/var/tmp/mattermost_local.socket`) on this machine appears shared/stale across concurrent Mattermost dev-server processes for other projects — deploy had to force `MM_LOCALSOCKETPATH` to a nonexistent path to make `pluginctl` fall back to token-based HTTP auth against the actual target server, otherwise it silently talked to the wrong server instance and got a false "uploads disabled" error.

Invoked `/math` via `POST /api/v4/commands/execute` against a live channel:

| Input | Response type | Text |
|---|---|---|
| `2 x 2` | in_channel | `` `2 x 2` = `4` `` |
| `2(3+4)` | in_channel | `` `2(3+4)` = `14` `` |
| `sin(90)` | in_channel | `` `sin(90)` = `1` `` (degrees convention confirmed) |
| `50% + 10` | in_channel | `` `50% + 10` = `10.5` `` (percent-rewrite confirmed) |
| `2 +` | ephemeral | `Could not parse expression.` (no raw Go/expr error leaked) |
| `1/0` | ephemeral | `Result is not a finite number (e.g. division by zero).` |

All 6 match the spec exactly.

## Out-of-scope (declared, carried from plan)

- Webapp changes — none made.
- `NewCommandHandler` returning `(Command, error)` for loud activation failure on registration error — left as-is, matching the pre-existing `hello` convention (log-and-continue), not introduced as a new pattern here.
- `DisableAllBuiltins()`/`EnableBuiltin` — superseded by the character-allowlist mechanism.

## Open issues raised during implementation

- None blocking. One documented, intentional scope note from the exec agent: the percent-rewrite's function-name lookback (`findPercentOperandStart`) matches only `sqrt`/`sin`/`cos`/`log` (not `abs`), per the plan's explicit list for that specific sub-rule, while the allowlist and implicit-multiplication rules match all five including `abs`. This means `abs(-5)%` is not specially recognized as a function-call operand by the percent scanner (it would fall through to the paren-balancing branch anyway since `abs(-5)` ends in `)`, so behavior is actually unaffected — verified by reading `findPercentParenOperandStart`, which handles any `)`-terminated operand generically and only uses `percentFuncNames` to decide whether to also absorb the function name itself into the rewritten span). Not a functional gap, no test regression.
- int64 arithmetic can still overflow before `expr.AsFloat64()`'s final conversion (e.g. very large chained multiplications) — accepted, documented in the plan as a known v1 limitation, not fixed (contrived input, avoids scope creep).

## Suggested commit message

```
Add /math slash command with expr-lang evaluation

Implements /math <expression>: a normalization pass (unicode symbols,
comma decimals, implicit multiplication, percent-as-division-by-100)
feeds github.com/expr-lang/expr, with sqrt/sin/cos/log registered as
custom functions (trig in degrees, log base-10, per D-002). Replaces
the leftover starter-template `hello` command.
```

## Next

`/post-feature math-slash-command` — append D-003 (implementation decisions: package boundary, percent semantics, log base, length cap value), close any related open questions, update `.memory-bank/tech-details/stack.md` TODO on command-wiring conventions, draft commit + PR text.
