package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/auth"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ah := r.Header.Get("Authorization")
		if ah == "" || !strings.HasPrefix(ah, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(ah, "Bearer ")
		uid, err := auth.VerifyToken(token)
		if err != nil {
			log.Printf("auth failed: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Header.Set("X-User-ID", uid)
		next.ServeHTTP(w, r)
	})
}
