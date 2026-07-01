package egressquota

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/store"
)

const implicitAccount = "__service__"

type requestLogStore interface {
	ListRequestLogs(ctx context.Context, opts store.ListRequestLogsOpts) ([]store.RequestLog, error)
}

type Registry struct {
	mu       sync.Mutex
	now      func() time.Time
	counters map[string]*counterState
	limits   map[string]*limitState
	rrIndex  map[string]int
	alerted  map[string]bool

	OnExhausted func(ctx context.Context, vaultID, service string)
}

type counterState struct {
	dayKey     string
	dayCount   int
	monthKey   string
	monthCount int
}

type limitState struct {
	rpmHits       []time.Time
	inFlight      int
	cooldownUntil time.Time
}

type Decision struct {
	Service    string
	RetryAfter time.Duration
	Reason     string
}

type Reservation struct {
	reg           *Registry
	key           string
	service       string
	accountID     string
	credentialKey string
	released      bool
	recorded      bool
}

type Snapshot struct {
	VaultID        string
	MatchedService string
	AccountID      string
	CredentialKeys []string
	CreatedAt      time.Time
	Status         int
}

func New() *Registry {
	return &Registry{
		now:      time.Now,
		counters: make(map[string]*counterState),
		limits:   make(map[string]*limitState),
		rrIndex:  make(map[string]int),
		alerted:  make(map[string]bool),
	}
}

func (r *Registry) SeedFromRequestLogs(ctx context.Context, s requestLogStore, limit int) error {
	return r.loadFromRequestLogs(ctx, s, limit, false)
}

func (r *Registry) ReconcileFromRequestLogs(ctx context.Context, s requestLogStore, limit int) error {
	return r.loadFromRequestLogs(ctx, s, limit, true)
}

func RunReconcile(ctx context.Context, r *Registry, s requestLogStore, every time.Duration) {
	if r == nil || s == nil {
		return
	}
	if every <= 0 {
		every = 5 * time.Minute
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = r.ReconcileFromRequestLogs(ctx, s, 10000)
		}
	}
}

func (r *Registry) loadFromRequestLogs(ctx context.Context, s requestLogStore, limit int, replace bool) error {
	if limit <= 0 {
		limit = 10000
	}
	rows, err := s.ListRequestLogs(ctx, store.ListRequestLogsOpts{Limit: limit})
	if err != nil {
		return err
	}
	snapshots := make([]Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, Snapshot{
			VaultID:        row.VaultID,
			MatchedService: row.MatchedService,
			AccountID:      row.AccountID,
			CredentialKeys: row.CredentialKeys,
			CreatedAt:      row.CreatedAt,
			Status:         row.Status,
		})
	}
	if replace {
		r.Reconcile(snapshots)
	} else {
		r.Seed(snapshots)
	}
	return nil
}

func (r *Registry) Seed(rows []Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	dayKey, monthKey := windowKeys(now)
	for _, row := range rows {
		if row.MatchedService == "" || row.Status <= 0 || row.CreatedAt.IsZero() {
			continue
		}
		key := quotaKey(row.VaultID, row.MatchedService, snapshotAccountID(row))
		state := r.counterForLocked(key, now)
		rowDay, rowMonth := windowKeys(row.CreatedAt)
		if rowDay == dayKey {
			state.dayCount++
		}
		if rowMonth == monthKey {
			state.monthCount++
		}
	}
}

func (r *Registry) Reconcile(rows []Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	next := make(map[string]*counterState)
	dayKey, monthKey := windowKeys(now)
	for _, row := range rows {
		if row.MatchedService == "" || row.Status <= 0 || row.CreatedAt.IsZero() {
			continue
		}
		key := quotaKey(row.VaultID, row.MatchedService, snapshotAccountID(row))
		state := next[key]
		if state == nil {
			state = &counterState{dayKey: dayKey, monthKey: monthKey}
			next[key] = state
		}
		rowDay, rowMonth := windowKeys(row.CreatedAt)
		if rowDay == dayKey {
			state.dayCount++
		}
		if rowMonth == monthKey {
			state.monthCount++
		}
	}
	r.counters = next
}

