package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
)

type Entry struct {
	Database  string    `json:"database"`
	Status    Status    `json:"status"`
	FilePath  string    `json:"file_path,omitempty"`
	FileSize  int64     `json:"file_size,omitempty"`
	Duration  float64   `json:"duration_seconds"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

func filePath(backupDir string) string {
	return filepath.Join(backupDir, ".history.jsonl")
}

func Append(backupDir string, e Entry) error {
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return err
	}
	f, err := os.OpenFile(filePath(backupDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(e)
}

func Load(backupDir string) ([]Entry, error) {
	f, err := os.Open(filePath(backupDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StartedAt.After(entries[j].StartedAt)
	})
	return entries, nil
}
