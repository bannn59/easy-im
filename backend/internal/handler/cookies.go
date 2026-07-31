package handler

import "net/http"

// sessionCookieName is the HttpOnly cookie holding the access-token JWT.
const sessionCookieName = "easyim_session"

// CookieConfig controls session-cookie attributes.
type CookieConfig struct {
	// Secure marks the cookie HTTPS-only (production).
	Secure bool
	// Domain optionally scopes the cookie to a parent domain.
	Domain string
}

// setSessionCookie writes the session cookie carrying the JWT.
func setSessionCookie(w http.ResponseWriter, token string, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		Domain:   cfg.Domain,
	})
}

// clearSessionCookie expires the session cookie (logout).
func clearSessionCookie(w http.ResponseWriter, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		Domain:   cfg.Domain,
		MaxAge:   -1,
	})
}

// sessionToken returns the JWT from the session cookie, or "" if absent.
func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
