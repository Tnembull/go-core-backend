package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Tnembull/go-core-backend/pkg/response"
	"golang.org/x/time/rate"
)

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*clientLimiter
	rps      rate.Limit
	burst    int
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewIPRateLimiter(rps float64, burst int) *ipRateLimiter {
	i := &ipRateLimiter{
		limiters: make(map[string]*clientLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}

	// Background cleanup of stale IP entries
	go i.cleanupStaleClients()

	return i
}

func RateLimiter(rps float64, burst int) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(rps, burst)
	return limiter.Middleware
}

func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	cl, exists := i.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(i.rps, i.burst)
		i.limiters[ip] = &clientLimiter{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	cl.lastSeen = time.Now()
	return cl.limiter
}

func (i *ipRateLimiter) cleanupStaleClients() {
	for {
		time.Sleep(3 * time.Minute)
		i.mu.Lock()
		for ip, cl := range i.limiters {
			if time.Since(cl.lastSeen) > 5*time.Minute {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

func (i *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP (handle proxy headers if present)
		ip := r.Header.Get("CF-Connecting-IP")
		if ip == "" {
			ip = r.Header.Get("X-Forwarded-For")
		}
		if ip == "" {
			ip = strings.Split(r.RemoteAddr, ":")[0]
		} else {
			ip = strings.Split(ip, ",")[0]
		}

		limiter := i.getLimiter(ip)
		if !limiter.Allow() {
			traceID := GetRequestID(r.Context())
			response.Error(w, http.StatusTooManyRequests, "rate limit exceeded: please slow down your requests", traceID)
			return
		}

		next.ServeHTTP(w, r)
	})
}