func (r *Registry) Reserve(ctx context.Context, vaultID string, svc broker.Service) (*Reservation, *Decision) {
	if svc.Quota == nil && len(svc.Accounts) == 0 {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, &Decision{Service: svc.Name, RetryAfter: time.Second, Reason: "context"}
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	candidates := r.orderedCandidatesLocked(vaultID, svc, now)
	var best *Decision

	for _, candidate := range candidates {
		denial := r.reserveCandidateLocked(now, svc, candidate)
		if denial == nil {
			r.recordRoundRobinLocked(vaultID, svc, candidate.accountID)
			return &Reservation{
				reg:           r,
				key:           candidate.key,
				service:       svc.Name,
				accountID:     candidate.accountID,
				credentialKey: candidate.credentialKey,
			}, nil
		}
		best = earlierDecision(best, denial)
	}
	if best == nil {
		best = &Decision{Service: svc.Name, RetryAfter: time.Second, Reason: "exhausted"}
	}
	r.emitExhaustedAlertLocked(ctx, vaultID, svc.Name, now)
	return nil, best
}

func (r *Registry) Snapshot(vaultID, service string, credentialKeys []string) (dayCount, monthCount int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.counterForLocked(quotaKey(vaultID, service, accountKey(credentialKeys)), r.now())
	return state.dayCount, state.monthCount
}

func (r *Registry) Usage(vaultID, service, account string) (dayCount, monthCount int, coolingUntil time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := quotaKey(vaultID, service, account)
	counter := r.counterForLocked(key, r.now())
	limit := r.limitForLocked(key)
	return counter.dayCount, counter.monthCount, limit.cooldownUntil
}

func (r *Registry) counterForLocked(key string, now time.Time) *counterState {
	dayKey, monthKey := windowKeys(now)
	state := r.counters[key]
	if state == nil {
		state = &counterState{}
		r.counters[key] = state
	}
	if state.dayKey != dayKey {
		state.dayKey = dayKey
		state.dayCount = 0
	}
	if state.monthKey != monthKey {
		state.monthKey = monthKey
		state.monthCount = 0
	}
	return state
}

func (r *Registry) limitForLocked(key string) *limitState {
	state := r.limits[key]
	if state == nil {
		state = &limitState{}
		r.limits[key] = state
	}
	return state
}

type candidate struct {
	key           string
	accountID     string
	credentialKey string
	quota         broker.ServiceQuota
}

func (r *Registry) orderedCandidatesLocked(vaultID string, svc broker.Service, now time.Time) []candidate {
	base := serviceQuotaValue(svc.Quota)
	capacity := len(svc.Accounts)
	if capacity == 0 {
		capacity = 1
	}
	candidates := make([]candidate, 0, capacity)
	if len(svc.Accounts) == 0 {
		account := accountKey(svc.Auth.CredentialKeys())
		return []candidate{{key: quotaKey(vaultID, svc.Name, account), accountID: account, credentialKey: account, quota: base}}
	}
	for _, acct := range svc.Accounts {
		quota := base
		if acct.DailyCap != nil {
			quota.DailyCap = acct.DailyCap
		}
		if acct.MonthlyCap != nil {
			quota.MonthlyCap = acct.MonthlyCap
		}
		if acct.RPM != nil {
			quota.RPM = acct.RPM
		}
		// AICODE-NOTE: Quota state keys by service account id; credential_key is only the secret reference injected upstream.
		candidates = append(candidates, candidate{
			key:           quotaKey(vaultID, svc.Name, acct.ID),
			accountID:     acct.ID,
			credentialKey: acct.CredentialKey,
			quota:         quota,
		})
	}
	switch svc.Rotation {
	case "round_robin":
		offset := r.rrIndex[serviceKey(vaultID, svc.Name)] % len(candidates)
		ordered := append([]candidate{}, candidates[offset:]...)
		ordered = append(ordered, candidates[:offset]...)
		return ordered
	default:
		sort.SliceStable(candidates, func(i, j int) bool {
			left := r.counterForLocked(candidates[i].key, now)
			right := r.counterForLocked(candidates[j].key, now)
			return left.dayCount+left.monthCount < right.dayCount+right.monthCount
		})
		return candidates
	}
}

func (r *Registry) reserveCandidateLocked(now time.Time, svc broker.Service, c candidate) *Decision {
	counter := r.counterForLocked(c.key, now)
	if c.quota.DailyCap != nil && counter.dayCount >= *c.quota.DailyCap {
		return &Decision{Service: svc.Name, RetryAfter: untilNextDay(now), Reason: "daily_cap"}
	}
	if c.quota.MonthlyCap != nil && counter.monthCount >= *c.quota.MonthlyCap {
		return &Decision{Service: svc.Name, RetryAfter: untilNextMonth(now), Reason: "monthly_cap"}
	}

	limit := r.limitForLocked(c.key)
	if limit.cooldownUntil.After(now) {
		return &Decision{Service: svc.Name, RetryAfter: limit.cooldownUntil.Sub(now), Reason: "cooldown"}
	}
	if c.quota.Concurrency != nil && limit.inFlight >= *c.quota.Concurrency {
		return &Decision{Service: svc.Name, RetryAfter: time.Second, Reason: "concurrency"}
	}
	if c.quota.RPM != nil {
		retryAfter := reserveRPM(now, limit, *c.quota.RPM)
		if retryAfter > 0 {
			return &Decision{Service: svc.Name, RetryAfter: retryAfter, Reason: "rpm"}
		}
	}
	if c.quota.Concurrency != nil {
		limit.inFlight++
	}
	return nil
}

func (r *Registry) recordRoundRobinLocked(vaultID string, svc broker.Service, account string) {
	if svc.Rotation != "round_robin" || len(svc.Accounts) == 0 {
		return
	}
	for i, acct := range svc.Accounts {
		if acct.ID == account {
			r.rrIndex[serviceKey(vaultID, svc.Name)] = (i + 1) % len(svc.Accounts)
			return
		}
	}
}

func serviceQuotaValue(q *broker.ServiceQuota) broker.ServiceQuota {
	if q == nil {
		return broker.ServiceQuota{}
	}
	return *q
}

func (r *Registry) emitExhaustedAlertLocked(ctx context.Context, vaultID, service string, now time.Time) {
	if r.OnExhausted == nil {
		return
	}
	dayKey, _ := windowKeys(now)
	key := vaultID + "\x00" + service + "\x00" + dayKey
	if r.alerted[key] {
		return
	}
	r.alerted[key] = true
	go r.OnExhausted(context.WithoutCancel(ctx), vaultID, service)
}

func (res *Reservation) Commit(status int) {
	if res == nil || status <= 0 {
		return
	}
	res.reg.mu.Lock()
	defer res.reg.mu.Unlock()
	if res.recorded {
		return
	}
	state := res.reg.counterForLocked(res.key, res.reg.now())
	state.dayCount++
	state.monthCount++
	res.recorded = true
}

func (res *Reservation) AccountID() string {
	if res == nil {
		return ""
	}
	return res.accountID
}

func (res *Reservation) CredentialKey() string {
	if res == nil {
		return ""
	}
	return res.credentialKey
}

func (res *Reservation) Cooldown(d time.Duration) {
	if res == nil {
		return
	}
	if d <= 0 {
		d = time.Minute
	}
	res.reg.mu.Lock()
	defer res.reg.mu.Unlock()
	state := res.reg.limitForLocked(res.key)
	until := res.reg.now().Add(d)
	if until.After(state.cooldownUntil) {
		state.cooldownUntil = until
	}
}

func (res *Reservation) Release() {
	if res == nil {
		return
	}
	res.reg.mu.Lock()
	defer res.reg.mu.Unlock()
	if res.released {
		return
	}
	if state := res.reg.limits[res.key]; state != nil && state.inFlight > 0 {
		state.inFlight--
	}
	res.released = true
}

func WriteDenial(w http.ResponseWriter, d *Decision) {
	retryAfter := int(d.RetryAfter.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Vault-Quota-Exhausted", d.Service)
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"quota_exhausted","service":%q,"reason":%q}`+"\n", d.Service, d.Reason)))
}

func reserveRPM(now time.Time, state *limitState, rpm int) time.Duration {
	if rpm <= 0 {
		return 0
	}
	cutoff := now.Add(-time.Minute)
	kept := state.rpmHits[:0]
	for _, hit := range state.rpmHits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	state.rpmHits = kept
	if len(state.rpmHits) >= rpm {
		return state.rpmHits[0].Add(time.Minute).Sub(now)
	}
	state.rpmHits = append(state.rpmHits, now)
	return 0
}

func accountKey(keys []string) string {
	if len(keys) == 0 || keys[0] == "" {
		return implicitAccount
	}
	return keys[0]
}

func snapshotAccountID(row Snapshot) string {
	if row.AccountID != "" {
		return row.AccountID
	}
	return accountKey(row.CredentialKeys)
}

func quotaKey(vaultID, service, account string) string {
	return vaultID + "\x00" + service + "\x00" + account
}

func serviceKey(vaultID, service string) string {
	return vaultID + "\x00" + service
}

func earlierDecision(current, next *Decision) *Decision {
	if current == nil {
		return next
	}
	if next == nil {
		return current
	}
	if next.RetryAfter < current.RetryAfter {
		return next
	}
	return current
}

func windowKeys(t time.Time) (string, string) {
	utc := t.UTC()
	return utc.Format("2006-01-02"), utc.Format("2006-01")
}

func untilNextDay(t time.Time) time.Duration {
	utc := t.UTC()
	next := time.Date(utc.Year(), utc.Month(), utc.Day()+1, 0, 0, 0, 0, time.UTC)
	return next.Sub(utc)
}

func untilNextMonth(t time.Time) time.Duration {
	utc := t.UTC()
	next := time.Date(utc.Year(), utc.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return next.Sub(utc)
}
