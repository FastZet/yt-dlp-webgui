# yt-dlp-webgui

A small self-hosted web interface for `yt-dlp` that accepts a command-like input, runs the download on a personal Ubuntu VPS, stores the resulting file temporarily, and returns an expiring direct download link.

This repository is being built for a very specific use case:

- personal/self-hosted deployment
- lightweight resource usage
- a single-user control panel protected by username/password
- temporary public download links for finished files
- support for either the official `yt-dlp` runtime or a user-supplied modified `yt-dlp` source ZIP
- easy deployment behind Traefik on an existing VPS

The project is intentionally **not** trying to become a multi-tenant public downloader, a media library manager, or a full-featured YouTube clone. The goal is a practical tool that does one job cleanly.

---

## Why This Exists

`yt-dlp` is already an excellent command-line downloader, and the upstream project is a feature-rich audio/video downloader with support for thousands of sites. The problem this repository solves is not missing downloader capability; it is missing convenience for a browser-first workflow on a self-hosted VPS.

The intended flow is simple:

1. Open a web page.
2. Paste a `yt-dlp`-style command with flags and URL(s).
3. Let the VPS run the job.
4. Receive a direct download URL for the completed file.
5. Let the server expire and remove the file automatically.

This is useful when:

- the user already has a domain, reverse proxy, and VPS
- the user wants browser convenience without giving up command-level control
- downloads should happen on the server, not on the local device
- files should not stay on disk forever
- modified/forked `yt-dlp` code must be runnable without rebuilding a binary

---

## Project Goals

### Primary goals

- Provide a single-page browser UI for running `yt-dlp` jobs.
- Accept a command-like input instead of forcing a restrictive form-only UI.
- Save output files to the VPS temporarily.
- Generate a public expiring direct link for the output file.
- Delete files automatically after expiry or after successful transfer completion.
- Support two execution backends:
  - official `yt-dlp`
  - custom uploaded `yt-dlp` repository ZIP extracted and run from source
- Keep the service lightweight, easy to audit, and easy to deploy.

### Non-goals

- No public anonymous multi-user platform.
- No permanent file hosting.
- No database-heavy architecture unless clearly necessary later.
- No heavy SPA frontend framework unless the UI grows beyond the needs of a single-page tool.
- No shell execution of raw user input.

---

## Upstream Assumptions About yt-dlp

The design of this project depends on a few upstream facts about `yt-dlp`:

- The upstream repository is a Python project and contains a `yt_dlp/` source tree in the repo itself.
- The upstream project documents executable release artifacts for Linux/BSD and other platforms, including a platform-independent Unix binary and source distributions.
- The upstream project supports Python 3.10+ and strongly recommends `ffmpeg`/`ffprobe` for important download and post-processing workflows.
- The upstream documentation also covers plugins and embedding, which confirms that the tool is designed to be used in more than one runtime style beyond a single packaged executable.

Those points are exactly why this service is planned to support both:

1. an official installed `yt-dlp` executable, and
2. an extracted modified source repository run directly with Python.

---

## Core User Experience

The target user experience is:

### Control panel

A protected web page where the user can:

- enter a command-like `yt-dlp` argument string
- select runtime:
  - official runtime
  - custom uploaded runtime
- submit the job
- view live logs/progress
- see final file metadata
- copy the generated expiring download URL
- upload/replace a custom source ZIP

### Download lifecycle

The expected lifecycle for a job is:

1. user submits command
2. backend parses and sanitizes arguments
3. backend resolves execution mode
4. backend runs `yt-dlp`
5. output file is written to a controlled download directory
6. backend registers an expiring download token
7. UI shows the direct link
8. user downloads file
9. backend deletes file after successful transfer completion, or later by TTL cleanup if the transfer does not complete cleanly

### Important behavior note

“Delete after successful download” means **best-effort from the server side**, not cryptographic proof that the client saved the file permanently. The realistic implementation is:

- delete after the HTTP response finishes streaming successfully from the server side, or
- delete when the link expires, whichever comes first

That is the most reliable practical interpretation for web delivery.

---

## Design Principles

### 1. Lightweight first

This service should stay small in CPU, RAM, and code complexity. The heavy work is done by `yt-dlp`, Python, and `ffmpeg`; the web service itself should remain thin.

