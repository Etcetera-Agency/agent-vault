# Design: Quota and Account Management UI

## Fork-Conflict-Surface Choice

- New React components (`QuotaEditor.tsx`, `AccountPoolEditor.tsx`) rather than
  large edits to `ServicesTab.tsx`; the tab gains a minimal mount point.
- API round-trip achieved through the existing full-service update path; a read
  path for usage derives from the egress counters (read-only).
- New fork-local tests only.

## API Surface

- Service create/edit accepts/returns `quota`, `accounts`, `rotation` (additive).
- Read path returns, per account: `daily_used`/`daily_cap`,
  `monthly_used`/`monthly_cap`, and `state` (`available` | `exhausted` |
  `cooling` with `available_at`). Derived from `internal/egressquota`; never
  includes credential values.

## UI

- `QuotaEditor`: optional numeric inputs for `daily_cap`, `monthly_cap`, `rpm`,
  `concurrency`; blank = unset. `rotation` select.
- `AccountPoolEditor`: add/remove accounts; `id` + `credential_key` selector
  (keys only) + optional per-account overrides.
- Usage view: per-account bars/labels for daily and monthly usage vs cap; an
  exhausted/cooling badge with next-available time.
- Validation: reuse backend errors; block invalid input; allow empty quota.

## Non-Goals

- No new enforcement logic (consumes existing counters).
