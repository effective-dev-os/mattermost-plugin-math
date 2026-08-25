# /math slash command — pre-feature plan

Status: consilium-complete

## Task container

- slug: math-slash-command
- scope-fence: `server/mathexpr/` (new package: normalization + eval, zero Mattermost imports), `server/command/command.go` (register `math` trigger, wire mathexpr, remove leftover `hello` command), `server/command/command_test.go` (update/replace), `server/command/mocks/mock_commands.go` (regenerate if interface narrows), `go.mod`/`go.sum` (add `github.com/expr-lang/expr`). No webapp changes.
- definition-of-done:
  1. `server/mathexpr` package exports pure, testable `Normalize(raw string) (string, error)` and `Eval(raw string) (float64, error)` (or equivalent split), each step of the normalization pipeline a separate unit-tested function.
  2. `sqrt`, `sin`, `cos`, `log` registered via `expr.Function` with `new(func(float64) float64)` type hints; `abs` NOT re-registered (builtin). `sin`/`cos` take degrees.
  3. Hard rune-length cap (1024, see rationale below) enforced before `expr.Compile`.
  4. `/math <expr>` registered in `server/command/command.go` following the existing trigger-dispatch pattern; success → `model.CommandResponseTypeInChannel`; errors → ephemeral, no raw Go error text leaked.
  5. Leftover starter-template `hello` command removed (plugin.json already scopes this plugin to `/math` only).
  6. Table-driven unit tests cover at minimum the D-002 test matrix (see Research findings + Blockers below for the two resolved-by-this-plan behaviors: percent and log base).
  7. `make check-style` and `make test` (or `go build ./... && go test ./...`) pass.
  8. Manual E2E: plugin built, deployed to a running Mattermost instance, `/math` invoked with several real cases, channel output visually confirmed correct (per AGENTS.md validation pipeline / ANTI-11 — unit tests passing is not "done").
- verification-contract: `go build ./...`, `go test ./... -v` (all green), `make check-style` if configured, manual `/math` smoke test against a live server recorded in `swarm-report/math-slash-command-e2e-scenario.md`.
- inputs-needed: none from secrets/env for the code itself. Local E2E deploy step needs `MM_SERVICESETTINGS_SITEURL` and `MM_ADMIN_TOKEN` as local env vars only (already provided out-of-band by the coordinator for this run — never committed, per INVARIANT §12).
- out-of-scope: webapp changes, autocomplete beyond the existing `AutocompleteData` pattern, persistence/history of past `/math` calls, rate-limiting beyond the length cap, i18n of error messages.

## TL;DR

- 4 agents, ~55 raw findings before dedup, ~34 after.
- 2 items required a genuine human/product decision beyond what D-002 already settled. **`AskUserQuestion` is not available as a tool in this run's toolset** (verified via `ToolSearch`), so per Auto-Mode guidance ("make the reasonable call, they'll redirect you if needed") these were resolved by direct analogy to an already-human-approved precedent (D-002's degrees decision) rather than left blocking. Both are called out below for override.
- Must-fix top 3: (1) normalization/eval must live in a new `server/mathexpr` package, not `server/command` — the latter is coupled to `pluginapi.Client` and untestable as pure logic; (2) `expr.Compile` must use `expr.Env(map[string]any{})` + `expr.AsFloat64()` — without these, unrelated builtins/identifiers, non-numeric results, and confusing runtime-only errors all leak through; (3) division by zero does NOT error in expr-lang (`1/0` → `+Inf`, `0/0` → `NaN`) — must be checked explicitly post-`Run` via `math.IsInf`/`math.IsNaN`.

## Resolved-here decisions (would otherwise be Blockers — see rationale)

