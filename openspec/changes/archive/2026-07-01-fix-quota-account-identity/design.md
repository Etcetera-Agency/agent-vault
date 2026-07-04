# Design: Quota Account Identity

## Context

`ServiceAccount` has two fields with different meanings:

- `id`: stable operator-facing account identity, used in quota specs and UI.
- `credential_key`: non-secret reference to stored credential material.

Runtime currently collapses both into `credential_key`. The fix separates
identity from injection.

## Decisions

- All quota registry keys use `(vault_id, service_name, account_id)`.
- Reservations carry both `account_id` and `credential_key`.
- `CredentialKeys` in request logs remains credential reference metadata; new
  `AccountID` stores selected account identity.
- Implicit single-credential services keep using internal account id
  `__service__`; UI may render that as `default`.

## Pseudocode

```go
type Candidate struct {
    accountID     string
    credentialKey string
    quota         ServiceQuota
}

func candidates(service):
    if len(service.Accounts) == 0:
        key := first(service.Auth.CredentialKeys())
        return []Candidate{{accountID: "__service__", credentialKey: key, quota: baseQuota}}

    out := []Candidate{}
    for acct in service.Accounts:
        out = append(out, Candidate{
            accountID: acct.ID,
            credentialKey: acct.CredentialKey,
            quota: merge(baseQuota, acctOverrides),
        })
    return orderByRotation(service.Rotation, out)

func reserveCandidate(candidate):
    quotaStateKey := vaultID + "\x00" + serviceName + "\x00" + candidate.accountID
    check day/month/rpm/concurrency/cooldown by quotaStateKey
    return Reservation{AccountID: candidate.accountID, CredentialKey: candidate.credentialKey}

func inject():
    reservation := quota.Reserve(service)
    if reservation.CredentialKey != "":
        service.Auth = accountAuth(service.Auth, reservation.CredentialKey)
    result.MatchedAccountID = reservation.AccountID
    result.CredentialKeys = []string{reservation.CredentialKey}
```

## Test Shape

- Least-used chooses `acct2` when `acct1` has higher usage, even when credential
  keys are `APIFY_TOKEN_1` and `APIFY_TOKEN_2`.
- Cooldown on `acct1` does not cool `APIFY_TOKEN_1` globally for another service.
- Usage endpoint reports counters under `id: "acct1"` and never uses
  `credential_key` as display identity.
- Restart seeding reconstructs day/month counters from request logs carrying
  `account_id`.

