# Vision

Math plugin for Mattermost: evaluates a math expression whenever `@math` is mentioned as the first or last word of a channel message, and posts the result in the channel (or the thread, if mentioned inside one).

Example:
```
@math (1 - 0.6) * 1234
```
gets a reply:
```
`(1 - 0.6) * 1234` = `493.6`
```

## Target audience

Internal Mattermost workspace users doing quick chat-based arithmetic (not a public marketplace-polished product for v1) — no settings/config surface, no i18n, matches the "no webapp UI planned for v1" stack decision.

## Definition of Done

- Arithmetic operators `+ - * / ^`, parentheses, `sqrt`, `sin`/`cos` (degrees), `log` (base-10), `abs` (expr-lang builtin).
- Human-notation normalization: `×`/`÷`/digit-adjacent `x`, unicode minus/en-dash/em-dash, comma-as-decimal, implicit multiplication (`2(3+4)`, `(3+4)2`, `(2+3)(4+5)`, `2sqrt(4)`), `%` as percent (`/100`) applied to the preceding operand regardless of position.
- Hard 1024-rune input cap, rejected before compile.
- Success posts a reply (`` `expr` = `result` ``) in the channel, or in the thread if the mention was posted inside one; malformed/unsafe input (empty, over-length, compile error, runtime error, non-finite result) gets the same treatment with a clear error message, never a raw Go/expr error string (errors are no longer ephemeral — see `.assistant/decisions.md` D-005: `MessageHasBeenPosted` has no request-bound ephemeral delivery guarantee).
- Engine: `github.com/expr-lang/expr` pinned `>=v1.17.7` for two DoS CVEs (see `.assistant/decisions.md` D-002).

## What we don't do

- No symbolic algebra, no LaTeX/rich-math rendering, no expression history/persistence, no per-user settings.
- No modulo operator exposed to users (every `%` is normalized to percent-division, see D-002 and the implementation decisions).
- No webapp/UI component — `@math` is a server-side message-hook trigger (no slash command, no UI) for v1.
