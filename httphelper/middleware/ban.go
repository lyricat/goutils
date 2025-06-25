package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lyricat/goutils/httphelper/util"
	"github.com/redis/go-redis/v9"
)

const (
	BanReasonManual        = "manual"
	BanReasonMaliciousPath = "malicious_path"
)

type Ban struct {
	rdb             *redis.Client
	maliciousPaths  []string
	rdbKey          string
	rdbBlacklistKey string
	onBlacklistAdd  func(ip string, reason string)
}

type BanParams struct {
	Rdb             *redis.Client
	MaliciousPaths  []string
	RdbKey          string
	RdbBlacklistKey string
	IPBlacklist     []string
	OnBlacklistAdd  func(ip string, reason string)
}

func NewBan(params BanParams) (*Ban, error) {
	if params.RdbKey == "" {
		params.RdbKey = "ban-%s"
	}
	if params.Rdb == nil {
		panic("Rdb is nil")
	}

	if params.RdbBlacklistKey == "" {
		params.RdbBlacklistKey = "ip_blacklist"
	}

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

	b := &Ban{
		rdb:             params.Rdb,
		maliciousPaths:  params.MaliciousPaths,
		rdbKey:          params.RdbKey,
		rdbBlacklistKey: params.RdbBlacklistKey,
		onBlacklistAdd:  params.OnBlacklistAdd,
	}

	if len(params.IPBlacklist) > 0 {
		if err := b.SetBlacklist(params.IPBlacklist); err != nil {
			return nil, fmt.Errorf("failed to set initial blacklist: %w", err)
		}
	}

	return b, nil
}

func (b *Ban) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		ipStr := util.GetRemoteIP(r)
		ip := net.ParseIP(ipStr)

		if b.isBlacklisted(ip, ipStr) {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		key := fmt.Sprintf(b.rdbKey, ipStr)

		ban, err := b.rdb.Get(ctx, key).Result()
		if err == nil && ban == "1" {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		path := r.URL.Path
		for _, p := range b.maliciousPaths {
			if strings.Contains(path, p) {
				b.rdb.Set(ctx, key, "1", time.Hour*4320)
				addedCount, err := b.rdb.SAdd(ctx, b.rdbBlacklistKey, ipStr).Result()
				if err == nil && addedCount > 0 {
					if b.onBlacklistAdd != nil {
						go b.onBlacklistAdd(ipStr, BanReasonMaliciousPath)
					}
				}
				http.Error(w, "", http.StatusNotFound)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (b *Ban) SetBlacklist(blacklist []string) error {
	ctx := context.Background()
	pipe := b.rdb.Pipeline()
	pipe.Del(ctx, b.rdbBlacklistKey)
	if len(blacklist) > 0 {
		// Convert []string to []interface{}
		s := make([]interface{}, len(blacklist))
		for i, v := range blacklist {
			s[i] = v
		}
		pipe.SAdd(ctx, b.rdbBlacklistKey, s...)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update blacklist in redis: %w", err)
	}

	return nil
}

func (b *Ban) BanIP(ip string) error {
	ctx := context.Background()
	addedCount, err := b.rdb.SAdd(ctx, b.rdbBlacklistKey, ip).Result()
	if err != nil {
		return fmt.Errorf("failed to add ip to blacklist in redis: %w", err)
	}

	if addedCount > 0 && b.onBlacklistAdd != nil {
		go b.onBlacklistAdd(ip, BanReasonManual)
	}

	return nil
}

func (b *Ban) UnbanIP(ipStr string) error {
	ctx := context.Background()
	if err := b.rdb.SRem(ctx, b.rdbBlacklistKey, ipStr).Err(); err != nil {
		return fmt.Errorf("failed to remove ip from blacklist in redis: %w", err)
	}

	return nil
}

func (b *Ban) IsBannedIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return b.isBlacklisted(ip, ipStr)
}

func (b *Ban) isBlacklisted(ip net.IP, ipStr string) bool {
	ctx := context.Background()
	blacklist, err := b.rdb.SMembers(ctx, b.rdbBlacklistKey).Result()
	if err != nil {
		return false
	}

	for _, entry := range blacklist {
		if ipStr == entry {
			return true
		}
		if ip != nil {
			if _, ipNet, err := parseIPOrCIDR(entry); err == nil {
				if ipNet != nil && ipNet.Contains(ip) {
					return true
				}
			} else if ipNet, err := parseWildcardIP(entry); err == nil {
				if ipNet.Contains(ip) {
					return true
				}
			}
		}
	}

	return false
}

func parseIPOrCIDR(s string) (net.IP, *net.IPNet, error) {
	if !strings.Contains(s, "/") {
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, nil, fmt.Errorf("invalid IP address format")
		}
		return ip, nil, nil
	}
	_, ipNet, err := net.ParseCIDR(s)
	return nil, ipNet, err
}

func parseWildcardIP(s string) (*net.IPNet, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid wildcard IP format")
	}

	ipParts := make([]string, 4)
	maskLen := 0
	wildcardStarted := false

	for i, part := range parts {
		if part == "*" {
			wildcardStarted = true
			ipParts[i] = "0"
		} else {
			if wildcardStarted {
				return nil, fmt.Errorf("wildcard `*` can only appear at the end")
			}
			ipParts[i] = part
			maskLen += 8
		}
	}

	cidr := fmt.Sprintf("%s/%d", strings.Join(ipParts, "."), maskLen)
	_, ipNet, err := net.ParseCIDR(cidr)
	return ipNet, err
}
