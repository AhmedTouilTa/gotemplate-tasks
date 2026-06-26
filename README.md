# GoTemplateLearn

A small todo app (Gin + sqlc + tmpl), plus **migrategen** — a library and CLI
that generates [golang-migrate](https://github.com/golang-migrate/migrate)
migration files by diffing your sqlc schema.

## migrategen

`migrategen` reads your `sqlc.yaml`, diffs the current `schema.sql` against the
last schema it generated for (stored in `db/migrations/.snapshot.sql`), and
writes a versioned up/down migration pair.

It uses the [Atlas](https://atlasgo.io) SDK to parse DDL and compute the diff,
so column adds, drops, renames, index and constraint changes are handled per
dialect — including SQLite's table-rebuild dance for operations SQLite can't do
in place.

### Usage

```sh
# After changing db/schema.sql:
go run ./cmd/migrategen generate --name add_priority
# wrote db/migrations/000002_add_priority.up.sql
#       db/migrations/000002_add_priority.down.sql
```

Flags:

| Flag        | Default                | Meaning                                            |
|-------------|------------------------|----------------------------------------------------|
| `--name`    | *(required)*           | Migration name, used in the filename               |
| `--config`  | `db/sqlc.yaml`         | Path to your sqlc config                           |
| `--out`     | `<sqlc dir>/migrations`| Output directory for migration files               |
| `--dev-url` | `$MIGRATEGEN_DEV_URL`  | Atlas dev database URL (PostgreSQL only)           |

Running with no schema change writes nothing and reports "no changes".

To **apply** the generated migrations to a database, see
[docs/running-migrations.md](docs/running-migrations.md) — note the Homebrew
`migrate` binary lacks a SQLite driver and won't work.

### Engines

- **SQLite** — works out of the box. Diffing runs against an in-memory SQLite
  database (no external dependency).
- **PostgreSQL** — requires a throwaway "dev" database to normalize DDL. Provide
  one via `--dev-url` or `MIGRATEGEN_DEV_URL`, e.g.
  `postgres://localhost:5432/dev?sslmode=disable`.

### How it tracks state

`db/migrations/.snapshot.sql` holds the schema the last migration was generated
from; commit it alongside the migration files. The first run (no snapshot)
diffs against an empty database, producing an "init" migration that creates
everything.

### As a library

```go
cfg, _ := migrategen.LoadConfig("db/sqlc.yaml", "", "")
res, _ := migrategen.Generate(context.Background(), migrategen.Options{
    Config: cfg,
    Name:   "add_priority",
})
```
