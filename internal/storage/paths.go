package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fastzet/yt-dlp-webgui/internal/config"
)

// Paths groups the main application storage directories.
type Paths struct {
	DataDir          string
	DownloadsDir     string
	CustomRuntimeDir string
	UploadsDir       string
	LogsDir          string
}

// NewPaths derives storage paths from config.
func NewPaths(cfg *config.Config) *Paths {
	return &Paths{
		DataDir:          cfg.DataDir,
		DownloadsDir:     cfg.DownloadsDir,
		CustomRuntimeDir: cfg.CustomRuntimeDir,
		UploadsDir:       cfg.UploadsDir,
		LogsDir:          cfg.LogsDir,
	}
}

// EnsureDirs creates all required application directories if they do not exist.
func (p *Paths) EnsureDirs() error {
	dirs := []string{
		p.DataDir,
		p.DownloadsDir,
		p.CustomRuntimeDir,
		p.UploadsDir,
		p.LogsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return nil
}

// JobDownloadDir returns the directory used for a single download job.
func (p *Paths) JobDownloadDir(jobID string) string {
	return filepath.Join(p.DownloadsDir, jobID)
}

// JobOutputTemplate returns the yt-dlp output template for a given job.
// Example: /abs/path/data/downloads/<jobID>/%(title)s.%(ext)s
func (p *Paths) JobOutputTemplate(jobID string) string {
	return filepath.Join(p.JobDownloadDir(jobID), "%(title)s.%(ext)s")
}

// StagedUploadZipPath returns the temporary uploaded ZIP path for a given upload ID.
func (p *Paths) StagedUploadZipPath(uploadID string) string {
	return filepath.Join(p.UploadsDir, uploadID+".zip")
}

// StagedRuntimeExtractDir returns the staging extraction directory for a new custom runtime.
func (p *Paths) StagedRuntimeExtractDir(uploadID string) string {
	return filepath.Join(p.UploadsDir, uploadID+"_extracted")
}

// ActiveCustomRuntimeDir returns the currently active persistent custom runtime directory.
func (p *Paths) ActiveCustomRuntimeDir() string {
	return p.CustomRuntimeDir
}

// ActiveCustomYTDLPEntry returns the expected filesystem entry point for the custom runtime.
// The uploaded source tree is expected to contain a yt_dlp package.
func (p *Paths) ActiveCustomYTDLPEntry() string {
	return filepath.Join(p.CustomRuntimeDir, "yt_dlp")
}

// CleanupUploadArtifacts removes temporary upload ZIP and extracted staging directory.
func (p *Paths) CleanupUploadArtifacts(uploadID string) error {
	zipPath := p.StagedUploadZipPath(uploadID)
	extractDir := p.StagedRuntimeExtractDir(uploadID)

	var firstErr error

	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		firstErr = fmt.Errorf("removing staged zip %s: %w", zipPath, err)
	}
	if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) && firstErr == nil {
		firstErr = fmt.Errorf("removing staged extract dir %s: %w", extractDir, err)
	}

	return firstErr
}
