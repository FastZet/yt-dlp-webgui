package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Server
	Env  string
	Addr string

	// Auth
	Username     string
	PasswordHash string

	// Session
	SessionSecret string
	SessionTTL    time.Duration

	// Downloads
	DownloadTTL    time.Duration
	MaxActiveJobs  int

	// Paths
	DataDir          string
	DownloadsDir     string
	CustomRuntimeDir string
	UploadsDir       string
	LogsDir          string

	// Binaries
	OfficialYtdlpPath string
	PythonBin         string
	FfmpegBin         string
	FfprobeBin        string
}

// Load reads configuration from environment variables and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{
		Env:  envOr("APP_ENV", "production"),
		Addr: envOr("APP_ADDR", ":8080"),

		Username:     os.Getenv("APP_USERNAME"),
		PasswordHash: os.Getenv("APP_PASSWORD_HASH"),

		SessionSecret: os.Getenv("SESSION_SECRET"),

		DataDir:          envOr("DATA_DIR", "./data"),
		DownloadsDir:     envOr("DOWNLOADS_DIR", "./data/downloads"),
		CustomRuntimeDir: envOr("CUSTOM_RUNTIME_DIR", "./data/runtime/custom"),
		UploadsDir:       envOr("UPLOADS_DIR", "./data/uploads"),
		LogsDir:          envOr("LOGS_DIR", "./data/logs"),

		OfficialYtdlpPath: envOr("OFFICIAL_YTDLP_PATH", "/usr/local/bin/yt-dlp"),
		PythonBin:         envOr("PYTHON_BIN", "python3"),
		FfmpegBin:         envOr("FFMPEG_BIN", "ffmpeg"),
		FfprobeBin:        envOr("FFPROBE_BIN", "ffprobe"),
	}

	// Parse durations
	var err error
	cfg.SessionTTL, err = parseDuration("SESSION_TTL", "24h")
	if err != nil {
		return nil, err
	}
	cfg.DownloadTTL, err = parseDuration("DOWNLOAD_TTL", "1h")
	if err != nil {
		return nil, err
	}

	// Parse int
	cfg.MaxActiveJobs, err = parseInt("MAX_ACTIVE_JOBS", 1)
	if err != nil {
		return nil, err
	}
	if cfg.MaxActiveJobs < 1 {
		cfg.MaxActiveJobs = 1
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Resolve all paths to absolute
	cfg.resolveAbsPaths()

	return cfg, nil
}

func (c *Config) validate() error {
	var errs []string

	if strings.TrimSpace(c.Username) == "" {
		errs = append(errs, "APP_USERNAME must not be empty")
	}
	if strings.TrimSpace(c.PasswordHash) == "" {
		errs = append(errs, "APP_PASSWORD_HASH must not be empty — generate one with scripts/hash-password.sh")
	}
	if strings.TrimSpace(c.SessionSecret) == "" {
		errs = append(errs, "SESSION_SECRET must not be empty — generate with: openssl rand -hex 32")
	}
	if len(c.SessionSecret) < 32 {
		errs = append(errs, "SESSION_SECRET must be at least 32 characters")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed:\n  - " + strings.Join(errs, "\n  - "))
	}
	return nil
}

func (c *Config) resolveAbsPaths() {
	c.DataDir = absPath(c.DataDir)
	c.DownloadsDir = absPath(c.DownloadsDir)
	c.CustomRuntimeDir = absPath(c.CustomRuntimeDir)
	c.UploadsDir = absPath(c.UploadsDir)
	c.LogsDir = absPath(c.LogsDir)
}

// IsDev returns true when running in development mode.
func (c *Config) IsDev() bool {
	return strings.ToLower(c.Env) == "development"
}

// --- helpers ---

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := envOr(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %q — expected a Go duration like 1h, 30m, 24h", key, raw)
	}
	return d, nil
}

func parseInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %q — expected an integer", key, raw)
	}
	return v, nil
}

func absPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
