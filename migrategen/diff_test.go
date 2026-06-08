package migrategen

import (
	"context"
	"strings"
	"testing"
)

// sqliteCfg is a Config that diffs SQLite schemas using an in-memory dev DB, so
// these tests need no external services.
func sqliteCfg() *Config {
	return &Config{Engine: EngineSQLite}
}

func TestPlanSQL_CreateTable(t *testing.T) {
	ctx := context.Background()
	stmts, err := planSQL(ctx, sqliteCfg(), "", `CREATE TABLE tasks (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`)
	if err != nil {
		t.Fatalf("planSQL: %v", err)
	}
	if len(stmts) != 1 || !strings.Contains(stmts[0], "CREATE TABLE") || !strings.Contains(stmts[0], "tasks") {
		t.Fatalf("expected a CREATE TABLE tasks statement, got %v", stmts)
	}
}

func TestPlanSQL_AddColumn(t *testing.T) {
	ctx := context.Background()
	old := `CREATE TABLE tasks (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`
	new := `CREATE TABLE tasks (id INTEGER PRIMARY KEY, name TEXT NOT NULL, priority INTEGER);`

	up, err := planSQL(ctx, sqliteCfg(), old, new)
	if err != nil {
		t.Fatalf("planSQL up: %v", err)
	}
	if len(up) != 1 || !strings.Contains(up[0], "ADD COLUMN") || !strings.Contains(up[0], "priority") {
		t.Fatalf("expected ADD COLUMN priority, got %v", up)
	}

	down, err := planSQL(ctx, sqliteCfg(), new, old)
	if err != nil {
		t.Fatalf("planSQL down: %v", err)
	}
	if len(down) == 0 {
		t.Fatalf("expected a non-empty down plan")
	}
}

func TestPlanSQL_NoChange(t *testing.T) {
	ctx := context.Background()
	ddl := `CREATE TABLE tasks (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`
	stmts, err := planSQL(ctx, sqliteCfg(), ddl, ddl)
	if err != nil {
		t.Fatalf("planSQL: %v", err)
	}
	if len(stmts) != 0 {
		t.Fatalf("expected no statements for identical schemas, got %v", stmts)
	}
}
