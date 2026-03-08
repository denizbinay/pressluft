package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"pressluft/internal/auth"
)

type HandlerOptions struct {
	Authenticator   auth.Authenticator
	AuthService     *auth.Service
	ActivityStore   ActivityEmitter
	Logger          loggerLike
	IsDev           bool
	ControlPlaneURL string
}

type ActivityEmitter interface {
	Emit(ctx context.Context, in any) (any, error)
}

type loggerLike interface {
	Warn(string, ...any)
}

func withOperatorAuth(next http.Handler, authenticator auth.Authenticator) http.Handler {
	if authenticator == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := authenticator.Authenticate(r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.ContextWithActor(r.Context(), actor)))
	})
}

func withOptionalActor(next http.Handler, authenticator auth.Authenticator) http.Handler {
	if authenticator == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := authenticator.Authenticate(r)
		if err != nil && !errors.Is(err, auth.ErrUnauthenticated) {
			respondError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if actor.IsAuthenticated() {
			r = r.WithContext(auth.ContextWithActor(r.Context(), actor))
		}
		next.ServeHTTP(w, r)
	})
}

func withAuthorization(next http.Handler, allow func(auth.Actor) bool) http.Handler {
	if allow == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := auth.ActorFromContext(r.Context())
		if !allow(actor) {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler, isDev bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if isDev {
			w.Header().Set("X-Pressluft-Dev-Mode", "insecure")
		}
		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (l *rateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	values := l.windows[key][:0]
	for _, ts := range l.windows[key] {
		if now.Sub(ts) < l.window {
			values = append(values, ts)
		}
	}
	if len(values) >= l.limit {
		l.windows[key] = values
		return false
	}
	l.windows[key] = append(values, now)
	return true
}

func withRateLimit(next http.Handler, limiter *rateLimiter, prefix string) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := prefix + ":" + requestRateLimitKey(r)
		if !limiter.Allow(key) {
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestRateLimitKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
