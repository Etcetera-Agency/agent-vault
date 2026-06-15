# Implementation Tasks

## Phase 1: Data Model

- [x] 1. Add `Methods []string` to `internal/broker.Service` with JSON/YAML tag `methods,omitempty`.
- [x] 2. Add method normalization helper that uppercases valid tokens and collapses the sole-element `["*"]` to an empty (unrestricted) list.
- [x] 3. Add method validation helper that rejects empty strings, duplicates, invalid method tokens, and `*` mixed with other methods.

## Phase 2: Ingest Paths

- [x] 4. Wire validation through `broker.Validate`.
- [x] 5. Add `Methods []string` to `proposal.Service`, copy it in `toBrokerService`, and wire validation through proposal service validation.
- [x] 5a. Confirm proposal upsert sets `methods` exactly as proposed (no preserve-on-empty), so a service can be widened back to any-method.
- [x] 6. Ensure direct service set/add/upsert endpoints preserve `methods`.
- [x] 7. Ensure CLI YAML service list/set/add round-trips `methods`.

## Phase 3: Tests

- [x] 8. Add broker validation tests for valid methods.
- [x] 9. Add broker validation tests for invalid methods, duplicates, mixed `*`, and the `["*"]` → unrestricted normalization.
- [x] 10. Add server service API round-trip test for `methods`.
- [x] 11. Add proposal validation + merge test proving service `methods` survive `toBrokerService` and that upsert replaces (not preserves) an emptied method set.

## Phase 4: Docs

- [x] 12. Document `methods` as accepted policy data and note enforcement arrives in next slice.
