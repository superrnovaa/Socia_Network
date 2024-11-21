package middleware

import (
    "net/http"
    "sync"
    "time"
)

type RateLimiter struct {
    requests map[string][]time.Time
    mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
    }
}

func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr

        rl.mu.Lock()
        defer rl.mu.Unlock()

        now := time.Now()
        if len(rl.requests[ip]) >= 100 {
            if now.Sub(rl.requests[ip][0]) < time.Minute {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            rl.requests[ip] = rl.requests[ip][1:]
        }
        rl.requests[ip] = append(rl.requests[ip], now)

        next.ServeHTTP(w, r)
    })
}
