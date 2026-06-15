# Implementation Tasks

## Phase 1: API Capability Check

- [x] 1. Verify existing service APIs can create, replace, disable, and delete one service without dropping fields.
- [x] 2. If single-service update is incomplete, extend the focused service update API to accept full service fields.
- [x] 3. Add API tests for editing host/path, methods, auth config, enabled state, and substitutions.

## Phase 2: UI Editor

- [x] 4. Add service create/edit form for name, host/path, methods, enabled state, and auth type.
- [x] 5. Add credential key selector/input for auth configs without displaying credential values.
- [x] 6. Add substitution editor for placeholder, credential key, and substitution locations.
- [x] 7. Add delete and disable flows with confirmation where destructive.
- [x] 8. Ensure list/table displays name, host/path, methods, enabled state, auth type, and referenced credential keys.

## Phase 3: Validation And Safety

- [x] 9. Reuse backend validation errors in UI submission feedback.
- [x] 10. Prevent saving duplicate service names or invalid method sets.
- [x] 11. Ensure editing a Telegram placeholder service preserves substitutions.
- [x] 12. Ensure editing a Google read service preserves `methods: ["GET"]`.

## Phase 4: Docs

- [x] 13. Document editing agent allowlists through UI.
- [x] 14. Document that credential values are never shown in the editor.
- [x] 15. Document that agents should refresh `/discover` after allowlist changes.
