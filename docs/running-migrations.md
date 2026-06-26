# Running migrations with the golang-migrate CLI

`migrategen` only **generates** migration files (`db/migrations/*.up.sql` /
`*.down.sql`). To **apply** them to a database you use the
[golang-migrate](https://github.com/golang-migrate/migrate) CLI.

## Install the CLI (with a SQLite driver)

> **Gotcha:** the Homebrew `golang-migrate` binary is **not** built with any
> SQLite driver. Both `sqlite://` and `sqlite3://` fail with
> `unknown driver sqlite (forgotten import?)`. SQLite support is an opt-in build
> tag, so you must build the CLI yourself.

This project already depends on the pure-Go `modernc.org/sqlite` driver, whose
golang-migrate driver name is **`sqlite`**. Build the CLI with that tag:

```sh
go install -tags 'sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

This installs `migrate` to `$(go env GOPATH)/bin` (usually `~/go/bin`).

If you also have the Homebrew build installed, it will **shadow** the working one
when `/opt/homebrew/bin` comes before `~/go/bin` on your `PATH`. Remove it (it
can't do SQLite anyway):

```sh
brew uninstall golang-migrate
```

Verify the working binary is the one on your `PATH`:

```sh
which migrate            # should be ~/go/bin/migrate
type -a migrate          # lists every migrate on PATH, in order
```

## Apply / roll back

The DB URL is `sqlite://<path-to-db-file>`.

```sh
# Apply all pending migrations
migrate -path db/migrations -database "sqlite://test.db" up

# Roll back the most recent migration
migrate -path db/migrations -database "sqlite://test.db" down 1

# Show the current version
migrate -path db/migrations -database "sqlite://test.db" version
```

## Recovering from a "dirty" state

If a migration fails partway, golang-migrate marks that version **dirty** and
refuses to continue:

```
error: Dirty database version 1. Fix and force version.
```

Inspect the tracking table:

```sh
sqlite3 test.db "SELECT version, dirty FROM schema_migrations;"
# 1|1   ->  version 1, dirty = 1
```

Once the actual schema matches the intended state of that version, mark it clean
with `force` (this only updates the tracking row; it does **not** run any SQL):

```sh
migrate -path db/migrations -database "sqlite://test.db" force 1
```

## Note: GORM auto-migrate vs. golang-migrate

`main.go` also runs GORM `AutoMigrate` on startup. If the app has already created
the tables, applying `000001_init.up.sql` will fail with
`table tasks already exists`, and leave the version dirty (see above). Pick one
source of truth for schema management:

- let **golang-migrate** own the schema (drop the GORM `AutoMigrate` call), or
- use golang-migrate only for changes GORM doesn't handle.

Otherwise the two collide on a fresh database.
