package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

func GenerateCSRFToken(secret, sessionID string) string {
	window := time.Now().Unix() / 3600
	return computeCSRF(secret, sessionID, window)
}
func ValidateCSRFToken(secret, sessionID, token string) bool {
	window := time.Now().Unix() / 3600
	return token == computeCSRF(secret, sessionID, window) ||
		token == computeCSRF(secret, sessionID, window-1)
}
func computeCSRF(secret, sessionID string, window int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	mac.Write([]byte{byte(window >> 24), byte(window >> 16), byte(window >> 8), byte(window)})
	return hex.EncodeToString(mac.Sum(nil))[:32]
}
func CSRFTokenFromRequest(r *http.Request, secret string) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return GenerateCSRFToken(secret, cookie.Value)
}