### 2. Browser convenience without losing CLI power

The UI should be simple, but the user should still be able to express useful `yt-dlp` options rather than being limited to a few drop-downs.

### 3. No shell execution

User input must never be interpolated into a shell command string. The backend should tokenize input and pass arguments directly as an argv array to the subprocess.

### 4. Controlled filesystem behavior

All output paths must be enforced server-side. The user may influence download options, but not arbitrary file placement outside approved directories.

### 5. Temporary storage by default

Files exist only long enough to be downloaded. The server is not a permanent media archive.

### 6. Replaceable custom runtime

A modified `yt-dlp` ZIP should persist across restarts and remain active until replaced by a newer uploaded ZIP.

---

## Why Go Was Chosen

This project currently plans to use **Go** for the backend.

### Why Go fits this repository

- Single static binary deployment is simple.
- Memory usage is low and predictable.
- Standard library HTTP support is mature and sufficient.
- Subprocess management is straightforward.
- Streaming logs to the browser is simple.
- Docker packaging is easy.
- Iteration speed during development is fast.

### Why not Rust for v1

Rust is excellent, but for this specific project it adds complexity without a proportionate benefit:

- compile times are slower
- async process/log streaming is more boilerplate-heavy
- the memory delta becomes far less important once Python/`yt-dlp`/`ffmpeg` are spawned
- the project’s main risk is safe orchestration, not raw backend performance

That said, the architecture intentionally stays simple enough that a future Rust rewrite would be possible if ever desired.

---

## Planned Tech Stack

| Layer | Planned choice | Reason |
|---|---|---|
| Backend | Go | Low overhead, simple deployment, strong stdlib |
| Frontend | Server-rendered/static HTML + small vanilla JS | One-page UI does not need a heavy frontend framework |
| Reverse proxy | Traefik | Already present on target VPS |
| Runtime for official mode | installed `yt-dlp` binary or configured executable path | simplest official execution path |
| Runtime for custom mode | Python 3 + extracted custom source tree | required for repo ZIPs that do not include a binary |
| Media helpers | `ffmpeg` / `ffprobe` | important for merge/post-process workflows as upstream also recommends. |
| Containerization | Docker / Docker Compose | reproducible deployment on VPS |
| Authentication | username/password from env, verified against Argon2id hash | simple and secure enough for a personal tool |
| Session handling | signed secure cookie | avoids database dependency |
| Realtime updates | Server-Sent Events (SSE) | enough for one-way progress streaming |

---

## Authentication Plan

The web control panel will be protected by application-level authentication.

### Credential model

The repository will support environment variables such as:

- `APP_USERNAME`
- `APP_PASSWORD_HASH`

The password should **not** be stored in plaintext in normal operation. The preferred model is:

- generate an Argon2id hash offline or via a helper command
- store only the hash in `.env`
- compare submitted password against that hash at login time

### Salt, hashing, and encryption

Planned security posture:

- **Hashing:** yes, with Argon2id
- **Per-password salt:** yes, embedded as part of the Argon2id encoded hash string
- **Encryption of stored password:** no, because password verification requires comparison, not reversible recovery

Encryption is not the right primitive for password storage here. Hashing with Argon2id is.

### Session behavior

After login, the app will issue a signed cookie with:

- `HttpOnly`
- `Secure` when served behind HTTPS
- `SameSite=Lax` or stricter if practical
- short idle session lifetime with refresh on activity if needed

### Public download links vs authenticated UI

The **control panel** is authenticated. The **generated file link** is intentionally public but unguessable and time-limited. That preserves the original requirement that the final file be accessible as a direct shareable URL without forcing a login on the actual download path.

---

## Runtime Modes

The system will support two distinct download runtimes.

### 1. Official runtime mode

This mode uses a configured official `yt-dlp` executable path, such as:

- `/usr/local/bin/yt-dlp`
- or another path mounted into the container

Use this mode when the standard upstream runtime is sufficient.

### 2. Custom runtime mode

This mode uses an uploaded ZIP containing a modified `yt-dlp` source repository.

Expected flow:

1. user uploads ZIP through the web UI
2. backend validates ZIP structure
3. backend extracts ZIP into a persistent runtime directory
4. backend marks it as the active custom runtime
5. future jobs can run against that custom source tree until replaced