### 1. Mid-expression `%` semantics
**Decision: every `%` is treated as postfix-percent, uniformly, regardless of position.** Each `%` triggers a backward scan from its position to find the immediately preceding operand — a numeric literal, or a balanced `(...)` group optionally prefixed by a function-call identifier or unary minus — and rewrites that operand as `(<span>/100)`. This removes expr-lang's integer-modulo operator from user-facing input entirely (documented in a code comment, since it's non-obvious). `50% + 10` → `(50/100) + 10` = 0.6. `10%3` → `(10/100)3` → implicit-multiplication then inserts `*` → `(10/100)*3` = 0.3 (not `10 mod 3 = 1`).
Rationale: this is the exact "recommended" behavior given in the originating task brief ("treat every % occurrence the same way as trailing... rather than leaving it as modulo or erroring silently"), and it composes cleanly with the implicit-multiplication pass already required (no special-casing needed — the paren-before-digit rule glues the rewritten span to whatever follows). 3 of 4 consilium agents (architect, skeptic, reviewer) flagged this as `requires_human: true` because D-002 says it "must be explicitly decided during implementation" — that explicit decision is recorded here, with rationale, per the brief's own instruction to decide-and-document rather than re-ask when a recommendation was already given. **Flagged for override**: if the human wants `%` to keep modulo semantics in binary-infix position (e.g., `10 % 3`) and only mean percent when trailing/immediately-postfix (`50%`), say so and this gets revised before `/implementor` finalizes.

### 2. `log()` base
**Decision: `log` = base-10 (`math.Log10`), not natural log.** D-002 does not specify this, and no prior decision covers it. Direct precedent: D-002's own stated rationale for degrees on `sin`/`cos` — "chat users expect `sin(90) = 1` not radians," i.e., prioritize calculator/everyday convention over programming-language convention. The same reasoning applies to `log`: calculator users expect `log(100) = 2`; only programmers expect natural log from a bare `log`. **Flagged for override**: if natural log (`math.Log`) is actually wanted, or a second `ln` function should be added, say so before `/implementor` finalizes; this is a one-line change plus a test-row change.

## Blockers (HIGH, requires_human, unresolved)

None remaining — the two items above were the only HIGH `requires_human: true` findings from the consilium; both resolved above by direct precedent per Auto-Mode guidance, with explicit override flags surfaced to the human in the final report.

## Concerns (MEDIUM) — binding on /implementor

