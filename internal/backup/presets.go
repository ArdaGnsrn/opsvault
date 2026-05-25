package backup

import "github.com/ArdaGnsrn/opsvault/internal/config"

// PresetExcludes maps preset keys to the glob patterns they expand to.
var PresetExcludes = map[string][]string{
	"node_modules": {"node_modules"},
	"vendor":       {"vendor"},
	"git":          {".git"},
	"python":       {"__pycache__", "*.pyc", "*.pyo", ".venv", "venv"},
	"build":        {"dist", "build", "target", "out", ".next", ".nuxt"},
	"logs":         {"*.log", "logs"},
	"temp":         {"*.tmp", "*.temp", "tmp", "temp"},
	"ide":          {".idea", ".vscode", "*.swp", "*.swo"},
	"cache":        {"cache", ".cache", "*.cache"},
	"docker":       {".docker"},
}

// PresetLabels defines the display order and labels for the wizard.
var PresetLabels = []struct {
	Key   string
	Label string
}{
	{"node_modules", "node_modules"},
	{"vendor", "vendor  (PHP / Go)"},
	{"git", ".git"},
	{"python", "Python  (__pycache__, .venv, *.pyc)"},
	{"build", "Build   (dist, build, target, .next)"},
	{"logs", "Logs    (*.log)"},
	{"temp", "Temp    (*.tmp, tmp/)"},
	{"ide", "IDE     (.idea, .vscode)"},
	{"cache", "Cache   (cache/, .cache/)"},
	{"docker", "Docker  (.docker/)"},
}

// ResolveExcludes merges preset patterns with custom excluded_paths into a
// single deduplicated slice ready for the archiver.
func ResolveExcludes(cfg config.PathConfig) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	for _, key := range cfg.PresetExcludes {
		for _, pat := range PresetExcludes[key] {
			add(pat)
		}
	}
	for _, pat := range cfg.ExcludedPaths {
		add(pat)
	}
	return out
}
