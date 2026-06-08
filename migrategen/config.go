// Package migrategen generates golang-migrate migration files by diffing a
// project's sqlc schema against the last schema it generated migrations for.
//
// It uses the Atlas SDK (ariga.io/atlas) to parse DDL and compute the
// structural diff, then renders the result as paired
// {version}_{name}.up.sql / {version}_{name}.down.sql files.
package migrategen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// Engine identifies the SQL dialect a schema targets.
type Engine string

const (
	EngineSQLite   Engine = "sqlite"
	EnginePostgres Engine = "postgresql"
)

// Config describes where a project's schema lives and where migrations and the
// snapshot should be written. It is normally derived from a sqlc.yaml via
// LoadConfig, but can be built by hand for the library API.
type Config struct {
	// Engine is the SQL dialect ("sqlite" or "postgresql").
	Engine Engine
	// SchemaPath is the absolute path to the current sqlc schema file.
	SchemaPath string
	// OutDir is the directory migration files are written to.
	OutDir string
	// SnapshotPath is the file holding the last schema we generated for.
	SnapshotPath string
	// DevURL is the Atlas dev-database URL. Required for postgresql; ignored
	// for sqlite (an in-memory database is used instead).
	DevURL string
}

// sqlcConfig mirrors the slice of the sqlc.yaml (v2) we care about.
type sqlcConfig struct {
	SQL []struct {
		Engine string    `yaml:"engine"`
		Schema stringList `yaml:"schema"`
	} `yaml:"sql"`
}

// stringList accepts either a single scalar or a sequence in YAML, matching
// sqlc's "schema" field which may be a path or a list of paths.
type stringList []string

func (s *stringList) UnmarshalYAML(unmarshal func(any) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = []string{single}
		return nil
	}
	var many []string
	if err := unmarshal(&many); err != nil {
		return err
	}
	*s = many
	return nil
}

// LoadConfig reads a sqlc.yaml and derives a Config. Paths in the sqlc.yaml are
// resolved relative to the config file's directory, matching sqlc's behaviour.
// The first entry of the sql: block is used.
func LoadConfig(configPath, outDir, devURL string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read sqlc config %s: %w", configPath, err)
	}

	var sc sqlcConfig
	if err := yaml.Unmarshal(raw, &sc); err != nil {
		return nil, fmt.Errorf("parse sqlc config %s: %w", configPath, err)
	}
	if len(sc.SQL) == 0 {
		return nil, fmt.Errorf("sqlc config %s has no sql: entries", configPath)
	}

	entry := sc.SQL[0]
	if len(entry.Schema) == 0 {
		return nil, fmt.Errorf("sqlc config %s: first sql entry has no schema", configPath)
	}
	if len(entry.Schema) > 1 {
		return nil, fmt.Errorf("sqlc config %s: multiple schema files are not supported yet", configPath)
	}

	engine, err := normalizeEngine(entry.Engine)
	if err != nil {
		return nil, err
	}

	configDir := filepath.Dir(configPath)
	schemaPath := entry.Schema[0]
	if !filepath.IsAbs(schemaPath) {
		schemaPath = filepath.Join(configDir, schemaPath)
	}

	if outDir == "" {
		outDir = filepath.Join(configDir, "migrations")
	}

	return &Config{
		Engine:       engine,
		SchemaPath:   schemaPath,
		OutDir:       outDir,
		SnapshotPath: filepath.Join(outDir, ".snapshot.sql"),
		DevURL:       devURL,
	}, nil
}

// normalizeEngine maps the engine strings sqlc accepts onto our Engine values.
func normalizeEngine(s string) (Engine, error) {
	switch s {
	case "sqlite":
		return EngineSQLite, nil
	case "postgresql", "postgres":
		return EnginePostgres, nil
	default:
		return "", fmt.Errorf("unsupported sqlc engine %q (want sqlite or postgresql)", s)
	}
}
