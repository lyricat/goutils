package middleware

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/middleware"
	"github.com/lyricat/goutils/httphelper/util"
)

type LoggerParams struct {
	SkippedPaths []string
	SkipMethods  []string
	TraceIPs     []string
	traceIPSet   map[string]bool
}

func Logger(params LoggerParams) func(next http.Handler) http.Handler {
	// don't log following paths
	if params.SkippedPaths == nil {
		// default skipped paths
		params.SkippedPaths = []string{
			"/", "/robots.txt", "/favicon.ico",
			"/_hc", "/hc", "/health",
		}
	}

	skippedPathSet := make(map[string]bool)
	for _, path := range params.SkippedPaths {
		skippedPathSet[path] = true
	}

	if params.SkipMethods == nil {
		params.SkipMethods = []string{
			http.MethodOptions,
		}
	}

	if len(params.TraceIPs) > 0 {
		params.traceIPSet = make(map[string]bool)
		for _, ip := range params.TraceIPs {
			params.traceIPSet[ip] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skippedPathSet[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			if slices.Contains(params.SkipMethods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			ip := util.GetRemoteIP(r)
			if _, ok := params.traceIPSet[ip]; ok {
				slog.Warn("[logger] trace ip", "ip", ip, "path", r.URL.Path, "method", r.Method)
			}
			middleware.Logger(next).ServeHTTP(w, r)
		})
	}
}
