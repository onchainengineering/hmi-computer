package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/postgres"
)

func main() {
	connection, closeFn, err := postgres.Open()
	if err != nil {
		panic(err)
	}
	defer closeFn()
	db, err := sql.Open("postgres", connection)
	if err != nil {
		panic(err)
	}
	err = database.MigrateUp(db)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command(
		"pg_dump",
		"--schema-only",
		connection,
		"--no-privileges",
		"--no-owner",
		"--no-comments",

		// We never want to manually generate
		// queries executing against this table.
		"--exclude-table=schema_migrations",
	)
	cmd.Env = []string{
		"PGTZ=UTC",
		"PGCLIENTENCODING=UTF8",
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	for _, sed := range []string{
		// Remove all comments.
		"/^--/d",
		// Public is implicit in the schema.
		"s/ public\\./ /",
		// Remove database settings.
		"s/SET.*;//g",
		// Remove select statements. These aren't useful
		// to a reader of the dump.
		"s/SELECT.*;//g",
		// Removes multiple newlines.
		"/^$/N;/^\\n$/D",
	} {
		cmd := exec.Command("sed", "-e", sed)
		cmd.Stdin = bytes.NewReader(output.Bytes())
		output = bytes.Buffer{}
		cmd.Stdout = &output
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			panic(err)
		}
	}

	dump := fmt.Sprintf("-- Code generated by 'make coderd/database/generate'. DO NOT EDIT.\n%s", output.Bytes())
	_, mainPath, _, ok := runtime.Caller(0)
	if !ok {
		panic("couldn't get caller path")
	}
	err = os.WriteFile(filepath.Join(mainPath, "..", "..", "dump.sql"), []byte(dump), 0600)
	if err != nil {
		panic(err)
	}
}
