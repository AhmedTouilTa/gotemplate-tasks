package migrategen

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/migrate"
)

// Options configures a single Generate run.
type Options struct {
	Config *Config
	// Name is the human-readable migration name (the {name} in the filename).
	Name string
}

// Result describes the outcome of a Generate run.
type Result struct {
	// NoChanges is true when the schema is identical to the snapshot and no
	// files were written.
	NoChanges bool
	// Version is the numeric version assigned to the migration (0 if none).
	Version uint
	// UpPath / DownPath are the written files (empty if NoChanges).
	UpPath, DownPath string
}

// Generate diffs the current sqlc schema against the stored snapshot and writes
// a golang-migrate up/down pair plus an updated snapshot. When the schema is
// unchanged it writes nothing and reports NoChanges.
func Generate(ctx context.Context, opts Options) (*Result, error) {
	cfg := opts.Config
	if opts.Name == "" {
		return nil, fmt.Errorf("migration name is required")
	}

	newDDL, err := readSchemaFile(cfg.SchemaPath)
	if err != nil {
		return nil, err
	}
	oldDDL, err := readSnapshot(cfg.SnapshotPath)
	if err != nil {
		return nil, err
	}

	up, err := planSQL(ctx, cfg, oldDDL, newDDL)
	if err != nil {
		return nil, fmt.Errorf("plan up migration: %w", err)
	}
	if len(up) == 0 {
		return &Result{NoChanges: true}, nil
	}

	down, err := planSQL(ctx, cfg, newDDL, oldDDL)
	if err != nil {
		return nil, fmt.Errorf("plan down migration: %w", err)
	}

	res, err := writeMigration(cfg, opts.Name, up, down)
	if err != nil {
		return nil, err
	}

	if err := writeSnapshot(cfg.SnapshotPath, newDDL); err != nil {
		return nil, err
	}
	return res, nil
}

// planSQL loads the "from" and "to" DDL into dev databases, diffs them, and
// returns the SQL statements that migrate from -> to.
func planSQL(ctx context.Context, cfg *Config, fromDDL, toDDL string) ([]string, error) {
	from, err := loadSchema(ctx, cfg, fromDDL)
	if err != nil {
		return nil, err
	}
	defer from.closer()

	to, err := loadSchema(ctx, cfg, toDDL)
	if err != nil {
		return nil, err
	}
	defer to.closer()

	// Schemas may be inspected under different names (e.g. a temp postgres
	// schema). Normalise so the diff doesn't see a spurious schema rename and
	// statements aren't schema-qualified.
	from.schema.Name = ""
	to.schema.Name = ""

	changes, err := to.driver.SchemaDiff(from.schema, to.schema)
	if err != nil {
		return nil, fmt.Errorf("compute schema diff: %w", err)
	}
	if len(changes) == 0 {
		return nil, nil
	}

	plan, err := to.driver.PlanChanges(ctx, "diff", changes, func(o *migrate.PlanOptions) {
		noQualifier := ""
		o.SchemaQualifier = &noQualifier
	})
	if err != nil {
		return nil, fmt.Errorf("plan changes: %w", err)
	}

	stmts := make([]string, 0, len(plan.Changes))
	for _, c := range plan.Changes {
		stmts = append(stmts, strings.TrimSpace(c.Cmd))
	}
	return stmts, nil
}
