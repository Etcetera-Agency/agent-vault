# Implementation Tasks

- [x] 1. Extend the service create/edit API (additive) to round-trip `quota`,
      `accounts`, `rotation`; API tests incl. partial and multi-account configs.
- [x] 2. Add a read path exposing per-account daily/monthly usage vs cap and
      `state` (available/exhausted/cooling + available_at), derived from
      `internal/egressquota`; never returns credential values.
- [x] 3. Add `QuotaEditor` component: optional caps (daily/monthly/rpm/
      concurrency) + rotation select; blank means unset.
- [x] 4. Add `AccountPoolEditor` component: add/remove accounts, credential-key
      selector (keys only), per-account overrides.
- [x] 5. Mount both in `web/src/pages/vault/ServicesTab.tsx`; never render
      credential values.
- [x] 6. Add usage view: per-account daily/monthly usage vs cap + exhausted/
      cooling badge with next-available time.
- [x] 7. UI validation: reuse backend errors; block invalid (e.g. monthly < daily,
      non-numeric); allow empty quota.
- [x] 8. Docs: managing ceilings and reading usage in the UI; note credential
      values are never shown.
