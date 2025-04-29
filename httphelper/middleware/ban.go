package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lyricat/goutils/httphelper/util"
	"github.com/redis/go-redis/v9"
)

type BanParams struct {
	Rdb            *redis.Client
	MaliciousPaths []string
	RdbKey         string
	IPBlacklist    []string
	ipBlacklistSet map[string]bool
}

func Ban(params BanParams) func(next http.Handler) http.Handler {
	if params.RdbKey == "" {
		params.RdbKey = "ban-%s"
	}
	if params.Rdb == nil {
		panic("Rdb is nil")
	}

	// this middleware is used to ban the ip address if the ip address try to access potential malicious routes
	// example:
	// //\\ .../ .../ .../ .../ .../ .../etc/passwd
	if params.MaliciousPaths == nil {
		params.MaliciousPaths = []string{
			"/etc\\pass",
			"/etc/pass",
			"/etc/shadow",
			"/.env",
			"/error.log",
			"/wp-config.php",
			"/phpunit",
			"/settings.py",
			"/application.properties",
			"php.ini",
			"info.php",
			".htaccess",
			"config.ini",
			"wp-config",
		}
	}

	for _, ip := range params.IPBlacklist {
		params.ipBlacklistSet[ip] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()
			ip := util.GetRemoteIP(r)

			if _, ok := params.ipBlacklistSet[ip]; ok {
				http.Error(w, "", http.StatusNotFound)
				return
			}

			key := fmt.Sprintf(params.RdbKey, ip)

			// check if the ip address is banned
			// if the ip address is in the rdb, it's banned
			ban, err := params.Rdb.Get(ctx, key).Result()
			if err == nil && ban == "1" {
				http.Error(w, "", http.StatusNotFound)
				return
			}

			path := r.URL.Path
			for _, p := range params.MaliciousPaths {
				if strings.Contains(path, p) {
					// add the ip address to the ban list for 24 hour
					params.Rdb.Set(ctx, key, p, time.Hour*24)
					http.Error(w, "", http.StatusNotFound)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
