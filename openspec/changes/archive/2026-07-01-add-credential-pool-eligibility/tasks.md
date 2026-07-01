# Implementation Tasks

- [x] 1. Add credential metadata storage for `pool_provider` with migration and
      default empty value for existing credentials.
- [x] 2. Extend store interfaces and SQL implementation to read/update
      credential pool provider without touching credential ciphertext.
- [x] 3. Extend credential list/update API with non-secret `pool_provider`.
- [x] 4. Add `account_pool_provider` to broker service config and service
      response/types.
- [x] 5. Enforce server-side validation: account pools require
      `account_pool_provider`; every account credential must have matching
      credential `pool_provider`.
- [x] 6. Keep mail proxy/Gmail/Google Calendar credentials unpoolable by default;
      add tests proving they are rejected unless explicitly assigned a matching
      pool provider.
- [x] 7. Update credential UI: show pool state and let admins set/clear provider.
- [x] 8. Update service UI: require provider for account pools, filter credential
      choices by matching provider, and show clear validation errors.
- [x] 9. Update SDK/types/docs for credential pool metadata and service
      `account_pool_provider`.
- [x] 10. Add tests for successful matching provider, rejected unpoolable
      credential, rejected wrong provider, and metadata-only API response.
- [x] 11. Run `go test ./internal/store ./internal/server ./internal/broker`.
- [x] 12. Run `npm run build` in `web`.
- [x] 13. Run `openspec validate add-credential-pool-eligibility --strict`.
