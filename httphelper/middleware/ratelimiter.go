package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lyricat/goutils/httphelper/util"
	"github.com/redis/go-redis/v9"
)

type (
	RateLimiterParams struct {
		Rdb             *redis.Client
		RdbKey          string
		RateLimitConfig RateLimitConfig
	}

	RateLimitConfig struct {
		Threshold       int64   `yaml:"threshold"`
		Period          string  `yaml:"period"`
		GlobalRateLimit Global  `yaml:"global"`
		RouteRateLimits []Route `yaml:"routes"`
	}

	Global struct {
		Threshold int64  `yaml:"threshold"`
		Period    string `yaml:"period"`
	}

	Route struct {
		Method    string `yaml:"method"`
		Prefix    string `yaml:"prefix"`
		Threshold int64  `yaml:"threshold"`
		Period    string `yaml:"period"`
	}
)

type ctxKey int

const RouteNotFoundContextKey ctxKey = iota

func RateLimiter(params RateLimiterParams) func(next http.Handler) http.Handler {
	if params.RdbKey == "" {
		params.RdbKey = "limiter-%s:%s"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if params.Rdb == nil {
				panic("Rdb is nil")
			}

			ctx := r.Context()

			ip := util.GetRemoteIP(r)

			var err error

			defaultPeriod := time.Minute
			defaultThreshold := int64(512)
			if params.RateLimitConfig.Threshold > 0 {
				defaultThreshold = params.RateLimitConfig.Threshold
			}
			if params.RateLimitConfig.Period != "" {
				defaultPeriod, err = time.ParseDuration(params.RateLimitConfig.Period)
				if err != nil {
					defaultPeriod = time.Minute
				}
			}

			// for global rate limit, 1000 req/min
			path := ip
			period, err := time.ParseDuration(params.RateLimitConfig.GlobalRateLimit.Period)
			if err != nil {
				slog.Warn("[goutils] time.ParseDuration failed, use default value", "error", err, "period", params.RateLimitConfig.GlobalRateLimit.Period)
				period = defaultPeriod
			}
			if count, err := hit(ctx, params.Rdb, "ip", path, period, params.RateLimitConfig.GlobalRateLimit.Threshold); err != nil {
				slog.Warn("[goutils] limiter.GlobalHit", "error", err, "ip", ip, "url", r.URL.Path, "period", period, "count", count)
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			// for api rate limit, 512 req/min
			apiPath := r.URL.Path
			path = fmt.Sprintf("%s>%s", ip, apiPath)
			thd := defaultThreshold
			period = defaultPeriod

			// route specific rate limit
			for _, route := range params.RateLimitConfig.RouteRateLimits {
				if route.Method != r.Method || !strings.HasPrefix(apiPath, route.Prefix) {
					continue
				}

				if route.Threshold > 0 {
					thd = route.Threshold
					period, err = time.ParseDuration(route.Period)
					if err != nil {
						slog.Warn("[goutils] time.ParseDuration failed, use default value", "error", err, "period", route.Period, "route.method", route.Method, "route.prefix", route.Prefix)
						period = defaultPeriod
						break
					}
				}
			}

			if count, err := hit(ctx, params.Rdb, "ip", path, period, thd); err != nil {
				slog.Warn("[goutils] limiter.Hit", "error", err, "ip", ip, "url", r.URL.Path, "period", period, "count", count)
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hit(ctx context.Context, rdb *redis.Client, category, path string, expiry time.Duration, maxHit int64) (int64, error) {
	key := fmt.Sprintf("limiter-%s:%s", category, path)
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	_, err = rdb.Expire(ctx, key, expiry).Result()
	if err != nil {
		return 0, err
	}

	if count >= maxHit {
		_, err := rdb.Expire(ctx, key, expiry).Result()
		if err != nil {
			return 0, err
		}
		return count, errors.New("too many requests")
	}

	return 0, nil
}
