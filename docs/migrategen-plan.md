# Plan: `migrategen` — generate golang-migrate migrations from sqlc schemas

> Status: **implemented**. See `migrategen/`, `cmd/migrategen/`, and the
> generated baseline in `db/migrations/`.

## Context

The project keeps its database schema in a single sqlc `schema.sql`
(`db/schema.sql`, referenced by `db/sqlc.yaml`). When that schema changes there
was no mechanism to evolve a real database — `main.go` just re-ran the full DDL
with `CREATE TABLE IF NOT EXISTS`. `migrategen` is a reusable library + CLI that,
on each schema change, **diffs the previous schema against the current sqlc
`schema.sql` and emits versioned
[golang-migrate](https://github.com/golang-migrate/migrate) `*.up.sql` /
`*.down.sql` files**, so schema changes become reviewable, ordered, reversible
migrations.

Decisions locked in with the user:
- **Diff-based** (not snapshot) migrations.
- Output format: **golang-migrate** (`{version}_{name}.up.sql` / `.down.sql`).
- Engines: **SQLite + PostgreSQL**.
- Diff engine: **wrap the Atlas Go SDK** (`ariga.io/atlas`) rather than hand-roll a DDL parser/differ.
- Interface: **importable Go package + thin CLI** (`migrategen generate`).

## How it works (data flow)

```
sqlc.yaml ──► discover engine + schema.sql path
                     │
   stored snapshot (OLD schema)        current schema.sql (NEW schema)
                     │                          │
        load into Atlas dev DB ──► inspect ──► oldSchema   newSchema
                     │                          │
        Atlas SchemaDiff(old → new) ──► PlanChanges ──► UP statements
        Atlas SchemaDiff(new → old) ──► PlanChanges ──► DOWN statements
                     │
        render golang-migrate files: NNNNNN_<name>.up.sql / .down.sql
                     │
        overwrite stored snapshot with NEW schema (for next run)
```

Why a "dev database": Atlas parses/normalizes DDL by executing it against a
throwaway database, then introspecting it.
- **SQLite** → in-memory dev DB (`:memory:`), pure-Go via the existing
  `modernc.org/sqlite` dependency. **No external dependency.** `MaxOpenConns` is
  pinned to 1 so every query hits the same in-memory database.
- **PostgreSQL** → needs a real throwaway Postgres, supplied via `--dev-url` /
  `MIGRATEGEN_DEV_URL`. Each load runs inside a uniquely named temporary schema
  that is dropped afterwards, so old/new loads never collide and the dev DB is
  left untouched.

Atlas `SchemaDiff` is one-directional, so we call it twice (old→new for up,
new→old for down) to get reversible migrations. Inspected schema names are
normalized to `""` before diffing so a temp-schema name never shows up as a
spurious rename or qualifier.

## Package / file layout

Code lives under the existing module `gotemplates/todo`:

- `migrategen/config.go` — locate & parse `sqlc.yaml` (v2: `sql[].engine`,
  `sql[].schema`); resolve schema path, output dir, snapshot path. Uses
  `github.com/goccy/go-yaml`. `schema` may be a scalar or a list.
- `migrategen/devdb.go` — open an Atlas driver against a dev DB per engine and
  load DDL → `InspectSchema` into a `*schema.Schema`.
  - sqlite: `sql.Open("sqlite", ":memory:")` → `sqlite.Open(db)` → inspect `main`.
  - postgres: `sqlclient.Open(devURL)`, create temp schema, `SET search_path`,
    exec DDL, inspect, drop schema on close.
- `migrategen/snapshot.go` — read/write `db/migrations/.snapshot.sql`; a missing
  snapshot yields an empty string (first run diffs against an empty DB).
- `migrategen/diff.go` — core engine. `Generate(ctx, Options) (*Result, error)`:
  reads schema + snapshot, `planSQL` both directions, writes the file pair and
  updates the snapshot. `NoChanges` short-circuits when the up plan is empty.
- `migrategen/render.go` — golang-migrate writer: scan output dir for existing
  `^(\d+)_…\.(up|down)\.sql$`, next version = max+1, zero-padded to 6 digits;
  write `{version}_{name}.up.sql` / `.down.sql`; sanitize the name into a slug.
- `cmd/migrategen/main.go` — thin CLI: `generate --name … [--config] [--out] [--dev-url]`,
  with env fallback `MIGRATEGEN_DEV_URL` and sensible defaults.

Default output dir: `<sqlc dir>/migrations` (i.e. `db/migrations`). Default
config: first of `db/sqlc.yaml`, `sqlc.yaml`, `db/sqlc.yml`, `sqlc.yml`.

## Dependencies

- Added `ariga.io/atlas` (`sql/sqlite`, `sql/postgres`, `sql/schema`,
  `sql/migrate`, `sqlclient`) + its transitive deps via `go mod tidy`.
- Reuses the existing `modernc.org/sqlite` driver for the in-memory dev DB.
- golang-migrate itself is **not** a build dependency — we only emit files in its
  naming convention.

## Key Atlas SDK calls (Atlas v1.2.2)

- `sqlite.Open(db) (migrate.Driver, error)` — the driver implements
  `schema.Inspector`, `schema.Differ`, `migrate.PlanApplier`.
- `driver.InspectSchema(ctx, name, opts) (*schema.Schema, error)` — DDL → model.
- `driver.SchemaDiff(from, to, ...DiffOption) ([]schema.Change, error)`.
- `driver.PlanChanges(ctx, name, changes, ...PlanOption) (*migrate.Plan, error)`;
  `plan.Changes[i].Cmd` are the SQL statements to write.
- Postgres mirror: `sqlclient.Open` returns a `*Client` embedding a
  `migrate.Driver` with the same methods.

## Verification (all passed)

End-to-end against the repo's real schema (`db/schema.sql`, engine `sqlite`):

1. `go build ./...`, `go vet`, `go test ./migrategen/` — all green.
2. **Init run** (no snapshot): `go run ./cmd/migrategen generate --name init`
   → `000001_init.up.sql` (`CREATE TABLE tasks …`) + `.down.sql` (`DROP TABLE tasks`).
3. **No-change run** → "no schema changes since last migration; nothing to do",
   no files written.
4. **Diff run** (added `priority INTEGER`):
   `go run ./cmd/migrategen generate --name add_priority`
   → up = `ALTER TABLE tasks ADD COLUMN priority …`; down = SQLite table-rebuild
   (`CREATE new_tasks` → `INSERT … SELECT` → `DROP tasks` → `RENAME`).
5. Generated SQL applied cleanly in real `sqlite3` in both directions.
6. Unit tests (`migrategen/diff_test.go`): create-table, add-column (up & down),
   no-change — using an in-memory SQLite dev DB, zero external services.

## Out of scope (not built)

- An `apply`/`up`/`down` runner (would wrap golang-migrate) — generation only.
- Data migrations / non-DDL changes.
- Multiple schema files or multiple `sql:` blocks in one `sqlc.yaml` (uses the
  first block; errors on multiple schema files).
- PostgreSQL path is wired but was not exercised here (no dev DB available);
  needs `--dev-url`.
