package auth

import (
	"net/http"
	"time"

	"github.com/fastzet/yt-dlp-webgui/internal/config"
)

// LoginHandler returns an http.Handler that serves the login form (GET)
// and processes login submissions (POST).
func LoginHandler(cfg *config.Config, sm *SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleLoginGet(w, r)
		case http.MethodPost:
			handleLoginPost(w, r, cfg, sm)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// LogoutHandler returns an http.Handler that clears the session cookie
// and redirects to the login page.
func LogoutHandler(sm *SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sm.ClearCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

// RequireAuth is middleware that checks for a valid session cookie.
// Unauthenticated requests are redirected to /login.
func RequireAuth(sm *SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sm.Validate(r); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- internal handlers ---

func handleLoginGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(loginPage("")))
}

func handleLoginPost(w http.ResponseWriter, r *http.Request, cfg *config.Config, sm *SessionManager) {
	// Enforce a small constant delay to slow brute-force attempts.
	// This is applied regardless of outcome so timing does not reveal
	// whether the username or password was wrong.
	defer func() { time.Sleep(300 * time.Millisecond) }()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate username first (constant-time string comparison).
	usernameOK := constantTimeStringEqual(username, cfg.Username)

	// Always run password verification to avoid short-circuiting timing leak.
	passwordErr := VerifyPassword(password, cfg.PasswordHash)

	if !usernameOK || passwordErr != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(loginPage("Invalid username or password.")))
		return
	}

	if err := sm.CreateCookie(w, username); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// constantTimeStringEqual compares two strings in constant time.
func constantTimeStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	eq := 1
	ab := []byte(a)
	bb := []byte(b)
	for i := range ab {
		if ab[i] != bb[i] {
			eq = 0
		}
	}
	return eq == 1
}

// loginPage returns a minimal self-contained HTML login form.
// errMsg is shown below the form when non-empty.
func loginPage(errMsg string) string {
	errHTML := ""
	if errMsg != "" {
		errHTML = `<p class="error">` + errMsg + `</p>`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>yt-dlp webgui — login</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root {
    --bg: #171614; --surface: #1c1b19; --border: #393836;
    --text: #cdccca; --muted: #797876; --primary: #4f98a3;
    --primary-hover: #227f8b; --error: #d163a7;
    --radius: 0.5rem; --font: 'Inter', system-ui, sans-serif;
  }
  body {
    min-height: 100dvh; display: flex; align-items: center;
    justify-content: center; background: var(--bg);
    font-family: var(--font); color: var(--text); padding: 1rem;
  }
  .card {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 2rem; width: 100%;
    max-width: 380px;
  }
  h1 { font-size: 1.125rem; font-weight: 600; margin-bottom: 1.5rem; }
  label { display: block; font-size: 0.875rem; color: var(--muted); margin-bottom: 0.375rem; }
  input {
    width: 100%; padding: 0.625rem 0.75rem; background: var(--bg);
    border: 1px solid var(--border); border-radius: var(--radius);
    color: var(--text); font-size: 1rem; outline: none;
    transition: border-color 180ms ease;
  }
  input:focus { border-color: var(--primary); }
  .field { margin-bottom: 1rem; }
  button {
    width: 100%; padding: 0.625rem; background: var(--primary);
    color: #fff; border: none; border-radius: var(--radius);
    font-size: 1rem; font-weight: 500; cursor: pointer;
    transition: background 180ms ease; margin-top: 0.5rem;
  }
  button:hover { background: var(--primary-hover); }
  .error {
    font-size: 0.875rem; color: var(--error);
    margin-bottom: 1rem; padding: 0.5rem 0.75rem;
    border: 1px solid var(--error); border-radius: var(--radius);
  }
</style>
</head>
<body>
  <div class="card">
    <h1>yt-dlp webgui</h1>
    ` + errHTML + `
    <form method="POST" action="/login" autocomplete="off">
      <div class="field">
        <label for="username">Username</label>
        <input type="text" id="username" name="username"
               autocomplete="username" required autofocus>
      </div>
      <div class="field">
        <label for="password">Password</label>
        <input type="password" id="password" name="password"
               autocomplete="current-password" required>
      </div>
      <button type="submit">Sign in</button>
    </form>
  </div>
</body>
</html>`
}
