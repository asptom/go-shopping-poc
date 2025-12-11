package cors

import (
	"net/http"
	"strings"

	"go-shopping-poc/internal/platform/config"
)

// New returns a Chi-compatible middleware function that applies CORS
// according to values from the provided config.
func New(cfg *config.Config) func(http.Handler) http.Handler {
	origins := parseList(cfg.GetCORSAllowedOrigins(), ",")
	methods := cfg.GetCORSAllowedMethods()
	if methods == "" {
		methods = "GET,POST,PUT,DELETE,OPTIONS"
	}
	headers := cfg.GetCORSAllowedHeaders()
	if headers == "" {
		headers = "Content-Type,Authorization"
	}
	allowCreds := cfg.GetCORSAllowCredentials()
	maxAge := cfg.GetCORSMaxAge()
	if maxAge == "" {
		maxAge = "3600"
	}

	// normalize origins list for quick checks
	allowAnyOrigin := false
	trimmedOrigins := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "*" {
			allowAnyOrigin = true
			break
		}
		if o != "" {
			trimmedOrigins = append(trimmedOrigins, o)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Access-Control-Allow-Origin
			if allowAnyOrigin {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && originAllowed(origin, trimmedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// other headers
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if allowCreds {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			// expose common headers if needed (optional)
			w.Header().Set("Access-Control-Max-Age", maxAge)

			// handle preflight
			if r.Method == http.MethodOptions {
				// Some browsers require Content-Length for 204/200 responses; use 200 OK
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// parseList splits a comma-separated list and trims entries
func parseList(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func originAllowed(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
		// allow simple wildcard suffix e.g. https://*.example.com
		if strings.HasPrefix(a, "*.") {
			// match hostname suffix
			if strings.HasSuffix(origin, a[1:]) {
				return true
			}
		}
	}
	return false
}
