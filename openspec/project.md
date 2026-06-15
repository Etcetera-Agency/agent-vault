# Agent Vault OpenSpec Context

## Purpose

This OpenSpec tree tracks Agent Vault changes needed for agent-safe service API proxying. The goal is to let agents use normal third-party APIs and SDKs through Agent Vault without receiving real service credentials, while Agent Vault enforces host, path, and HTTP method policy.

## Ground Rules

- Keep real credentials only in Agent Vault.
- Give agents placeholder credential values and an Agent Vault token only.
- Require strict deny for unmatched hosts before enabling real integrations.
- Start with narrow host, path, and method scopes.
- Keep slices vertical and implementation-sized: one schema step, one enforcement step, one management-surface step, one profile step.
- Record deferred work in `openspec/TODO.md` before considering OpenSpec work complete.

## Fork Discipline

This repository is a **fork of upstream `github.com/Infisical/agent-vault`**. We intend to keep pulling upstream updates, so every slice MUST minimize the merge-conflict surface against upstream. This is a hard constraint, not a preference.

Rules for all implementation work:

- **Prefer new files over edits to upstream files.** Put new logic (method normalization/validation, method matching helpers, agent profile, fixtures) in dedicated new files (e.g. `internal/broker/methods.go`) rather than inside existing upstream bodies. New files never conflict on rebase.
- **When an upstream file must change, keep the diff minimal and additive.** Append struct fields at the tail with `,omitempty`; add new exported helpers instead of rewriting existing ones; do not reorder, rename, or reformat surrounding upstream code.
- **Do not change existing upstream function signatures when a wrapper or post-step achieves the same result.** A signature change forces edits to every call site — maximal conflict surface. Prefer adding a new function or a thin gate at the caller.
- **Mark every local fork edit with a `// fork-local:` comment** so rebases are mechanical and reviewers can find our delta.
- **Keep tests in new fork-local `*_test.go` files** rather than editing upstream test files.
- **Mirror upstream style exactly** (naming, comment density, error-message shape) so diffs stay small and reviewable.

Each slice's `design.md` records its concrete fork-conflict-surface choice.

## Source Plan

Current slices are derived from `/Users/theDay/Hermes/SERVICE_API_PROXY_PLAN.md`.
