package auth

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter é uma janela fixa em memória. Para um usuário e um processo
// só, isso é o bastante (01-arquitetura.md: sem Redis) — o objetivo é só
// segurar tentativas de força bruta contra /auth/login (03-api.md §8).
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*bucket
	max      int
	window   time.Duration
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{attempts: make(map[string]*bucket), max: max, window: window}
}

func (l *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.attempts[key]
	if !ok || now.After(b.resetAt) {
		l.attempts[key] = &bucket{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}
	if b.count >= l.max {
		return false, time.Until(b.resetAt)
	}
	b.count++
	return true, 0
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
