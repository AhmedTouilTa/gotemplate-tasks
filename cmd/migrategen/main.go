// Command migrategen generates golang-migrate migration files by diffing a
// project's sqlc schema against the last schema it generated for.
//
// Usage:
//
//	migrategen generate --name add_priority [--config db/sqlc.yaml] [--out db/migrations] [--dev-url ...]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"gotemplates/todo/migrategen"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "generate" {
		usage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	name := fs.String("name", "", "migration name (required), e.g. add_priority")
	config := fs.String("config", defaultConfig(), "path to sqlc.yaml")
	out := fs.String("out", "", "output directory (default: <sqlc dir>/migrations)")
	devURL := fs.String("dev-url", os.Getenv("MIGRATEGEN_DEV_URL"), "Atlas dev database URL (required for postgresql; env MIGRATEGEN_DEV_URL)")
	fs.Parse(os.Args[2:])

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required")
		fs.Usage()
		os.Exit(2)
	}

	cfg, err := migrategen.LoadConfig(*config, *out, *devURL)
	if err != nil {
		fail(err)
	}

	res, err := migrategen.Generate(context.Background(), migrategen.Options{
		Config: cfg,
		Name:   *name,
	})
	if err != nil {
		fail(err)
	}

	if res.NoChanges {
		fmt.Println("no schema changes since last migration; nothing to do")
		return
	}
	fmt.Printf("wrote %s\n      %s\n", res.UpPath, res.DownPath)
}

// defaultConfig picks the first sqlc.yaml that exists in the usual spots.
func defaultConfig() string {
	for _, p := range []string{"db/sqlc.yaml", "sqlc.yaml", "db/sqlc.yml", "sqlc.yml"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "db/sqlc.yaml"
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrategen generate --name <name> [--config db/sqlc.yaml] [--out dir] [--dev-url url]")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
