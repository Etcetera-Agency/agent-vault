# Implementation Tasks

## Phase 1: Discovery And API Types

- [x] 1. Add `methods` to discover service response, rendering empty/omitted (unrestricted) as the canonical `["*"]` token.
- [x] 2. Update TypeScript SDK/API types for services and discovery if exported.
- [x] 3. Add server tests for `/discover` returning methods and omitting credential values.

## Phase 2: CLI

- [x] 4. Add method flag or prompt support to service add/set flows.
- [x] 5. Ensure service list YAML includes methods.
- [x] 6. Add CLI tests for method serialization and validation errors.

## Phase 3: Web UI

- [x] 7. Add method multi-select or checkbox control to service create/edit UI.
- [x] 8. Show method badges in service list/table.
- [x] 9. Render an unrestricted service as the `["*"]` badge/token (with `Any method` tooltip/label), consistent with `/discover` and CLI.
- [x] 10. Add UI/API integration tests for method create/edit when test harness exists.

## Phase 4: Docs

- [x] 11. Update service docs with method allowlist examples.
- [x] 12. Update agent protocol docs so agents know `/discover.services[].methods`.
- [x] 13. Update CLI docs for method flags/YAML.
