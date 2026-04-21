package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"aba-pocket/internal/models"
	"aba-pocket/internal/repository"
)

const (
	SessionCookieName = "aba_session"
	SessionDuration   = 24 * time.Hour
	contextKeyUser    = contextKey("user")

	adminLoginPath = "/admin/login"
)

type contextKey string

func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SetSessionCookie(w http.ResponseWriter, sessionID string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
	})
}

// Middleware validates the session cookie and injects the user into the request context.
func Middleware(repos *repository.Repositories, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Redirect(w, r, adminLoginPath, http.StatusSeeOther)
				return
			}

			session, err := repos.Users.GetSession(r.Context(), cookie.Value)
			if err != nil {
				ClearSessionCookie(w, secure)
				http.Redirect(w, r, adminLoginPath, http.StatusSeeOther)
				return
			}

			user, err := repos.Users.GetByID(r.Context(), session.UserID)
			if err != nil || !user.IsActive {
				ClearSessionCookie(w, secure)
				http.Redirect(w, r, adminLoginPath, http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(contextKeyUser).(*models.User)
	return u
}
