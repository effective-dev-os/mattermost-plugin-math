# Vision

Math plugin for Mattermost: adds a `/math <expression>` slash command to Mattermost chat that evaluates the expression and posts the formatted result back into the channel.

Example:
```
/math (1 - 0.6) * 1234
```
returns:
```
`(1 - 0.6) * 1234` = `493.6`
```

## Target audience

Internal Mattermost workspace users doing quick chat-based arithmetic (not a public marketplace-polished product for v1) — no settings/config surface, no i18n, matches the "no webapp UI planned for v1" stack decision.

## Definition of Done

- Arithmetic operators `+ - * / ^`, parentheses, `sqrt`, `sin`/`cos` (degrees), `log` (base-10), `abs` (expr-lang builtin).
- Human-notation normalization: `×`/`÷`/digit-adjacent `x`, unicode minus/en-dash/em-dash, comma-as-decimal, implicit multiplication (`2(3+4)`, `(3+4)2`, `(2+3)(4+5)`, `2sqrt(4)`), `%` as percent (`/100`) applied to the preceding operand regardless of position.
- Hard 1024-rune input cap, rejected before compile.
- Success posts in-channel (`` `expr` = `result` ``); malformed/unsafe input (empty, over-length, compile error, runtime error, non-finite result) returns a clear ephemeral error, never a raw Go/expr error string.
- Engine: `github.com/expr-lang/expr` pinned `>=v1.17.7` for two DoS CVEs (see `.assistant/decisions.md` D-002).

## What we don't do

- No symbolic algebra, no LaTeX/rich-math rendering, no expression history/persistence, no per-user settings.
- No modulo operator exposed to users (every `%` is normalized to percent-division, see D-002 and the implementation decisions).
- No webapp/UI component — `/math` is server-only for v1.
