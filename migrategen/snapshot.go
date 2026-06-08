package migrategen

import (
	"fmt"
	"os"
	"path/filepath"
)

// readSnapshot returns the DDL of the last schema we generated migrations for.
// A missing snapshot yields an empty string (not an error): the first run then
// diffs the current schema against an empty database.
func readSnapshot(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read snapshot %s: %w", path, err)
	}
	return string(b), nil
}

// writeSnapshot records the current schema DDL so the next run diffs against it.
func writeSnapshot(path, ddl string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(ddl), 0o644); err != nil {
		return fmt.Errorf("write snapshot %s: %w", path, err)
	}
	return nil
}
