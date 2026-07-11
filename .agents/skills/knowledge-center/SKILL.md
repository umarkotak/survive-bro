---
name: knowledge-center
description: Locate, interpret, and maintain the Survive Bro project source of truth across product rules, architecture, WebSocket contracts, gameplay data, delivery milestones, and repository decisions. Use before project planning or implementation, when answering repository-specific questions, resolving conflicting requirements, or documenting a changed decision.
---

# Knowledge Center

Use repository evidence, not remembered assumptions.

## Read order

1. Read root `AGENTS.md` for routing and precedence.
2. Read [references/docs-index.md](references/docs-index.md).
3. Load only the documents relevant to the question.
4. Inspect current code and data before treating planned structure as implemented.

## Resolve truth

- Prefer the user's latest explicit instruction.
- Use `contracts/` for wire shape and cross-app semantics.
- Use `game-data/` for balance/content contracts. Treat top-level status `runtime` and sections explicitly named in `runtimeSections` as loaded implementation; remaining `design-contract` sections must be checked against code.
- Use `docs/mvp-spec.md` for player-visible requirements and limits.
- Use `docs/architecture.md` for ownership and invariants.
- Use `docs/implementation-plan.md` for order and gates, not proof of completion.

Call out contradictions that affect behavior, compatibility, security, or scope. Do not silently merge competing rules.

## Maintain knowledge

When implementation changes a source-of-truth decision:

1. Update the owning document in the same change.
2. Avoid duplicating the same rule in multiple documents.
3. Link to the authoritative document from supporting notes.
4. Record measured results as results, never as universal capacity claims.
5. Keep future ideas labeled as future or non-goal.

Return a concise answer with exact file references and identify whether each statement is specified, planned, implemented, or verified.
