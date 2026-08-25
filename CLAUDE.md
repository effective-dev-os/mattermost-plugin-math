# mattermost-plugin-math

Mattermost plugin that evaluates a math expression when you mention `@math` in a channel message (e.g. `@math 2 + 2` or `2 + 2 @math`) and posts the result in the channel (or in the thread, if mentioned inside one).

`PROJECT_TYPE: 2` (production, human-gated) — see `.memory-bank/steerings/project-types.md` for what that implies (mandatory human approval on architecture, blocking gates in `production` mode).

## Stack

Go server (Mattermost plugin API) + TypeScript/React webapp. See `.memory-bank/tech-details/stack.md`.

## Entry points

- Working agreement (philosophy, code rules, hard-stops, routing): `AGENTS.md`
- Project knowledge: `.memory-bank/index.md`
- Working memory (rules, decisions, open questions): `.assistant/`

## Sources of truth (in priority order)

1. `.assistant/INVARIANTS.md` — hard rules every agent respects.
2. `AGENTS.md` — complete working agreement.
3. `.memory-bank/index.md` — table of contents for project knowledge.
4. This file — short entry point.

## Modules

Installed: **maintenance** (audit, diagnose, memory-bank-defrag, refactor, reflect, research skills).
