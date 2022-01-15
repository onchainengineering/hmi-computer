# Runs migrations to output a dump of the database.
database/dump.sql: $(wildcard database/migrations/*.sql)
	go run database/dump/main.go

# Generates Go code for querying the database.
database/generate: database/dump.sql database/query.sql
	cd database && sqlc generate && rm db_tmp.go
	cd database && gofmt -w -r 'Querier -> querier' *.go
	cd database && gofmt -w -r 'Queries -> sqlQuerier' *.go
.PHONY: database/generate

fmt/prettier:
	@echo "--- prettier"
# Avoid writing files in CI to reduce file write activity
ifdef CI
	yarn run format:check
else
	yarn run format:write
endif
.PHONY: fmt/prettier

fmt: fmt/prettier
.PHONY: fmt

gen: database/generate peerbroker/proto provisionersdk/proto
.PHONY: gen

# Lightweight build for coderd that doesn't require building the front-end
# first. This lets us quickly spin up an API process for development,
# while using `next dev` to handle the front-end.
dev/go/coderd:
	go build -o build/coderd cmd/coderd/main.go
.PHONY: dev/build/go/coderd-dev

build/go/coderd:
	go build -tags=embed -o build/coderd cmd/coderd/main.go
.PHONY: build/go/coderd

build/go: build/go/coderd
.PHONY: build/go

build/ui: 
	yarn build
	yarn export
.PHONY: build/ui

build: build/go/coderd build/ui
.PHONY: build

# Generates the protocol files.
peerbroker/proto: peerbroker/proto/peerbroker.proto
	cd peerbroker/proto && protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-drpc_out=. \
		--go-drpc_opt=paths=source_relative \
		./peerbroker.proto
.PHONY: peerbroker/proto

# Generates the protocol files.
provisionersdk/proto: provisionersdk/proto/provisioner.proto
	cd provisionersdk/proto && protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-drpc_out=. \
		--go-drpc_opt=paths=source_relative \
		./provisioner.proto
.PHONY: provisionersdk/proto