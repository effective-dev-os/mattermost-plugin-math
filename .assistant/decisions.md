# Decisions Log

> Append-only chronological record. When a decision is overturned, add a new entry with date + reason. Never edit or delete prior entries.

---

## D-001 — Harness installed
**Date:** 2026-08-25
**Status:** accepted
**Decision:** Effective Harness installed at commit `ba679efe5a4ffffd8a90a2c156c68856871fde27` via the `/setup` skill. `PROJECT_TYPE: 2`. Primary stack: backend (Go), web frontend (TypeScript/React). Module installed: maintenance.
**Source:** `git@github.com:effective-dev-os/harness.git@ba679efe5a4ffffd8a90a2c156c68856871fde27`
**Touch policy chosen at install:** overwrite (no existing CLAUDE.md was present)

## D-002 — Expression evaluation engine: expr-lang/expr
**Date:** 2026-08-25
**Status:** accepted
**Decision:** `/math` will evaluate arithmetic via `github.com/expr-lang/expr`, pinned `>=v1.17.7` (covers both GHSA-93mq-9ffx-83m2/CVE-2025-29786, fixed 1.17.0, and GHSA-cfpf-hrx2-8rv6/CVE-2025-68156/GO-2025-4245, fixed 1.17.7 — do not pin `>=v1.17.0` alone, that leaves the second advisory open). Pin the canonical module path `github.com/expr-lang/expr`, not the legacy `github.com/antonmedv/expr` alias. `^`/`**` mean exponent in expr-lang (not XOR), no gotcha there. `abs` ships as an expr-lang builtin — do not re-register it. `sqrt`, `sin`, `cos`, `log` must be registered manually via `expr.Function`. `sin`/`cos` accept **degrees** (user-facing decision — convert to radians internally before calling `math.Sin`/`math.Cos`), per human confirmation, since chat users expect `sin(90) = 1` not radians.

The plugin must run a text-normalization pass on the raw slash-command string before `expr.Compile`, since no candidate library natively parses human notation. Minimum normalization set (test matrix required before merge, per skeptic critique): `×`/digit-adjacent `x` → `*`; `÷` → `/`; unicode minus (U+2212) and dash variants (U+2013 en dash, U+2014 em dash) → `-`; comma-between-digits → `.` (decimal, not thousands-separator — chat input is assumed single-number, no thousands grouping); implicit multiplication for digit-before-paren (`2(3+4)`), paren-before-digit (`(3+4)2`), paren-before-paren (`(2+3)(4+5)`), and digit-before-function-name (`2sqrt(4)`) → insert `*`; trailing/postfix `%` → rewritten as `/100` on the preceding numeric subexpression (mid-expression `%` behavior must be explicitly decided during implementation — support or reject-with-clear-error, not left ambiguous). Add a hard input-length cap before `expr.Compile` as defense-in-depth alongside the version pin.
**Source:** `swarm-report/research-go-expr-library-2026-08-25.md` (researcher + skeptic + reviewer consilium). Human-approved: library adoption + degrees convention confirmed via AskUserQuestion, 2026-08-25.
**Resolves:** OQ-002.

## D-003 — `/math` implemented: normalization pipeline, percent semantics, log base, package boundary
**Date:** 2026-08-25
**Status:** accepted
**Decision:** `/math <expression>` shipped in `server/command/command.go` (leftover starter-template `hello` command removed entirely — `plugin.json` already scoped this plugin to `/math` only). Normalization + evaluation live in a new package `server/mathexpr` (zero Mattermost imports, pure functions, independently unit-tested), consumed by `server/command` as a service layer. Normalization runs as a fixed-order pipeline: unicode-symbol→ASCII, comma-decimal, percent-rewrite, implicit-multiplication, then a character/identifier allowlist as the final validation step (chosen over `expr.DisableAllBuiltins()` as the single mechanism keeping expr-lang's other ~60 builtins and string/array/pipe/range/comparison syntax unreachable). `expr.Compile` uses `expr.Env(map[string]any{})` (clean compile errors on unknown identifiers) and `expr.AsFloat64()` (uniform float64 result). Dependency pinned at `github.com/expr-lang/expr v1.17.8` (satisfies D-002's `>=v1.17.7` floor; confirmed via OSV/GHSA cross-check that no new CVE was disclosed as of 2026-08-25).

Two decisions not settled by D-002 were resolved during `/pre-feature` by direct analogy to D-002's own degrees precedent, since `AskUserQuestion` was unavailable in this run's toolset and both are documented here for human override:
- **Percent semantics:** every `%` — trailing or mid-expression — is rewritten to `(<preceding operand>/100)`, uniformly, regardless of position. This removes expr-lang's integer-modulo operator from user-facing input entirely (e.g. `10%3` evaluates as `(10/100)*3 = 0.3`, not `10 mod 3 = 1`). Matches the task brief's own "recommended" default and composes cleanly with the implicit-multiplication pass (no special-casing needed).
- **`log()` base:** base-10 (`math.Log10`), not natural log — same "calculator convention over programming-language convention" reasoning D-002 used for `sin`/`cos` degrees (chat users expect `log(100) = 2`).

Other implementation-level decisions: hard input cap of 1024 runes on the raw expression (measured via `utf8.RuneCountInString`, before normalization — matches Mattermost's own `Command.URL` length bound; Mattermost enforces no length cap on slash-command text server-side, so this is not redundant, confirmed against `server/public` v0.1.21 + `api4/command.go`). Division/modulo-by-zero: expr-lang's `/` never errors on zero (`1/0` → `+Inf`, `0/0` → `NaN`, both as *successful* results) — checked explicitly post-`Run` via `math.IsInf`/`math.IsNaN`, mapped to a fixed ephemeral error message. Success responses use `model.CommandResponseTypeInChannel` explicitly (the pre-existing `hello` pattern left `ResponseType` unset, which the Mattermost server treats as ephemeral — not copied). Results formatted via `FormatResult` (epsilon-snap to nearest integer within `1e-9`, `strconv.FormatFloat` with `'g'`/12 precision) so degree-based trig prints clean (`sin(180)` → `"0"`, not `"1.2246467991473515e-16"`).

**Alternatives rejected:**
- `expr.DisableAllBuiltins()` + `expr.EnableBuiltin("abs")` as a second hardening layer — redundant given the character-allowlist already blocks every identifier outside `{sqrt, sin, cos, log, abs}`; not adding two overlapping defense mechanisms.
- `NewCommandHandler` returning `(Command, error)` so a registration failure fails plugin activation loudly — left as the pre-existing `hello`-era convention (log-and-continue) to avoid an unrelated refactor; noted as a pre-existing gap, not introduced or fixed by this feature.
- Natural log for `log()` — rejected in favor of base-10 per the calculator-convention precedent above.
- Keeping `%` as modulo in binary-infix position and percent only when strictly trailing — rejected in favor of one uniform rule (simpler, matches the task brief's recommendation, composes with implicit-multiplication without special-casing).
**Source:** `swarm-report/math-slash-command-plan-2026-08-25.md` (architect + skeptic + researcher + reviewer consilium), `swarm-report/math-slash-command-implementation-2026-08-25.md` (verified independently by the orchestrator: re-ran build/vet/test, read every changed file, deployed to a live local Mattermost instance and confirmed 6 real `/math` invocations behave as specified).
**Closes:** none new (OQ-001, OQ-002 already resolved prior to this feature — this decision records the implementation that carries them out).
