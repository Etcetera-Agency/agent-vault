# Design: Quota Failover Replay

## Context

The proxy can retry only if it can rebuild the outbound request exactly enough
for another account attempt. GET/HEAD are trivial. Body methods need a fresh
reader for each attempt.

## Decision

For requests whose matched service has an account pool, the proxy materializes
the request body before the first upstream attempt. This uses the existing
request body size limit. Each attempt gets a new `io.Reader` over the same bytes.

## Pseudocode

```go
func forwardRequest(req):
    inject := creds.Inject(...)
    replay := buildReplayPlan(req, inject)

    for attempt := 0; attempt < maxAttempts; attempt++ {
        outReq := replay.NewRequest(inject)
        resp := upstream.RoundTrip(outReq)

        if !isQuotaReject(resp):
            commit(inject.Reservation, resp.StatusCode)
            return resp

        cooldown(inject.Reservation, retryAfter(resp))
        release(inject.Reservation)

        if !replay.CanRetry():
            return resp

        retryInject, err := creds.Inject(...)
        if err != nil:
            return resp
        inject = retryInject
    }
    return lastResp

func buildReplayPlan(req, inject):
    if hasBodySubstitutions(inject):
        bodyBytes := materializeAndSubstitute(req.Body)
        return replayable(bodyBytes)
    if serviceHasAccountPool(inject):
        bodyBytes := materialize(req.Body)
        return replayable(bodyBytes)
    return streaming(req.Body)
```

## Safety

- Retry count is bounded by the number of configured accounts and a small hard
  cap.
- Request body limit is enforced before materialization.
- Headers are rebuilt per attempt so selected account auth replaces previous
  injected auth.
- Failed reservation is cooled and released before selecting the next account.

