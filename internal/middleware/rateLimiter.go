package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	ttl      time.Duration
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastseen time.Time
}

func NewIPRateLimiter(ttl time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		ttl:      ttl,
	}
	go rl.deleteLoop(ttl)
	return rl
}

// deleteLoop 启动后台定时清理协程，删除过期未活跃的限流器条目
// 参数:
//   - interval: 清理间隔
func (rl *IPRateLimiter) deleteLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// 遍历所有限流器条目，比较 lastSeen 与当前时间的差值
		for key, entry := range rl.limiters {
			// 差值超过 interval 的条目将被从 map 中删除
			if time.Since(entry.lastseen) > interval {
				delete(rl.limiters, key)
			}
		}
	}
}

// getLimiter 获取或创建指定 key 的令牌桶限流器
//
// 参数:
//   - key: 限流器唯一标识，通常由 "IP:接口名称" 组成，如 "192.168.1.100:register"
//   - r: 令牌补充速率，例如 rate.Every(20*time.Minute) 表示每 20 分钟产生 1 个令牌
//   - burst: 桶容量，即允许的最大突发请求数
//
// 返回值:
//   - *rate.Limiter: 令牌桶限流器实例，调用方可直接调用 Allow()/Wait() 等方法
func (rl *IPRateLimiter) getLimiter(key string, r rate.Limit, brush int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[key]
	// 若 key 不存在，则创建新的限流器并存入 map
	if !exists {
		limiter := rate.NewLimiter(r, brush)
		entry = &rateLimiterEntry{limiter: limiter, lastseen: time.Now()}
		rl.limiters[key] = entry
	}	
	// 若 key 已存在，则直接复用，避免重复创建
	// 每次调用都更新 lastSeen 时间戳，用于后续过期清理
	entry.lastseen = time.Now()
	return entry.limiter
}	

// RateLimit 获取挂载的 RateLimit 中间件函数
//
// 参数
//   - endPointKey: 接口名称，如 "login"、"register"
//   - r: 令牌补充速率，例如 rate.Every(20*time.Minute) 表示每 20 分钟产生 1 个令牌
//   - burst: 桶容量，即允许的最大突发请求数
// 返回值:
//   - gin.HandlerFunc: 挂载在 router 的处理函数
func (rl *IPRateLimiter) RateLimit(endPointKey string, r rate.Limit, brush int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		key := ip + ":" + endPointKey
		limiter := rl.getLimiter(key, r, brush)

		if !limiter.Allow() {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后重试",
			})
			ctx.Abort()
			return
		}
	}
}