Planned execution style for custom mode:

- run Python against the extracted project, not through a shell
- expected target is equivalent to invoking the `yt_dlp` source tree directly from the uploaded repository

### Persistence requirement

The uploaded custom runtime must survive restarts. Therefore it will be stored in a persistent application data directory, not in `/tmp`.

### Replacement behavior

When a new ZIP is uploaded:

- validate first
- extract into a staging path
- atomically switch active custom runtime to the new version
- keep rollback logic in mind if validation or extraction fails

---

## Command Input Model

The UI should allow a command-like argument string rather than only separate form fields.

### Why command-like input was chosen

- the target user already knows `yt-dlp`
- many useful options are awkward to model in a generic form
- advanced usage should stay possible without bloating the UI

### How it will work safely

The backend will:

1. tokenize input into arguments
2. validate and sanitize tokens
3. inject or override controlled filesystem arguments
4. run subprocess with direct argv, never via `/bin/sh -c`

### Security restrictions

Some options should be blocked or overridden because they would undermine the service model. Likely examples include:

- custom output path options that escape the controlled directory
- execution hooks
- arbitrary config file loading
- dangerous plugin path overrides
- batch file input from arbitrary server paths
- other flags that break isolation or cleanup guarantees

The exact allow/deny policy will be documented in code once implemented.

---

## File and Link Lifecycle

### Controlled download directory

All completed files will live under an application-managed path such as:

- `data/downloads/<job-id>/`

This keeps cleanup easy and prevents path confusion.

### Expiring link model

After a successful job, the backend registers metadata like:

- token
- file path
- original filename
- expiry time
- claimed/downloaded state
- job ID

The user receives a direct link such as:

`https://downloads.example.com/dl/<token>/<filename>`

### Expiry behavior

A link becomes invalid when:

- TTL expires
- file has already been served and deleted
- job output has already been cleaned up
- token is unknown or revoked

### Cleanup behavior

Cleanup should happen in multiple layers:

- immediate deletion after a successful completed transfer when possible
- periodic sweep for expired files
- startup reconciliation pass for stale files/tokens
- failure cleanup for interrupted or partial jobs

This layered approach prevents disk accumulation.

---

## Realtime Job Updates

The UI should show progress and logs while a job is running.

### Planned mechanism

Use **Server-Sent Events (SSE)** for:

- job started
- current status
- stdout lines
- stderr lines
- completion
- failure
- generated link info

### Why SSE instead of WebSockets

For this use case, the browser mostly needs one-way updates from server to client. SSE is simpler than WebSockets and fits the architecture better.

---

## Security Model

This repository is a personal tool, but it still needs real safeguards.

### Security boundaries

- login protects the control panel
- public links are random, time-limited capability URLs
- downloads happen only inside approved directories
- subprocesses run without shell expansion
- uploaded ZIPs must be validated before extraction
- ZIP extraction must defend against zip-slip / path traversal
- user input must be bounded by size limits and timeouts

### Additional protections planned

- request body size limits
- one active job at a time by default
- optional queue semantics
- configurable max runtime
- configurable max output age
- audit logging of job metadata
- rejection of suspicious arguments
- optional IP logging for the authenticated panel if useful

### Threat model assumptions

This is for a trusted personal deployment, not an internet-scale hostile SaaS. Security should be responsible and practical, but the system is not being designed as a public abuse-resistant platform.

---

## Deployment Model

The service is being designed for an Ubuntu VPS that already has:

- Docker
- Traefik
- a domain/subdomain

### Planned deployment pattern

- application runs in a container
- Traefik terminates TLS
- Traefik routes a chosen host/subdomain to the container
- persistent volumes hold runtime data and custom uploaded source tree
- environment variables are loaded from `.env`

### Likely components

- app container
- mounted persistent `data/` directory
- optional bind mount or package install for official `yt-dlp`
- Python runtime available for custom source execution
- `ffmpeg` / `ffprobe` available in container or host path

---

## Planned Repository Structure

The exact structure may evolve, but the current intended layout is:

