package migrategen

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	atlassqlite "ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	_ "ariga.io/atlas/sql/postgres" // register the postgres Atlas client
	_ "modernc.org/sqlite"          // pure-Go sqlite driver for the in-memory dev DB
)

// loadedSchema is an inspected schema plus the resources that must be released
// once diffing is done.
type loadedSchema struct {
	schema *schema.Schema
	driver migrate.Driver
	closer func() error
}

// loadSchema spins up a throwaway "dev database" for the configured engine,
// executes ddl into it, and inspects the result into an Atlas schema model.
// The returned loadedSchema must be closed by the caller.
//
// ddl may be empty, which yields an empty schema — this is how the very first
// run produces an "init" migration that creates everything.
func loadSchema(ctx context.Context, cfg *Config, ddl string) (*loadedSchema, error) {
	switch cfg.Engine {
	case EngineSQLite:
		return loadSQLite(ctx, ddl)
	case EnginePostgres:
		return loadPostgres(ctx, cfg, ddl)
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

// loadSQLite uses an in-memory SQLite database. MaxOpenConns is pinned to 1 so
// every query hits the same in-memory database rather than a fresh one.
func loadSQLite(ctx context.Context, ddl string) (*loadedSchema, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open in-memory sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if ddl != "" {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply schema to sqlite dev db: %w", err)
		}
	}

	drv, err := atlassqlite.Open(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open atlas sqlite driver: %w", err)
	}

	s, err := drv.InspectSchema(ctx, "main", nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("inspect sqlite schema: %w", err)
	}

	return &loadedSchema{schema: s, driver: drv, closer: db.Close}, nil
}

// loadPostgres connects to a user-provided dev database and isolates the work
// in a uniquely named temporary schema, so the old and new loads never collide
// and the dev database is left untouched afterwards.
func loadPostgres(ctx context.Context, cfg *Config, ddl string) (*loadedSchema, error) {
	if cfg.DevURL == "" {
		return nil, fmt.Errorf("postgresql requires a dev database; pass --dev-url or set MIGRATEGEN_DEV_URL (e.g. postgres://localhost:5432/dev?sslmode=disable)")
	}

	client, err := sqlclient.Open(ctx, cfg.DevURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres dev db: %w", err)
	}

	name := "migrategen_dev_" + randHex()
	if _, err := client.DB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %q", name)); err != nil {
		client.Close()
		return nil, fmt.Errorf("create temp schema: %w", err)
	}
	dropSchema := func() error {
		_, err := client.DB.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA %q CASCADE", name))
		return err
	}

	if _, err := client.DB.ExecContext(ctx, fmt.Sprintf("SET search_path TO %q", name)); err != nil {
		dropSchema()
		client.Close()
		return nil, fmt.Errorf("set search_path: %w", err)
	}
	if ddl != "" {
		if _, err := client.DB.ExecContext(ctx, ddl); err != nil {
			dropSchema()
			client.Close()
			return nil, fmt.Errorf("apply schema to postgres dev db: %w", err)
		}
	}

	s, err := client.InspectSchema(ctx, name, nil)
	if err != nil {
		dropSchema()
		client.Close()
		return nil, fmt.Errorf("inspect postgres schema: %w", err)
	}

	closer := func() error {
		dropErr := dropSchema()
		closeErr := client.Close()
		if dropErr != nil {
			return dropErr
		}
		return closeErr
	}
	return &loadedSchema{schema: s, driver: client.Driver, closer: closer}, nil
}

func randHex() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
