package v1

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Login rate-limit defaults. Mirror the v1.x web flow knobs so the
// SPA login surface fails the same way the HTML form does. Tests
// override LoginWindow / LoginMaxFailures via the package-level
// vars; in production, cmd/gosidian.main wires them from
// cfg.Webauth.LoginWindow / cfg.Webauth.LoginMaxFailures so a single
// tunable controls both audiences.
var (
	LoginWindow      = 15 * time.Minute
	LoginMaxFailures = 5
)

// loginCleanupEvery determines how often (in failed attempts) the
// limiter sweeps stale entries. Cheap, no goroutine — happens
// in-line on the failing request itself.
const loginCleanupEvery = 100

// loginLimiter is a per-IP failed-login counter. Same shape as the
// v1.x server.loginLimiter but lives here so the api/v1 package
// stays self-contained (importing internal/server back into
// internal/api/v1 would invert the layering).
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	hits     int
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: map[string][]time.Time{}}
}

// allowed reports whether the IP is below the failure threshold for
// the current window. Lazy-prunes stale entries so the map doesn't
// grow unbounded under sustained brute force.
func (l *loginLimiter) allowed(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-LoginWindow)
	recent := l.attempts[ip][:0]
	for _, t := range l.attempts[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	l.attempts[ip] = recent
	return len(recent) < LoginMaxFailures
}

// registerFail records a failed attempt and triggers periodic
// cleanup of stale IPs.
func (l *loginLimiter) registerFail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.attempts[ip] = append(l.attempts[ip], now)
	l.hits++
	if l.hits%loginCleanupEvery == 0 {
		l.sweep(now)
	}
}

// reset clears the IP's failure history. Called after a successful
// login so a typo'd attempt right before doesn't count against the
// next failed login burst.
func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// sweep prunes IPs with no recent attempts. Caller must hold mu.
func (l *loginLimiter) sweep(now time.Time) {
	cutoff := now.Add(-LoginWindow)
	for ip, ts := range l.attempts {
		recent := ts[:0]
		for _, t := range ts {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}
		if len(recent) == 0 {
			delete(l.attempts, ip)
		} else {
			l.attempts[ip] = recent
		}
	}
}

// clientIP extracts the client IP from a request, preferring the
// first entry of X-Forwarded-For when behind a reverse proxy. Falls
// back to RemoteAddr otherwise. Mirrors the v1.x server.clientIP
// shape so XFF parsing is consistent across audiences.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
