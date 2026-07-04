# Design: Credential Pool Eligibility

## Context

`accounts[]` is service-local policy, but pool safety belongs to credentials:
only the credential owner knows whether the token represents an interchangeable
provider account or a user/workspace identity that must stay isolated.

## Decisions

- Store `credentials.pool_provider TEXT NULL`.
- `NULL` or empty string means "not poolable".
- Provider id is a slug (`apify`, `amadeus`, `google-calendar`, etc.).
- Service config gains `account_pool_provider`.
- If `len(service.accounts) > 0`, `account_pool_provider` is required.
- Service validation checks every account credential exists and has matching
  `pool_provider`.
- Mail proxy and OAuth provider templates do not auto-enable pooling.

## Pseudocode

```go
type Credential struct {
    Key          string
    PoolProvider string // empty => not poolable
}

type Service struct {
    Accounts            []ServiceAccount
    AccountPoolProvider string
}

func validateServiceAccountPool(service, credentialsByKey):
    if len(service.Accounts) == 0:
        return nil
    if service.AccountPoolProvider == "":
        return error("account_pool_provider is required when accounts are set")
    for acct in service.Accounts:
        cred := credentialsByKey[acct.CredentialKey]
        if cred == nil:
            return error("credential_key does not exist")
        if cred.PoolProvider != service.AccountPoolProvider:
            return error("credential_key is not enabled for this account pool provider")
    return nil
```

## UI Shape

- Credential table: show pool state badge.
  - `Not poolable`
  - `Pool: apify`
- Credential edit modal: optional `Pool provider` field.
- Service account-pool editor:
  - provider input/select required when accounts exist
  - credential selector filters to matching pool provider
  - non-matching existing rows show validation error

## Security

- `pool_provider` is metadata only; no credential values are exposed.
- Default is not poolable, including migrated credentials.
- Account-pool validation runs server-side; UI filtering is only convenience.

