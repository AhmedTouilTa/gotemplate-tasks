package migrategen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// versionWidth is golang-migrate's default zero-padding for version numbers.
const versionWidth = 6

// migrationFilePattern matches a golang-migrate file's leading version number,
// e.g. "000002_add_priority.up.sql".
var migrationFilePattern = regexp.MustCompile(`^(\d+)_.*\.(up|down)\.sql$`)

// readSchemaFile reads the current sqlc schema DDL.
func readSchemaFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read schema %s: %w", path, err)
	}
	return string(b), nil
}

// writeMigration writes the up/down statement lists as a golang-migrate file
// pair, assigning the next sequential version number found in the output dir.
func writeMigration(cfg *Config, name string, up, down []string) (*Result, error) {
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("create migrations dir: %w", err)
	}

	version, err := nextVersion(cfg.OutDir)
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("%0*d_%s", versionWidth, version, sanitizeName(name))
	upPath := filepath.Join(cfg.OutDir, prefix+".up.sql")
	downPath := filepath.Join(cfg.OutDir, prefix+".down.sql")

	if err := os.WriteFile(upPath, []byte(formatStmts(up)), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", upPath, err)
	}
	if err := os.WriteFile(downPath, []byte(formatStmts(down)), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", downPath, err)
	}

	return &Result{Version: version, UpPath: upPath, DownPath: downPath}, nil
}

// nextVersion returns max(existing version)+1, or 1 if the dir has no
// migrations yet.
func nextVersion(dir string) (uint, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var max uint
	for _, e := range entries {
		m := migrationFilePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			continue
		}
		if uint(n) > max {
			max = uint(n)
		}
	}
	return max + 1, nil
}

// formatStmts joins SQL statements into a migration file body. Each statement
// is terminated with a semicolon and separated by a blank line.
func formatStmts(stmts []string) string {
	var b strings.Builder
	for _, s := range stmts {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ";"))
		if s == "" {
			continue
		}
		b.WriteString(s)
		b.WriteString(";\n\n")
	}
	return b.String()
}

// sanitizeName turns a free-form migration name into a filename-safe slug.
func sanitizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "migration"
	}
	return out
}
