// /m/cors/cors.go
package cors

import (
	"net/http"
	"net/url"
	"strings"
)

func Cors(allowedOrigins []string, allowCredentials bool) func(http.Handler) http.Handler {
	orig := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			orig[o] = struct{}{}
		}
	}

	isDevLocalhost := func(origin string) bool {
		if origin == "" { return false }
		u, err := url.Parse(origin)
		if err != nil { return false }
		host := u.Hostname()
		return host == "localhost" || host == "127.0.0.1"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowOrigin := ""

			// libera se a Origin estiver na whitelist "exata"
			if origin != "" {
				if _, ok := orig[origin]; ok {
					allowOrigin = origin
				}
				// libera qualquer localhost/127.0.0.1 (qualquer porta) para DEV
				if allowOrigin == "" && isDevLocalhost(origin) {
					allowOrigin = origin
				}
			}

			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Add("Vary", "Origin")
				if allowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			// evita preflight repetido por 10 min:
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