- **Package boundary** (architect, HIGH): `server/mathexpr`, zero Mattermost imports, exported errors as a sentinel set (`ErrEmptyInput`, `ErrTooLong`, `ErrUnsupportedSyntax`, `ErrCompile`, `ErrRuntime`, `ErrNonFiniteResult`). `server/command` maps sentinels → fixed ephemeral strings; never interpolates `err.Error()` from expr into a response (skeptic, HIGH: expr error text is multi-line, includes caret diagrams and raw Go stdlib text).
- **Pipeline order is fixed, not implementor's choice** (architect HIGH + skeptic MEDIUM, corroborated): (1) reject empty/oversized raw input (cap: 1024 runes via `utf8.RuneCountInString`, not `len` — architect flagged byte-vs-rune mismatch for `×`/`÷`/unicode minus); (2) unicode symbol → ASCII (`×`, digit-adjacent `x`/`X` → `*`; `÷` → `/`; U+2212/U+2013/U+2014 → `-`); (3) comma-between-digits → `.`; (4) percent rewrite (must run before implicit-multiplication so its emitted parens are visible to the paren-adjacency rules — e.g. `50%(2+3)` → `(50/100)(2+3)` → `(50/100)*(2+3)`); (5) implicit multiplication (digit-before-paren, paren-before-digit, paren-before-paren, digit-before-function-name); (6) character/identifier allowlist validation (digits, `.`, `+ - * / ^`, parens, whitespace, and identifiers exactly in `{sqrt, sin, cos, log, abs}` — reject everything else, which blocks string/array/pipe/range/comparison syntax and any other expr-lang builtin in one step).
- **RE2 has no backreferences/lookaround** (skeptic HIGH, verified): the percent rewrite (balanced-paren backward scan) and the hex/scientific-notation exclusions cannot be single regexes. Implement as hand-written rune-scanning functions, not `regexp`.
- **Literal collisions** (architect + skeptic, HIGH, verified empirically against v1.17.8): exclude `0x`/`0X`/`0b`/`0B`/`0o`/`0O`-prefixed runs from the digit-adjacent-`x` rule (`0x10` must stay `0x10` = 16, not become `0*10` = 0); exclude `[0-9][eE][+-]?[0-9]` from the digit-before-identifier rule (`1e5` must stay `1e5`, not become `1*e5`). Required test rows: `0x10`, `1e5`, `2e-3`.
- **Compile options** (researcher, HIGH, corroborated against v1.17.8 API + source): `expr.Env(map[string]any{})` (unknown identifiers → clean compile error, not a confusing runtime `cannot fetch X from <nil>`), `expr.AsFloat64()` (forces float64 result; also converts int results like `1+1`), `expr.Function("sqrt", fn, new(func(float64) float64))` (and same for `sin`/`cos`/`log`) — the type hint is what turns arity/type mistakes into compile errors instead of a panic inside the Go callback. Do not call `expr.MaxNodes(0)` (disables the CVE-2025-29786 mitigation, which defaults to 1e4 nodes and is otherwise sufficient — verified empirically, no extra guard needed here).
- **Division/modulo by zero never errors at the operator level for `/`** (architect + researcher, HIGH, verified empirically v1.17.8): `1/0` → `+Inf`, `-1/0` → `-Inf`, `0/0` → `NaN`, all as *successful* `expr.Run` results. After `Run`, explicitly check `math.IsInf`/`math.IsNaN` on the float64 result and map to `ErrNonFiniteResult`. (Modulo itself is moot post-normalization since every `%` is rewritten away per the resolved decision above — but if a literal `%` somehow reaches `expr.Compile` due to a normalization edge case, note expr's own `%` division-by-zero fails at compile time for constants and at runtime for variables — both surface as compile/runtime errors, not panics, since expr's VM recovers internally.)
- **Result formatting** (architect + skeptic, MEDIUM, verified): raw float64 output is user-hostile (`sin(30)` → `0.49999999999999994`, `sin(180)` → `1.2246467991473515e-16`). Implement one `formatResult(float64) string`: round to a fixed precision (e.g. `strconv.FormatFloat(v, 'g', 12, 64)`), trim trailing zeros, and snap results within a small epsilon (e.g. `1e-9`) of an integer to that integer so degree-based trig reads clean. Table-test `sin(30)==0.5`, `sin(180)==0`, `cos(90)==0`.
- **Response type and Markdown escaping** (skeptic + reviewer, HIGH/MEDIUM): the existing `hello` handler leaves `ResponseType` empty, which the Mattermost server treats as ephemeral (verified against vendored `server/v8` source) — do NOT copy that omission. Set `ResponseType: model.CommandResponseTypeInChannel` explicitly on success. Wrap the echoed expression/result in backticks or set `SkipSlackParsing: true` so `*`-heavy expressions (e.g. `2*3*4*5`) don't get mangled as Markdown emphasis.
- **Existing test breakage** (architect, MEDIUM): `server/command/command_test.go`'s `TestHelloCommand` sets exactly one `RegisterCommand` expectation on `plugintest.API`; since `hello` is being removed (not kept alongside `math`), this is moot for registration-count but the test itself must be replaced with math-command coverage, not left referencing a deleted trigger.
- **`Command` interface / mock hygiene** (architect + reviewer, MEDIUM/LOW): keep the interface to `Handle(args *model.CommandArgs) (*model.CommandResponse, error)` only — do not add `executeMathCommand` to the interface (mirrors the existing unexported-method-on-exported-interface bug in the current mock, which should not be perpetuated). Regenerate `server/command/mocks/mock_commands.go` via `make mock` if the interface shape changes at all.
- **Dependency add** (all agents, MEDIUM): `go get github.com/expr-lang/expr@v1.17.8` (latest, satisfies D-002's `>=v1.17.7` floor; researcher confirmed v1.17.8's release notes contain no security fix beyond v1.17.7, and no new CVE has been disclosed as of 2026-08-25 per OSV.dev/GHSA cross-check). Confirm canonical path used, `github.com/antonmedv/expr` appears nowhere in `go.mod`/`go.sum`.

