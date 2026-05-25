package envfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathFor returns the env file path alongside the given config file.
func PathFor(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), "env")
}

// Load reads key=value pairs from the env file and sets them in the process
// environment (only if the var is not already set).
func Load(envPath string) error {
	f, err := os.Open(envPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
	return scanner.Err()
}

// Write writes the given key=value map to the env file (mode 0600).
// Existing keys not present in vars are preserved.
func Write(envPath string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	existing := map[string]string{}
	if f, err := os.Open(envPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				existing[strings.TrimSpace(k)] = v
			}
		}
		f.Close()
	}

	for k, v := range vars {
		if v != "" {
			existing[k] = v
		}
	}

	if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
		return fmt.Errorf("creating env dir: %w", err)
	}

	f, err := os.OpenFile(envPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("writing env file: %w", err)
	}
	defer f.Close()

	for k, v := range existing {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	return nil
}
