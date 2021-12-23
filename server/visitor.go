package server

import (
	"golang.org/x/time/rate"
	"heckel.io/ntfy/util"
	"sync"
	"time"
)

const (
	// visitorExpungeAfter defines how long a visitor is active before it is removed from memory. This number
	// has to be very high to prevent e-mail abuse, but it doesn't really affect the other limits anyway, since
	// they are replenished faster (typically).
	visitorExpungeAfter = 24 * time.Hour
)

// visitor represents an API user, and its associated rate.Limiter used for rate limiting
type visitor struct {
	config        *Config
	requests      *rate.Limiter
	emails        *rate.Limiter
	subscriptions *util.Limiter
	seen          time.Time
	mu            sync.Mutex
}

func newVisitor(conf *Config) *visitor {
	return &visitor{
		config:        conf,
		requests:      rate.NewLimiter(rate.Every(conf.VisitorRequestLimitReplenish), conf.VisitorRequestLimitBurst),
		emails:        rate.NewLimiter(rate.Every(conf.VisitorEmailLimitReplenish), conf.VisitorEmailLimitBurst),
		subscriptions: util.NewLimiter(int64(conf.VisitorSubscriptionLimit)),
		seen:          time.Now(),
	}
}

func (v *visitor) RequestAllowed() error {
	if !v.requests.Allow() {
		return errHTTPTooManyRequests
	}
	return nil
}

func (v *visitor) EmailAllowed() error {
	if !v.emails.Allow() {
		return errHTTPTooManyRequests
	}
	return nil
}

func (v *visitor) AddSubscription() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.subscriptions.Add(1); err != nil {
		return errHTTPTooManyRequests
	}
	return nil
}

func (v *visitor) RemoveSubscription() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.subscriptions.Sub(1)
}

func (v *visitor) Keepalive() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.seen = time.Now()
}

func (v *visitor) Stale() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return time.Since(v.seen) > visitorExpungeAfter
}
