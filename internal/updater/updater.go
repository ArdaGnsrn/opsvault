package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiURL        = "https://api.github.com/repos/ArdaGnsrn/opsvault/releases/latest"
	cacheTTL      = 24 * time.Hour
	httpTimeout   = 5 * time.Second
)

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

// LatestVersion returns the latest GitHub release tag, using a 24h cache.
// Returns empty string on any error so callers can safely ignore it.
func LatestVersion() string {
	cacheFile := cacheFilePath()

	if cached := readCache(cacheFile); cached != "" {
		return cached
	}

	latest := fetchLatest()
	if latest != "" {
		writeCache(cacheFile, latest)
	}
	return latest
}

// IsNewer reports whether latestVersion is strictly newer than currentVersion.
// Compares semver strings of the form "v1.2.3".
func IsNewer(current, latest string) bool {
	if latest == "" || current == "" || current == "dev" {
		return false
	}
	return normalize(latest) != normalize(current) && latest > current
}

func fetchLatest() string {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var rel releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ""
	}
	return rel.TagName
}

type cacheEntry struct {
	Version   string    `json:"version"`
	FetchedAt time.Time `json:"fetched_at"`
}

func cacheFilePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "opsvault", "version-check.json")
}

func readCache(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ""
	}
	if time.Since(entry.FetchedAt) > cacheTTL {
		return ""
	}
	return entry.Version
}

func writeCache(path, version string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	data, err := json.Marshal(cacheEntry{Version: version, FetchedAt: time.Now()})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func normalize(v string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s", v), "v")
}
