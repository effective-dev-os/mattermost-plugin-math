# Research: Go library for /math expression evaluation

**Date:** 2026-08-25
**Resolves:** OQ-002 (.assistant/open-questions.md)

## TL;DR

1. **Adopt `github.com/expr-lang/expr`**, pinned **`>=v1.17.7`** (corrected — see risky drift below). Actively maintained, MIT, wide production adoption, position-annotated error messages, `^`/`**` both mean exponent (not XOR — the classic expr-lang gotcha does not apply).
2. No candidate library natively parses `2 x 2`, `×`, `÷`, unicode minus, comma decimals, implicit multiplication, or `%`-as-percent. The plugin must run a **text-normalization pass** on the raw slash-command string before `expr.Compile`.
3. Builtins: `abs`/`ceil`/`floor`/`round`/`min`/`max`/`mean`/`median` ship with expr-lang. `sqrt`, `sin`, `cos`, `log` must be registered manually via `expr.Function`, and `sin`/`cos` will be radians unless converted — needs a product decision (see risky drift).

## Safe drift (applied)

- `.assistant/open-questions.md` OQ-002 → resolved, decision recorded below.

## Risky drift (required human review — see AskUserQuestion in-session)

1. **New production dependency**: adding `github.com/expr-lang/expr` to `go.mod` (PROJECT_TYPE 2, production/human-gated — new third-party dep needs explicit sign-off).
2. **Version pin correction**: researcher initially cited `>=v1.17.0` as fixing GHSA-cfpf-hrx2-8rv6. Skeptic + reviewer independently caught this is wrong — two *separate* advisories exist:
   - `GHSA-93mq-9ffx-83m2` / CVE-2025-29786 (memory exhaustion via unbounded input, published 2025-03-16) — fixed in **v1.17.0**.
   - `GHSA-cfpf-hrx2-8rv6` / CVE-2025-68156 / GO-2025-4245 (unbounded recursion in `flatten`/`min`/`max`/`mean`/`median`, published 2025-12-16) — fixed in **v1.17.7**.
   Correct pin: **`>=v1.17.7`** to cover both. (Practical exposure to the recursion bug is low for this plugin's flat arithmetic strings, but the pin should close it anyway.)
3. **Trig unit convention**: `sin`/`cos` via Go's `math.Sin`/`math.Cos` are radians-based. A chat user typing `/math sin(90)` most likely expects degrees. Needs a product decision — convert degrees→radians before calling, or document radians-only.
4. **Module path provenance**: pin the canonical `github.com/expr-lang/expr`, not the legacy `github.com/antonmedv/expr` alias — reviewer flagged this as a stale-alias risk for future security patches.

## Conflicts with prior decisions

None — `.assistant/decisions.md` has only D-001 (harness install), no prior expr-library decision.

## Full researcher YAML

See task a391ce61bf94edcb8 output. Key candidates evaluated: Knetic/govaluate (archived, reject), expr-lang/expr (recommended), PaesslerAG/gval (stale, medium confidence), maja42/goval (fallback), soniah/evaler (8yr stale, known parse bug, reject), Zac-Garby/pluto (does not exist as a fit, reject), google/cel-go (overkill, reject), hand-rolled parser (fallback only).

## Skeptic critique (key points)

- `^`/`**` = exponent in expr-lang, not XOR — no issue, corroborated.
- `abs` is already builtin; researcher's fix note to "register abs via expr.Function" was imprecise — don't shadow the builtin.
- **HIGH**: conflated two distinct DoS advisories under one GHSA ID / fix-version — corrected above.
- Normalization regex needs a wider edge-case matrix: paren-before-digit `(3+4)2`, paren-paren `(2+3)(4+5)`, digit-before-function `2sqrt(4)`, mid-expression `%`, comma-as-thousands-separator ambiguity, and dash variants beyond U+2212 (en dash U+2013, em dash U+2014).

## Reviewer critique (key points)

- Confirmed HIGH version-pin error independently (corroborated by skeptic).
- "Google" listed as an expr-lang adopter could not be verified against the project's own adopters list — drop or re-source that specific claim (low stakes, not load-bearing).
- Transitive dependency count for `expr-lang/expr` not confirmed — run `go mod graph` diff on a scratch branch before merging.
- No prior-decision conflicts (only D-001 exists).
- `requires_human: true` correctly set at the recommendation level per PROJECT_TYPE 2 gate.