## Notes (LOW)

- int64 arithmetic can silently overflow before `expr.AsFloat64()`'s final-result conversion applies (e.g. `1000000*1000000*1000000*1000` wraps) — accepted as a known limitation for v1, not fixed (contrived input, low real-world likelihood for a chat calculator, avoids scope creep per AGENTS.md pragmatism). Not blocking.
- `strings.Fields(args.Command)` (used for trigger dispatch) collapses whitespace; do not reuse it to extract the expression argument — use `strings.TrimSpace(strings.TrimPrefix(args.Command, "/"+mathCommandTrigger))` instead so the length cap measures what the user actually typed.
- expr v1.17.8 also enforces a runtime memory budget independent of `MaxNodes` (verified: `repeat("a", 50000000)` fails fast with `memory budget exceeded`) — no additional allocation guard needed.
- `.memory-bank/tech-details/stack.md` carries a TODO about documenting server↔webapp slash-command wiring conventions; close it in `/post-feature` noting `/math` is server-only.

## Research findings (researcher, all confidence: corroborated/high unless noted)

- Latest `expr-lang/expr` release is v1.17.8 (2026-02-14); D-002's `>=v1.17.7` pin remains correct for both cited CVEs, no new advisory since.
- `abs` confirmed builtin; 64 builtins total including math-adjacent `ceil`, `floor`, `round`, `min`, `max`, `sum`, `mean`, `median` — none of these get registered by this feature, and the character-allowlist step (not a separate `DisableAllBuiltins` call) is the chosen single mechanism to keep them unreachable, since the allowlist already blocks any identifier outside `{sqrt, sin, cos, log, abs}`.
- Mattermost enforces no length cap on slash-command text server-side beyond `len(commandArgs.Command) > 1`/leading-`/` checks (verified against `server/public` v0.1.21 + `api4/command.go`) — the plugin-side cap is NOT redundant, confidence: corroborated. Recommended/chosen cap: 1024 runes, matching Mattermost's own `Command.URL` bound (confidence: medium, product threshold not a derived fact — flagged, but not escalated since the task brief explicitly delegates "pick a sane cap... and justify it" to the implementor).

## Out-of-scope (declared)

- Webapp changes (confirmed no UI planned for v1, `.memory-bank/tech-details/stack.md`).
- `NewCommandHandler` returning `(Command, error)` so registration failures fail plugin activation loudly (architect MEDIUM) — the existing `hello` pattern logs-and-continues on registration failure; this feature follows that existing convention rather than introducing a new one, to avoid an unrelated refactor. Noted as a pre-existing gap, not fixed here.
- `DisableAllBuiltins()` + `EnableBuiltin` narrowing (researcher LOW, optional hardening) — superseded by the character-allowlist mechanism chosen above; not adding a second overlapping defense layer.

## Open questions raised

None new — the two ambiguities found (percent semantics, log base) were resolved above rather than left open, with explicit override flags for the human to revise before `/implementor` finalizes.

---

## Per-agent verbatim findings

See subagent transcripts (not persisted verbatim here to keep this report scannable); full YAML retained in orchestrator conversation history for this run. Summary counts: architect 21 findings (5 HIGH), skeptic 19 findings (7 HIGH), researcher 18 findings (5 HIGH), reviewer 10 findings (2 HIGH).
