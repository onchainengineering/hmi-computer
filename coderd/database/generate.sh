#!/usr/bin/env bash

# This script turns many *.sql.go files into a single queries.sql.go file. This
# is due to sqlc's behavior when using multiple sql files to output them to
# multiple Go files. We decided it would be cleaner to move these to a single
# file for readability. We should probably contribute the option to do this
# upstream instead, because this is quite janky.

set -euo pipefail

cd "$(dirname "$0")"

sqlc generate

first=true
for fi in queries/*.sql.go; do
    # Find the last line from the imports section and add 1.
    cut=$(grep -n ')' "$fi" | head -n 1 | cut -d: -f1)
    cut=$((cut + 1))

    # Copy the header from the first file only, ignoring the source comment.
    if $first; then
        head -n 4 < "$fi" | grep -v "source" > queries.sql.go
        first=false
    fi

    # Append the file past the imports section into queries.sql.go.
    tail -n "+$cut" < "$fi" >> queries.sql.go
done

# Remove temporary go files.
mv queries/querier.go .
rm -f queries/*.go

# Fix struct/interface names.
gofmt -w -r 'Querier -> querier' -- *.go
gofmt -w -r 'Queries -> sqlQuerier' -- *.go

# Ensure correct imports exist. Modules must all be downloaded so we get correct
# suggestions.
go mod download
goimports -w queries.sql.go
