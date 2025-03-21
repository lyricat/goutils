package middleware

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/middleware"
)

type LoggerParams struct {
	SkippedPaths []string
	SkipMethods  []string
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
			middleware.Logger(next).ServeHTTP(w, r)
		})
	}
}