```text
yt-dlp-webgui/
├── README.md
├── .gitignore
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── cmd/
│   └── yt-dlp-webgui/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── middleware.go
│   ├── auth/
│   │   ├── password.go
│   │   ├── session.go
│   │   └── login.go
│   ├── config/
│   │   └── config.go
│   ├── jobs/
│   │   ├── manager.go
│   │   ├── queue.go
│   │   ├── runner.go
│   │   ├── parser.go
│   │   └── sanitize.go
│   ├── runtime/
│   │   ├── official.go
│   │   ├── custom.go
│   │   ├── detect.go
│   │   └── validate.go
│   ├── storage/
│   │   ├── downloads.go
│   │   ├── links.go
│   │   ├── cleanup.go
│   │   └── paths.go
│   ├── upload/
│   │   ├── zip.go
│   │   ├── extract.go
│   │   └── validate.go
│   ├── stream/
│   │   └── sse.go
│   └── web/
│       ├── handlers.go
│       ├── templates/
│       │   ├── layout.html
│       │   ├── login.html
│       │   └── app.html
│       └── static/
│           ├── app.css
│           └── app.js
├── data/
│   ├── downloads/
│   ├── runtime/
│   │   └── custom/
│   ├── uploads/
│   └── logs/
└── scripts/
    ├── dev.sh
    └── hash-password.sh
```

### Structure rationale

- `cmd/` holds the binary entrypoint
- `internal/` keeps implementation packages private to the module
- `data/` contains all mutable runtime state
- `scripts/` contains helpers for local development and ops tasks
- templates/static assets stay minimal because the UI is intentionally small

---

## Data Layout

### `data/downloads/`

Temporary finished files waiting to be downloaded.

### `data/runtime/custom/`

Persistent extracted custom `yt-dlp` source tree that stays active across restarts.

### `data/uploads/`

Transient uploaded ZIP handling area, if needed during validation/replacement.

### `data/logs/`

Operational logs, depending on how verbose logging is implemented.

---

## HTTP Surface

The route names may change, but the service will roughly need endpoints like:

| Route | Method | Purpose | Auth |
|---|---|---|---|
| `/login` | GET/POST | login form and submission | no |
| `/logout` | POST | end session | yes |
| `/` | GET | main control panel | yes |
| `/api/jobs` | POST | submit a new job | yes |
| `/api/jobs/:id` | GET | job status | yes |
| `/api/jobs/:id/events` | GET | SSE stream for logs/progress | yes |
| `/api/runtime/custom` | POST | upload/replace custom ZIP | yes |
| `/api/runtime` | GET | current runtime info | yes |
| `/api/links/:id` | GET | internal metadata lookup if needed | yes |
| `/dl/:token/:filename` | GET | direct expiring file download | no |

---

## Configuration

The exact names may change, but the repository will likely expose environment variables such as:

```env
APP_ENV=production
APP_ADDR=:8080

APP_USERNAME=admin
APP_PASSWORD_HASH=

SESSION_SECRET=
SESSION_TTL=24h

DOWNLOAD_TTL=1h
MAX_ACTIVE_JOBS=1

DATA_DIR=./data
DOWNLOADS_DIR=./data/downloads
CUSTOM_RUNTIME_DIR=./data/runtime/custom
UPLOADS_DIR=./data/uploads

OFFICIAL_YTDLP_PATH=/usr/local/bin/yt-dlp
PYTHON_BIN=python3
FFMPEG_BIN=ffmpeg
FFPROBE_BIN=ffprobe
```

### Important config note

For security, the preferred production setup is:

- `APP_PASSWORD_HASH` contains an Argon2id hash
- `SESSION_SECRET` is a long random secret
- `.env` is never committed to git

---

## Docker and Traefik Plan

### Container requirements

The image will likely include:

- compiled Go binary
- Python 3
- `ffmpeg`
- `ffprobe`
- optionally official `yt-dlp`, or a mounted executable path

### Traefik expectations

The app itself will stay simple and assume Traefik handles:

- HTTPS termination
- host routing
- proxy forwarding
- optional additional hardening headers if desired

### Example deployment shape

- `ytdlp.example.com` -> authenticated control panel
- generated public links may be served under the same host or a separate subdomain, depending on routing preference

A separate subdomain is not required, but it can make policy separation cleaner.

---

## Planned UI

The UI should stay intentionally minimal.

### Main screen

- runtime selector
- ZIP upload area
- command input field/textarea
- submit button
- live log panel
- final output card with direct link and expiry info

### UI philosophy

