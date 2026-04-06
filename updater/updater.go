package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// LatestRelease describes the latest available version.
type LatestRelease struct {
	Version  string            `json:"version"`
	Binaries map[string]Binary `json:"binaries"` // key: "linux/amd64", "linux/arm64", etc.
}

// Binary describes a downloadable binary.
type Binary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

const checkURL = "https://r2.conch-talk.com/dlc/latest.json"

// StartSchedule runs a background goroutine that checks for updates daily at UTC midnight.
func StartSchedule(currentVersion string, done <-chan struct{}) {
	go func() {
		// Wait until next UTC midnight
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		delay := next.Sub(now)
		log.Printf("[updater] next check in %v (UTC midnight)", delay.Round(time.Second))

		select {
		case <-time.After(delay):
		case <-done:
			return
		}

		// Check immediately, then every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		check(currentVersion)

		for {
			select {
			case <-ticker.C:
				check(currentVersion)
			case <-done:
				return
			}
		}
	}()
}

func check(currentVersion string) {
	log.Printf("[updater] checking for updates (current: %s)", currentVersion)

	release, err := fetchLatest()
	if err != nil {
		log.Printf("[updater] check failed: %v", err)
		return
	}

	if release.Version == currentVersion || release.Version == "" {
		log.Printf("[updater] up to date")
		return
	}

	key := runtime.GOOS + "/" + runtime.GOARCH
	bin, ok := release.Binaries[key]
	if !ok {
		log.Printf("[updater] no binary for %s", key)
		return
	}

	log.Printf("[updater] new version available: %s -> %s", currentVersion, release.Version)

	if err := downloadAndReplace(bin); err != nil {
		log.Printf("[updater] update failed: %v", err)
		return
	}

	log.Printf("[updater] updated to %s, restarting...", release.Version)
	restart()
}

func fetchLatest() (*LatestRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(checkURL)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release LatestRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &release, nil
}

func downloadAndReplace(bin Binary) error {
	// Download to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(bin.URL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status: %d", resp.StatusCode)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	tmp, err := os.CreateTemp("", "conchtalk-dlc-update-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // clean up on failure
	}()

	// Download and compute hash
	hasher := sha256.New()
	writer := io.MultiWriter(tmp, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	tmp.Close()

	// Verify checksum
	got := hex.EncodeToString(hasher.Sum(nil))
	if bin.SHA256 != "" && got != bin.SHA256 {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, bin.SHA256)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Atomic replace: rename over the existing binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Cross-device rename fallback: copy then remove
		if err := copyFile(tmpPath, execPath); err != nil {
			return fmt.Errorf("replace: %w", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func restart() {
	// Try systemctl restart first (most common deployment)
	if err := exec.Command("systemctl", "restart", "conchtalk-dlc").Run(); err != nil {
		log.Printf("[updater] systemctl restart failed: %v, exiting for manual restart", err)
		os.Exit(0)
	}
}
