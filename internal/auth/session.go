package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "ytdlp_session"
	// tokenLen is the number of random bytes in the session token.
	tokenLen = 32
)

// Session represents a validated session token and its metadata.
type Session struct {
	Token     string
	Username  string
	ExpiresAt time.Time
}

// SessionManager handles creation and validation of signed session cookies.
type SessionManager struct {
	secret []byte
	ttl    time.Duration
}

// NewSessionManager creates a SessionManager using the given secret and TTL.
func NewSessionManager(secret string, ttl time.Duration) *SessionManager {
	return &SessionManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// CreateCookie generates a new signed session cookie for the given username.
func (sm *SessionManager) CreateCookie(w http.ResponseWriter, username string) error {
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("generating session token: %w", err)
	}

	expiresAt := time.Now().Add(sm.ttl)

	// Payload: token|username|expiry_unix
	payload := fmt.Sprintf("%s|%s|%d", token, username, expiresAt.Unix())
	sig := sm.sign(payload)

	// Cookie value: payload.signature
	cookieVal := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    cookieVal,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// Validate reads and verifies the session cookie from the request.
// Returns the Session on success, or an error if missing, tampered, or expired.
func (sm *SessionManager) Validate(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, errors.New("no session cookie")
	}

	// Split value into payload_b64.signature
	dotIdx := strings.LastIndex(cookie.Value, ".")
	if dotIdx < 0 {
		return nil, errors.New("malformed session cookie: missing signature")
	}

	payloadB64 := cookie.Value[:dotIdx]
	sig := cookie.Value[dotIdx+1:]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, errors.New("malformed session cookie: invalid encoding")
	}
	payload := string(payloadBytes)

	// Constant-time signature check
	expectedSig := sm.sign(payload)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, errors.New("session cookie signature invalid")
	}

	// Parse payload: token|username|expiry_unix
	parts := strings.Split(payload, "|")
	if len(parts) != 3 {
		return nil, errors.New("malformed session cookie: unexpected payload format")
	}

	var expiryUnix int64
	if _, err := fmt.Sscanf(parts[2], "%d", &expiryUnix); err != nil {
		return nil, errors.New("malformed session cookie: invalid expiry")
	}

	expiresAt := time.Unix(expiryUnix, 0)
	if time.Now().After(expiresAt) {
		return nil, errors.New("session expired")
	}

	return &Session{
		Token:     parts[0],
		Username:  parts[1],
		ExpiresAt: expiresAt,
	}, nil
}

// ClearCookie removes the session cookie from the response.
func (sm *SessionManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// --- helpers ---

// sign returns an HMAC-SHA256 hex signature of the payload.
func (sm *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// generateToken returns a cryptographically random URL-safe token string.
func generateToken() (string, error) {
	b := make([]byte, tokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