- utility first, not flashy
- works well on desktop and mobile
- minimal JS
- no unnecessary frontend build chain
- no framework unless complexity later justifies one

---

## Error Handling Philosophy

The app should fail in a way that is understandable.

Examples:

- invalid login -> clear message
- missing custom runtime -> clear message
- malformed ZIP -> clear message
- blocked flag -> explicit explanation
- no output file produced -> explicit job failure state
- expired link -> proper 404/410-style response
- partial cleanup failure -> logged for operator attention

---

## Operational Philosophy

This project is meant to be maintainable by one technically comfortable self-hoster.

That means:

- readable code over clever code
- minimal moving parts
- explicit configuration
- predictable disk layout
- restart-safe custom runtime persistence
- easy local dev and easy Docker deployment

---

## Development Plan

### Phase 1: skeleton

- Go module setup
- config loading
- HTTP server
- login/logout flow
- basic HTML templates

### Phase 2: job execution

- command parser/tokenizer
- sanitize/blocklist/allowlist rules
- subprocess runner
- log capture
- one-job queue

### Phase 3: file serving

- detect output file
- token generation
- expiring direct link
- cleanup sweeper

### Phase 4: custom runtime

- ZIP upload
- ZIP validation
- secure extraction
- persistent active runtime replacement

### Phase 5: deployment

- Dockerfile
- compose file
- `.env.example`
- Traefik labels/docs
- production hardening pass

### Phase 6: polish

- better log UI
- clearer error states
- runtime metadata display
- maybe command history in memory per session if useful

---

## Design Trade-Offs

### Command textbox vs form-only UI

Chosen: command textbox.

Reason: flexibility matters more than rigid beginner UX for this repository’s target user.

### Go vs Rust

Chosen: Go.

Reason: simpler implementation and faster iteration for a subprocess-driven web tool.

### SSE vs WebSockets

Chosen: SSE.

Reason: simpler one-way progress streaming is enough.

### Public download link vs authenticated file route

Chosen: public tokenized link.

Reason: direct link sharing was a primary requirement.

### Persist custom ZIP across restarts

Chosen: yes.

Reason: explicit user requirement and central project differentiator.

### Database vs no database

Chosen: no database initially.

Reason: single-user personal tool does not need one for v1.

---

## Risks and Edge Cases

A few implementation details need careful handling:

- `yt-dlp` may produce multiple output files for some command combinations
- a command may complete successfully without a final media file in the expected place
- post-processing may rename the final output after download
- interrupted HTTP transfers may require delayed cleanup instead of immediate deletion
- custom ZIPs may contain unexpected structure and must be validated strictly
- some flags may conflict with the service’s enforced output-path model
- future custom forks may depend on additional Python packages not present in the base container

These are not reasons to avoid the design; they are just areas that need explicit handling in code.

---

## Future Possibilities

Not required for v1, but possible later:

- recent jobs page
- per-job downloadable logs
- command presets
- reusable output templates
- optional webhook callback on completion
- optional Telegram notification
- optional archive retention window
- optional support for multiple output files per job
- optional rate limiting and IP allowlisting
- optional separate download host/subdomain

---

## Status

This repository is currently in the planning/bootstrap phase. The architecture is being defined before implementation so the codebase stays small, intentional, and easy to operate.

The project direction is clear:

- browser-driven
- self-hosted
- lightweight
- secure enough for personal internet exposure
- compatible with both official and modified `yt-dlp` runtimes

---

## Initial Repository Notes

Recommended initial files:

- `.gitignore`
- `README.md`
- `.env.example`
- `Dockerfile`
- `docker-compose.yml`
- minimal Go app entrypoint
- placeholder template/static files
- password hash helper script

The `.gitignore` should exclude:

- `.env`
- `data/downloads/`
- `data/runtime/`
- `data/uploads/`
- generated binaries
- editor junk
- OS junk

---

## Summary of the Intended End State

The end result should be a small self-hosted web service that:

- runs behind Traefik on an Ubuntu VPS
- authenticates the control panel with env-configured credentials
- accepts `yt-dlp`-style commands from a single-page browser UI
- runs jobs through either an official runtime or a persistent custom uploaded source ZIP
- stores output only temporarily
- produces expiring public direct download links
- cleans up files automatically
- remains easy to audit, deploy, and maintain
