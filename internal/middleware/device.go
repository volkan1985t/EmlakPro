package middleware

import (
	"context"
	"net/http"
)

// DeviceAuthMiddleware — saha cihazları için X-Device-Token doğrulaması.
// Bağımlılığı gevşek tutmak için repository'yi doğrudan import etmez;
// dışarıdan bir resolve fonksiyonu alır (token -> user_id, bulundu mu).
type DeviceAuthMiddleware struct {
	resolve func(token string) (int64, bool)
}

func NewDeviceAuth(resolve func(token string) (int64, bool)) *DeviceAuthMiddleware {
	return &DeviceAuthMiddleware{resolve: resolve}
}

// Require — X-Device-Token header'ını doğrular; geçerliyse user_id'yi context'e
// koyar (ContextUserID), geçersizse 401 döner. Yalnızca salt-okunur sorgu
// uçlarında kullanılmalıdır.
func (m *DeviceAuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Device-Token")
		userID, ok := m.resolve(token)
		if !ok {
			jsonError(w, "Gecersiz cihaz token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
