package handlers

import (
	"net/http"
	"strings"

	"github.com/spike-events/spike-broker/v2/pkg/service"
	"github.com/spike-events/spike-broker/v2/pkg/utils"
)

// AuthMiddleware middleware de autenticação oauth.v3
func AuthMiddleware(oauth ...*service.AuthRid) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, auth := range oauth {
				if strings.HasPrefix(r.URL.Path, "/ws") {
					break
				}
				request := strings.Split(r.URL.Path, "/")
				if len(request) <= 2 || request[2] != auth.Service {
					continue
				}

				validBearer := true
				for _, p := range auth.Patterns {
					if p.Auth() {
						continue
					}

					/* Handle Public endpoints */
					endpoint := "/api/" + strings.ReplaceAll(p.EndpointNoMethod(), ".", "/")
					if r.URL.Path == endpoint && r.Method == p.Method {
						validBearer = false
						break
					}

					if !strings.Contains(p.EndpointNoMethod(), "$") {
						continue
					}

					partsEndpoint := strings.Split(endpoint, "/")
					partsRequest := strings.Split(r.URL.Path, "/")
					if len(partsEndpoint) != len(partsRequest) {
						continue
					}

					for i := range partsRequest {
						if strings.HasPrefix(partsEndpoint[i], "$") {
							partsRequest[i] = partsEndpoint[i]
							continue
						}
						if partsEndpoint[i] != partsRequest[i] {
							break
						}
					}
					if strings.Join(partsRequest, ".") == strings.Join(partsEndpoint, ".") {
						validBearer = false
						break
					}
				}

				if validBearer {
					rawToken, ok := utils.GetBearer(r)
					if !ok {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					var processedToken string
					processedToken, ok = auth.Auth.ValidateToken(rawToken)
					if !ok {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					r.Header.Set("token", processedToken)
				} else {
					rawToken, _ := utils.GetBearer(r)
					r.Header.Set("token", rawToken)
				}
				break
			}
			next.ServeHTTP(w, r)
		})
	}
}
