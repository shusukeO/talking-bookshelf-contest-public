package middleware

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter manages per-IP rate limiting
type IPRateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		rate:  r,
		burst: burst,
	}
}

// GetLimiter returns the rate limiter for a given IP
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	limiter, exists := l.limiters.Load(ip)
	if !exists {
		newLimiter := rate.NewLimiter(l.rate, l.burst)
		l.limiters.Store(ip, newLimiter)
		return newLimiter
	}
	return limiter.(*rate.Limiter)
}

// DailyQuota manages global daily request quota
type DailyQuota struct {
	count   int64
	limit   int64
	resetAt time.Time
	mu      sync.Mutex
}

// NewDailyQuota creates a new daily quota manager
func NewDailyQuota(limit int64) *DailyQuota {
	return &DailyQuota{
		limit:   limit,
		resetAt: nextMidnightPT(),
	}
}

// Allow checks if a request is allowed and increments the counter
func (q *DailyQuota) Allow() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if we need to reset
	if time.Now().After(q.resetAt) {
		log.Printf("[QUOTA] Daily quota reset. Previous count: %d", q.count)
		q.count = 0
		q.resetAt = nextMidnightPT()
	}

	if q.count >= q.limit {
		return false
	}
	q.count++
	return true
}

// Remaining returns the remaining quota
func (q *DailyQuota) Remaining() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.limit - q.count
}

// Count returns the current count
func (q *DailyQuota) Count() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

// nextMidnightPT returns the next midnight in Pacific Time (Gemini API reset time)
func nextMidnightPT() time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Fallback to UTC if timezone not found
		loc = time.UTC
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
//
// 実装は公開リポジトリから省略しています。
// 以下の2段階レート制限を適用:
// 1. グローバル日次クォータ（DailyQuota）の確認 → 超過時は 429 + Retry-After
// 2. IP単位レート制限（IPRateLimiter）の確認 → 超過時は 429 + Retry-After
// レスポンスはチャットUI互換の JSON 形式（response, emotion, suggestions, code）
func RateLimitMiddleware(ipLimiter *IPRateLimiter, quota *DailyQuota) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Rate limiting logic omitted from public repository.
		c.Next()
	}
}
