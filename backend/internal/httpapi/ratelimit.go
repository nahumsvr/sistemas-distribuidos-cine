package httpapi

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit is a token bucket: one request every Every, bursts up to Burst.
type RateLimit struct {
	Every time.Duration
	Burst int
}

// DefaultAuthRateLimit allows ~5 login/register attempts per minute per IP.
var DefaultAuthRateLimit = RateLimit{Every: 12 * time.Second, Burst: 5}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipLimiter keeps one bucket per client IP, in memory. Good enough for a
// single API instance; several instances would need a shared store (Redis).
type ipLimiter struct {
	cfg       RateLimit
	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

func newIPLimiter(cfg RateLimit) *ipLimiter {
	return &ipLimiter{cfg: cfg, visitors: map[string]*visitor{}, lastSweep: time.Now()}
}

func (l *ipLimiter) reserve(ip string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastSweep) > time.Minute {
		for k, v := range l.visitors {
			if now.Sub(v.lastSeen) > 10*time.Minute {
				delete(l.visitors, k)
			}
		}
		l.lastSweep = now
	}

	v, found := l.visitors[ip]
	if !found {
		v = &visitor{limiter: rate.NewLimiter(rate.Every(l.cfg.Every), l.cfg.Burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = now

	res := v.limiter.ReserveN(now, 1)
	if delay := res.DelayFrom(now); delay > 0 {
		res.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// clientIP uses the TCP peer address. X-Forwarded-For is ignored on purpose:
// there is no trusted proxy in front of the API, so that header is
// client-controlled and would let attackers bypass the limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *ipLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, retry := l.reserve(clientIP(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many attempts, try again later")
			return
		}
		next(w, r)
	}
}
