package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestBan creates a new Ban instance with a real Redis client for testing.
// It reads Redis address and password from environment variables:
// REDIS_ADDR (e.g., "localhost:6379")
// REDIS_PASSWORD
func newTestBan(t *testing.T, initialBlacklist []string) *Ban {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default for local testing
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	dbStr := os.Getenv("REDIS_DB")
	fmt.Printf("dbStr: %v\n", dbStr)
	fmt.Printf("redisAddr: %v\n", redisAddr)
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		t.Fatalf("failed to convert REDIS_DB to int: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       db, // Use default DB
	})

	// Check if connection is alive
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("could not connect to redis at %s: %v. Set REDIS_ADDR and REDIS_PASSWORD env variables.", redisAddr, err)
	}

	// Cleanup: flush redis db after test
	t.Cleanup(func() {
		if err := rdb.FlushDB(context.Background()).Err(); err != nil {
			t.Fatalf("failed to flush redis db: %v", err)
		}
		rdb.Close()
	})

	// Flush DB before the test starts to ensure clean state
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("failed to flush redis db before test: %v", err)
	}

	ban, err := NewBan(BanParams{
		Rdb:         rdb,
		IPBlacklist: initialBlacklist,
	})
	if err != nil {
		t.Fatalf("failed to create Ban instance: %v", err)
	}

	return ban
}

func TestBanMiddleware(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("allows requests from non-banned IPs", func(t *testing.T) {
		ban := newTestBan(t, nil)

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("blocks requests from manually banned IPs", func(t *testing.T) {
		ban := newTestBan(t, nil)

		if err := ban.BanIP("1.2.3.4"); err != nil {
			t.Fatalf("failed to ban IP: %v", err)
		}

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("allows requests after unbanning an IP", func(t *testing.T) {
		ban := newTestBan(t, nil)

		if err := ban.BanIP("1.2.3.4"); err != nil {
			t.Fatalf("failed to ban IP: %v", err)
		}

		if err := ban.UnbanIP("1.2.3.4"); err != nil {
			t.Fatalf("failed to unban IP: %v", err)
		}

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("blocks requests from IPs in a banned CIDR range", func(t *testing.T) {
		ban := newTestBan(t, nil)

		if err := ban.BanIP("1.2.3.0/24"); err != nil {
			t.Fatalf("failed to ban IP: %v", err)
		}

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.100:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("blocks requests from IPs matching a wildcard", func(t *testing.T) {
		ban := newTestBan(t, nil)

		if err := ban.BanIP("2.3.*.*"); err != nil {
			t.Fatalf("failed to ban IP: %v", err)
		}

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "2.3.4.5:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("bans and blocks ip for accessing malicious path", func(t *testing.T) {
		ban := newTestBan(t, nil)

		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/.env", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}

		// Wait for the ban to be processed
		time.Sleep(100 * time.Millisecond)

		// Check if the IP is in the Redis blacklist
		isMember, err := ban.rdb.SIsMember(context.Background(), ban.rdbBlacklistKey, "10.0.0.1").Result()
		if err != nil {
			t.Fatalf("failed to check redis set member: %v", err)
		}
		if !isMember {
			t.Error("IP should be added to the Redis blacklist")
		}

		// Make another request from the same IP to a normal path
		reqNormal := httptest.NewRequest("GET", "/", nil)
		reqNormal.RemoteAddr = "10.0.0.1:1234"
		rrNormal := httptest.NewRecorder()
		handler.ServeHTTP(rrNormal, reqNormal)

		if rrNormal.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rrNormal.Code)
		}
	})

	t.Run("uses initial blacklist provided at creation", func(t *testing.T) {
		initialBlacklist := []string{"4.4.4.4", "5.5.0.0/16"}
		ban := newTestBan(t, initialBlacklist)

		handler := ban.Handler(baseHandler)

		// Test direct IP ban
		req1 := httptest.NewRequest("GET", "/", nil)
		req1.RemoteAddr = "4.4.4.4:1234"
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr1.Code)
		}

		// Test CIDR ban
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.RemoteAddr = "5.5.6.7:1234"
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr2.Code)
		}
	})

	t.Run("SetBlacklist correctly overrides the existing blacklist", func(t *testing.T) {
		ban := newTestBan(t, []string{"1.1.1.1"})

		if err := ban.SetBlacklist([]string{"2.2.2.2", "3.3.3.0/24"}); err != nil {
			t.Fatalf("failed to set blacklist: %v", err)
		}

		handler := ban.Handler(baseHandler)

		// Old banned IP should be allowed
		req1 := httptest.NewRequest("GET", "/", nil)
		req1.RemoteAddr = "1.1.1.1:1234"
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr1.Code)
		}

		// New IPs should be blocked
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.RemoteAddr = "2.2.2.2:1234"
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr2.Code)
		}

		req3 := httptest.NewRequest("GET", "/", nil)
		req3.RemoteAddr = "3.3.3.10:1234"
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		if rr3.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr3.Code)
		}
	})

	t.Run("IsBannedIP works correctly", func(t *testing.T) {
		ban := newTestBan(t, []string{"8.8.8.8"})

		if !ban.IsBannedIP("8.8.8.8") {
			t.Error("expected 8.8.8.8 to be banned")
		}
		if ban.IsBannedIP("8.8.8.9") {
			t.Error("expected 8.8.8.9 not to be banned")
		}

		if err := ban.BanIP("9.9.9.0/24"); err != nil {
			t.Fatalf("failed to ban ip: %v", err)
		}
		if !ban.IsBannedIP("9.9.9.100") {
			t.Error("expected 9.9.9.100 to be banned")
		}
		if ban.IsBannedIP("9.9.10.1") {
			t.Error("expected 9.9.10.1 not to be banned")
		}
	})

	t.Run("onBlacklistAdd callback is triggered", func(t *testing.T) {
		var receivedIP, receivedReason string
		callbackCalled := make(chan bool, 1)

		onAdd := func(ip, reason string) {
			receivedIP = ip
			receivedReason = reason
			callbackCalled <- true
		}

		ban := newTestBan(t, nil)
		ban.onBlacklistAdd = onAdd

		// Test manual ban
		if err := ban.BanIP("7.7.7.7"); err != nil {
			t.Fatalf("failed to ban ip: %v", err)
		}

		select {
		case <-callbackCalled:
			if receivedIP != "7.7.7.7" {
				t.Errorf("expected ip 7.7.7.7, got %s", receivedIP)
			}
			if receivedReason != BanReasonManual {
				t.Errorf("expected reason %s, got %s", BanReasonManual, receivedReason)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("onBlacklistAdd callback was not called for manual ban")
		}

		// Test malicious path ban
		handler := ban.Handler(baseHandler)
		req := httptest.NewRequest("GET", "/wp-config.php", nil)
		req.RemoteAddr = "7.7.7.8:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}

		select {
		case <-callbackCalled:
			if receivedIP != "7.7.7.8" {
				t.Errorf("expected ip 7.7.7.8, got %s", receivedIP)
			}
			if receivedReason != BanReasonMaliciousPath {
				t.Errorf("expected reason %s, got %s", BanReasonMaliciousPath, receivedReason)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("onBlacklistAdd callback was not called for malicious path ban")
		}

	})
}
