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
