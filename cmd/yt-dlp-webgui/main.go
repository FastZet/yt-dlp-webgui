package main

import (
	"log"
	"net/http"

	"github.com/fastzet/yt-dlp-webgui/internal/auth"
	"github.com/fastzet/yt-dlp-webgui/internal/config"
	"github.com/fastzet/yt-dlp-webgui/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	paths := storage.NewPaths(cfg)
	if err := paths.EnsureDirs(); err != nil {
		log.Fatalf("ensuring storage directories: %v", err)
	}

	sessionManager := auth.NewSessionManager(cfg.SessionSecret, cfg.SessionTTL)

	mux := http.NewServeMux()

	// Public auth routes
	mux.Handle("/login", auth.LoginHandler(cfg, sessionManager))
	mux.Handle("/logout", auth.LogoutHandler(sessionManager))

	// Protected root route
	mux.Handle("/", auth.RequireAuth(sessionManager, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>yt-dlp webgui</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    :root {
      --bg: #171614;
      --surface: #1c1b19;
      --surface-2: #201f1d;
      --border: #393836;
      --text: #cdccca;
      --muted: #797876;
      --primary: #4f98a3;
      --primary-hover: #227f8b;
      --radius: 0.5rem;
      --font: Inter, system-ui, sans-serif;
    }
    body {
      min-height: 100dvh;
      background: var(--bg);
      color: var(--text);
      font-family: var(--font);
      padding: 2rem;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
      margin-bottom: 2rem;
    }
    h1 {
      font-size: 1.5rem;
      font-weight: 600;
    }
    .muted {
      color: var(--muted);
      margin-top: 0.35rem;
    }
    .card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 1.25rem;
      margin-bottom: 1rem;
    }
    textarea {
      width: 100%;
      min-height: 180px;
      resize: vertical;
      background: var(--surface-2);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 0.875rem 1rem;
      font: inherit;
      outline: none;
    }
    textarea:focus {
      border-color: var(--primary);
    }
    .actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 1rem;
      margin-top: 1rem;
      flex-wrap: wrap;
    }
    .btn, button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      border: none;
      border-radius: var(--radius);
      padding: 0.7rem 1rem;
      font: inherit;
      cursor: pointer;
      text-decoration: none;
    }
    .btn-primary {
      background: var(--primary);
      color: white;
    }
    .btn-primary:hover {
      background: var(--primary-hover);
    }
    .btn-secondary {
      background: transparent;
      color: var(--text);
      border: 1px solid var(--border);
    }
    .grid {
      display: grid;
      gap: 1rem;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    }
    code {
      background: var(--surface-2);
      padding: 0.125rem 0.375rem;
      border-radius: 0.375rem;
    }
    form.inline {
      margin: 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="topbar">
      <div>
        <h1>yt-dlp webgui</h1>
        <p class="muted">Bootstrap app is running. Login, config loading, and storage setup are working.</p>
      </div>
      <form class="inline" method="POST" action="/logout">
        <button class="btn btn-secondary" type="submit">Log out</button>
      </form>
    </div>

    <div class="card">
      <p>This is the initial protected control panel placeholder.</p>
      <p class="muted" style="margin-top: 0.75rem;">
        Next steps: add job submission, custom ZIP upload, live logs, expiring download links, and cleanup logic.
      </p>
    </div>

    <div class="grid">
      <div class="card">
        <strong>Command input</strong>
        <p class="muted" style="margin-top: 0.5rem;">Planned as a command-like textarea for yt-dlp arguments.</p>
      </div>
      <div class="card">
        <strong>Runtime mode</strong>
        <p class="muted" style="margin-top: 0.5rem;">Will support official yt-dlp and persistent custom source ZIP runtime.</p>
      </div>
      <div class="card">
        <strong>Delivery</strong>
        <p class="muted" style="margin-top: 0.5rem;">Expiring direct download links with automatic file cleanup.</p>
      </div>
    </div>

    <div class="card">
      <label for="command" style="display:block; margin-bottom:0.75rem;">Future command input area</label>
      <textarea id="command" placeholder="Example: --format mp4 https://www.youtube.com/watch?v=..."></textarea>
      <div class="actions">
        <span class="muted">Not functional yet — this is just the first running shell.</span>
        <button class="btn btn-primary" type="button" disabled>Run download</button>
      </div>
    </div>
  </div>
</body>
</html>`))
	})))

	log.Printf("yt-dlp-webgui listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("starting server: %v", err)
	}
}
