# Design: Account-Only Rotation

## Context

`quota == nil` currently means "do nothing in egressquota." That is correct only
when no account pool exists. Account pools are independent policy: select which
credential to inject. Quota caps are optional constraints on top.

## Decision

Quota registry reservation runs when either condition is true:

- service has `quota`
- service has one or more `accounts`

No configured quota fields means no cap/rate/concurrency denial. Cooldown can
still deny an account after upstream 429, and pool exhaustion can occur when all
accounts are cooling.

## Pseudocode

```go
func shouldUseQuotaRegistry(service):
    return service.Quota != nil || len(service.Accounts) > 0

func Reserve(service):
    if !shouldUseQuotaRegistry(service):
        return nil, nil

    baseQuota := emptyQuota()
    if service.Quota != nil:
        baseQuota = *service.Quota

    for candidate in orderedCandidates(service):
        denial := checkConfiguredLimitsOnly(candidate.quota)
        if denial == nil:
            return reservation(candidate), nil

    return nil, poolExhaustedDecision()

func checkConfiguredLimitsOnly(quota):
    if quota.daily_cap set: check daily
    if quota.monthly_cap set: check monthly
    if quota.rpm set: reserve rpm token
    if quota.concurrency set: reserve semaphore
    always check cooldown
```

## Test Shape

- Account-only service with `round_robin` injects account 1 then account 2.
- Account-only service with cooled account 1 skips to account 2.
- Account-only service with all accounts cooling returns 429 with retry-after.
- Service with no quota and no accounts stays unchanged.

