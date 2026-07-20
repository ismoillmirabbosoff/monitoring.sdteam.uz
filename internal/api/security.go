package api

import (
	"net/http"
	"sync"
	"time"
)

// ipLimiter — IP bo'yicha sirg'aluvchi-oyna (sliding window) rate limiter.
// Xotirada ishlaydi, davriy tozalanadi. Brute force va bitta IP'dan flood'ga qarshi.
type ipLimiter struct {
	mu     sync.Mutex
	hits   map[string][]int64
	max    int
	window int64 // soniya
	lastGC int64
}

func newIPLimiter(max int, window time.Duration) *ipLimiter {
	return &ipLimiter{hits: make(map[string][]int64), max: max, window: int64(window.Seconds())}
}

func (l *ipLimiter) allow(key string) bool {
	now := time.Now().Unix()
	cutoff := now - l.window
	l.mu.Lock()
	defer l.mu.Unlock()

	// davriy tozalash (eski yozuvlarni o'chirish)
	if now-l.lastGC > 120 {
		for k, ts := range l.hits {
			keep := ts[:0]
			for _, t := range ts {
				if t > cutoff {
					keep = append(keep, t)
				}
			}
			if len(keep) == 0 {
				delete(l.hits, k)
			} else {
				l.hits[k] = keep
			}
		}
		l.lastGC = now
	}

	ts := l.hits[key]
	keep := ts[:0]
	for _, t := range ts {
		if t > cutoff {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.max {
		l.hits[key] = keep
		return false
	}
	l.hits[key] = append(keep, now)
	return true
}

// limit — berilgan limiter bilan handlerni o'raydi (429 qaytaradi limit oshsa).
func (s *Server) limit(l *ipLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			writeErr(w, http.StatusTooManyRequests, "juda ko'p urinish — birozdan keyin qayta urinib ko'ring")
			return
		}
		next(w, r)
	}
}

// secure — xavfsizlik headerlari (XSS/clickjacking/MIME himoya) + global (DDoS) rate limit.
func (s *Server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")
		// CSP: hammasi same-origin. Vue :style uchun style'da 'unsafe-inline'.
		// WS backend bridge orqali (brauzer tashqi WS'ga ulanmaydi), audio proxy — self.
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; connect-src 'self'; media-src 'self'; font-src 'self' data:; "+
				"frame-ancestors 'self'; base-uri 'self'; form-action 'self'; object-src 'none'")

		// Global DDoS himoya (bitta IP'dan juda ko'p so'rov)
		if !s.globalLimiter.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "30")
			writeErr(w, http.StatusTooManyRequests, "juda ko'p so'rov")
			return
		}
		// So'rov tanasini cheklash (katta-body DoS'ga qarshi) — o'zgartiruvchi so'rovlar uchun
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2 MB
		}
		next.ServeHTTP(w, r)
	})
}
