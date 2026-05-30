package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// basit in-memory rate limiter — login endpoint için
// 50 kullanıcı için yeterli, daha fazlası için Redis gerekir

type loginAttempt struct {
	count     int
	firstSeen time.Time
	blockedAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
	// 5 dakikada 10 deneme hakkı, ardından 15 dakika blok
	maxAttempts int
	window      time.Duration
	blockFor    time.Duration
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		attempts:    make(map[string]*loginAttempt),
		maxAttempts: 10,
		window:      5 * time.Minute,
		blockFor:    15 * time.Minute,
	}
	// Eski kayıtları temizle (her 10 dakikada)
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		now := time.Now()
		for ip, a := range rl.attempts {
			if now.Sub(a.firstSeen) > rl.blockFor+rl.window {
				delete(rl.attempts, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) isBlocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.attempts[ip]
	if !ok {
		return false
	}
	now := time.Now()

	// Blok süresi dolmuşsa sıfırla
	if !a.blockedAt.IsZero() && now.Sub(a.blockedAt) > rl.blockFor {
		delete(rl.attempts, ip)
		return false
	}
	// Bloklu mu?
	if !a.blockedAt.IsZero() {
		return true
	}
	// Window dolmuşsa sıfırla
	if now.Sub(a.firstSeen) > rl.window {
		delete(rl.attempts, ip)
		return false
	}
	// Limit aşıldı mı?
	if a.count >= rl.maxAttempts {
		a.blockedAt = now
		return true
	}
	return false
}

func (rl *RateLimiter) record(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.attempts[ip]
	if !ok {
		rl.attempts[ip] = &loginAttempt{count: 1, firstSeen: time.Now()}
		return
	}
	a.count++
}

func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// LoginRateLimit — chi middleware olarak kullan
func (rl *RateLimiter) LoginRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Reverse proxy varsa gerçek IP'yi al
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip = xff
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			ip = xri
		}

		if rl.isBlocked(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Çok fazla başarısız giriş denemesi. 15 dakika bekleyin.",
			})
			return
		}

		// Başarısız girişi kaydet (handler sonrası 401 dönmüşse)
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		if wrapped.status == http.StatusUnauthorized {
			rl.record(ip)
		} else if wrapped.status == http.StatusOK {
			// Başarılı giriş — sayacı sıfırla
			rl.Reset(ip)
		}
	})
}

// responseWriter — status code'u yakala
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
