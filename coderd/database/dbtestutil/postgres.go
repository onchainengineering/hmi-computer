package dbtestutil

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gofrs/flock"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/database/migrations"
	"github.com/coder/coder/v2/cryptorand"
)

type ConnectionParams struct {
	Username string
	Password string
	Host     string
	Port     string
	DBName   string
}

func (p ConnectionParams) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", p.Username, p.Password, p.Host, p.Port, p.DBName)
}

var (
	connectionParamsInitOnce       sync.Once
	defaultConnectionParams        ConnectionParams
	errDefaultConnectionParamsInit error
)

// initDefaultConnectionParams initializes the default connection parameters
// by checking if the database is running locally. If it is, it will use the
// local database. If it's not, it will start a new container and use that.
func initDefaultConnectionParams() error {
	params := ConnectionParams{
		Username: "postgres",
		Password: "postgres",
		Host:     "127.0.0.1",
		Port:     "5432",
		DBName:   "postgres",
	}
	dsn := params.DSN()
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		err = db.Ping()
		if closeErr := db.Close(); closeErr != nil {
			return xerrors.Errorf("close db: %w", closeErr)
		}
	}
	shouldOpenContainer := false
	if err != nil {
		errSubstrings := []string{
			"connection refused",          // this happens on Linux when there's nothing listening on the port
			"No connection could be made", // like above but Windows
		}
		errString := err.Error()
		for _, errSubstring := range errSubstrings {
			if strings.Contains(errString, errSubstring) {
				shouldOpenContainer = true
				break
			}
		}
	}
	if err != nil && shouldOpenContainer {
		// If there's no database running on the default port, we'll start a
		// postgres container. We won't be cleaning it up so it can be reused
		// by subsequent tests. It'll keep on running until the user terminates
		// it themselves.
		container, _, err := openContainer(DBContainerOptions{
			Name: "coder-test-postgres",
			Port: 5432,
		})
		if err != nil {
			return xerrors.Errorf("open container: %w", err)
		}
		params.Host = container.Host
		params.Port = container.Port
		dsn = params.DSN()

		// Retry connecting for a cumulative 5 seconds.
		// The fact that openContainer succeeded does not
		// mean that port forwarding is ready.
		for i := 0; i < 20; i++ {
			db, err = sql.Open("postgres", dsn)
			if err == nil {
				err = db.Ping()
				if closeErr := db.Close(); closeErr != nil {
					return xerrors.Errorf("close db, container: %w", closeErr)
				}
			}
			if err == nil {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	} else if err != nil {
		return xerrors.Errorf("open postgres connection: %w", err)
	}
	defaultConnectionParams = params
	return nil
}

type OpenOptions struct {
	DBFrom *string
}

type OpenOption func(*OpenOptions)

// WithDBFrom sets the template database to use when creating a new database.
// Overrides the DB_FROM environment variable.
func WithDBFrom(dbFrom string) OpenOption {
	return func(o *OpenOptions) {
		o.DBFrom = &dbFrom
	}
}

// Open creates a new PostgreSQL database instance.
// If there's a database running at localhost:5432, it will use that.
// Otherwise, it will start a new postgres container.
func Open(opts ...OpenOption) (string, func(), error) {
	connectionParamsInitOnce.Do(func() {
		errDefaultConnectionParamsInit = initDefaultConnectionParams()
	})
	if errDefaultConnectionParamsInit != nil {
		return "", func() {}, xerrors.Errorf("init default connection params: %w", errDefaultConnectionParamsInit)
	}

	openOptions := OpenOptions{}
	for _, opt := range opts {
		opt(&openOptions)
	}

	var (
		username = defaultConnectionParams.Username
		password = defaultConnectionParams.Password
		host     = defaultConnectionParams.Host
		port     = defaultConnectionParams.Port
	)

	// Use a time-based prefix to make it easier to find the database
	// when debugging.
	now := time.Now().Format("test_2006_01_02_15_04_05")
	dbSuffix, err := cryptorand.StringCharset(cryptorand.Lower, 10)
	if err != nil {
		return "", func() {}, xerrors.Errorf("generate db suffix: %w", err)
	}
	dbName := now + "_" + dbSuffix

	// if empty createDatabaseFromTemplate will create a new template db
	templateDBName := os.Getenv("DB_FROM")
	if openOptions.DBFrom != nil {
		templateDBName = *openOptions.DBFrom
	}
	if err = createDatabaseFromTemplate(defaultConnectionParams, dbName, templateDBName); err != nil {
		return "", func() {}, xerrors.Errorf("create database: %w", err)
	}

	cleanup := func() {
		cleanupDbURL := defaultConnectionParams.DSN()
		cleanupConn, err := sql.Open("postgres", cleanupDbURL)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cleanup database %q: failed to connect to postgres: %s\n", dbName, err.Error())
		}
		defer cleanupConn.Close()
		_, err = cleanupConn.Exec("DROP DATABASE " + dbName + ";")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to clean up database %q: %s\n", dbName, err.Error())
		}
	}

	dsn := ConnectionParams{
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
	}.DSN()
	return dsn, cleanup, nil
}

// createDatabaseFromTemplate creates a new database from a template database.
// If templateDBName is empty, it will create a new template database based on
// the current migrations, and name it "tpl_<migrations_hash>". Or if it's
// already been created, it will use that.
func createDatabaseFromTemplate(connParams ConnectionParams, newDBName string, templateDBName string) error {
	dbURL := connParams.DSN()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return xerrors.Errorf("connect to postgres: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			panic(err)
		}
	}()

	emptyTemplateDBName := templateDBName == ""
	if emptyTemplateDBName {
		templateDBName = fmt.Sprintf("tpl_%s", migrations.GetMigrationsHash()[:32])
	}
	_, err = db.Exec("CREATE DATABASE " + newDBName + " WITH TEMPLATE " + templateDBName)
	if err == nil {
		// Template database already exists and we successfully created the new database.
		return nil
	}
	tplDbDoesNotExistOccurred := strings.Contains(err.Error(), "template database") && strings.Contains(err.Error(), "does not exist")
	if (tplDbDoesNotExistOccurred && !emptyTemplateDBName) || !tplDbDoesNotExistOccurred {
		// First and case: user passed a templateDBName that doesn't exist.
		// Second and case: some other error.
		return xerrors.Errorf("create db with template: %w", err)
	}
	if !emptyTemplateDBName {
		// sanity check
		panic("templateDBName is not empty. there's a bug in the code above")
	}
	// The templateDBName is empty, so we need to create the template database.
	// We will use a tx to obtain a lock, so another test or process doesn't race with us.
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return xerrors.Errorf("begin tx: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			panic(err)
		}
	}()
	_, err = tx.Exec("SELECT pg_advisory_xact_lock(2137)")
	if err != nil {
		return xerrors.Errorf("acquire lock: %w", err)
	}

	// Someone else might have created the template db while we were waiting.
	tplDbExistsRes, err := tx.Query("SELECT 1 FROM pg_database WHERE datname = $1", templateDBName)
	if err != nil {
		return xerrors.Errorf("check if db exists: %w", err)
	}
	tplDbAlreadyExists := tplDbExistsRes.Next()
	if err := tplDbExistsRes.Close(); err != nil {
		return xerrors.Errorf("close tpl db exists res: %w", err)
	}
	if !tplDbAlreadyExists {
		// We will use a temporary template database to avoid race conditions. We will
		// rename it to the real template database name after we're sure it was fully
		// initialized.
		// It's dropped here to ensure that if a previous run of this function failed
		// midway, we don't encounter issues with the temporary database still existing.
		tmpTemplateDBName := "tmp_" + templateDBName
		if _, err := db.Exec("DROP DATABASE IF EXISTS " + tmpTemplateDBName); err != nil {
			return xerrors.Errorf("drop tmp template db: %w", err)
		}
		if _, err := db.Exec("CREATE DATABASE " + tmpTemplateDBName); err != nil {
			return xerrors.Errorf("create tmp template db: %w", err)
		}
		tplDbURL := ConnectionParams{
			Username: connParams.Username,
			Password: connParams.Password,
			Host:     connParams.Host,
			Port:     connParams.Port,
			DBName:   tmpTemplateDBName,
		}.DSN()
		tplDb, err := sql.Open("postgres", tplDbURL)
		if err != nil {
			return xerrors.Errorf("connect to template db: %w", err)
		}
		defer func() {
			if err := tplDb.Close(); err != nil {
				panic(err)
			}
		}()
		if err := migrations.Up(tplDb); err != nil {
			return xerrors.Errorf("migrate template db: %w", err)
		}
		if err := tplDb.Close(); err != nil {
			return xerrors.Errorf("close template db: %w", err)
		}
		if _, err := db.Exec("ALTER DATABASE " + tmpTemplateDBName + " RENAME TO " + templateDBName); err != nil {
			return xerrors.Errorf("rename tmp template db: %w", err)
		}
	}

	// Try to create the database again now that a template exists.
	if _, err = db.Exec("CREATE DATABASE " + newDBName + " WITH TEMPLATE " + templateDBName); err != nil {
		return xerrors.Errorf("create db with template after migrations: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}
	return nil
}

type DBContainerOptions struct {
	Port int
	Name string
}

type container struct {
	Resource *dockertest.Resource
	Pool     *dockertest.Pool
	Host     string
	Port     string
}

// OpenContainer creates a new PostgreSQL server using a Docker container. If port is nonzero, forward host traffic
// to that port to the database. If port is zero, allocate a free port from the OS.
// If name is set, we'll ensure that only one container is started with that name. If it's already running, we'll use that.
// Otherwise, we'll start a new container.
func openContainer(opts DBContainerOptions) (container, func(), error) {
	if opts.Name != "" {
		// We only want to start the container once per unique name,
		// so we take an inter-process lock to avoid concurrent test runs
		// racing with us.
		nameHash := sha256.Sum256([]byte(opts.Name))
		nameHashStr := hex.EncodeToString(nameHash[:])
		lock := flock.New(filepath.Join(os.TempDir(), "coder-postgres-container-"+nameHashStr[:8]))
		if err := lock.Lock(); err != nil {
			return container{}, nil, xerrors.Errorf("lock: %w", err)
		}
		defer func() {
			err := lock.Unlock()
			if err != nil {
				panic(err)
			}
		}()
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return container{}, nil, xerrors.Errorf("create pool: %w", err)
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "postgres")
	if err != nil {
		return container{}, nil, xerrors.Errorf("create tempdir: %w", err)
	}

	var resource *dockertest.Resource
	if opts.Name != "" {
		// If the container already exists, we'll use it.
		resource, _ = pool.ContainerByName(opts.Name)
	}
	if resource == nil {
		runOptions := dockertest.RunOptions{
			Repository: "gcr.io/coder-dev-1/postgres",
			Tag:        "13",
			Env: []string{
				"POSTGRES_PASSWORD=postgres",
				"POSTGRES_USER=postgres",
				"POSTGRES_DB=postgres",
				// The location for temporary database files!
				"PGDATA=/tmp",
				"listen_addresses = '*'",
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"5432/tcp": {{
					// Manually specifying a host IP tells Docker just to use an IPV4 address.
					// If we don't do this, we hit a fun bug:
					// https://github.com/moby/moby/issues/42442
					// where the ipv4 and ipv6 ports might be _different_ and collide with other running docker containers.
					HostIP:   "0.0.0.0",
					HostPort: strconv.FormatInt(int64(opts.Port), 10),
				}},
			},
			Mounts: []string{
				// The postgres image has a VOLUME parameter in it's image.
				// If we don't mount at this point, Docker will allocate a
				// volume for this directory.
				//
				// This isn't used anyways, since we override PGDATA.
				fmt.Sprintf("%s:/var/lib/postgresql/data", tempDir),
			},
			Cmd: []string{"-c", "max_connections=1000"},
		}
		if opts.Name != "" {
			runOptions.Name = opts.Name
		}
		resource, err = pool.RunWithOptions(&runOptions, func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
			config.Tmpfs = map[string]string{
				"/tmp": "rw",
			}
		})
		if err != nil {
			return container{}, nil, xerrors.Errorf("could not start resource: %w", err)
		}
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	host, port, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		return container{}, nil, xerrors.Errorf("split host and port: %w", err)
	}

	// wait for a cumulative 60 * 250ms = 15 seconds for the database to start
	for i := 0; i < 60; i++ {
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}
		_, err = resource.Exec([]string{"pg_isready", "-h", "127.0.0.1"}, dockertest.ExecOptions{
			StdOut: stdout,
			StdErr: stderr,
		})
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		return container{}, nil, xerrors.Errorf("pg_isready: %w", err)
	}

	return container{
			Host:     host,
			Port:     port,
			Resource: resource,
			Pool:     pool,
		}, func() {
			_ = pool.Purge(resource)
			_ = os.RemoveAll(tempDir)
		}, nil
}

// OpenContainerized creates a new PostgreSQL server using a Docker container.  If port is nonzero, forward host traffic
// to that port to the database.  If port is zero, allocate a free port from the OS.
func OpenContainerized(opts DBContainerOptions) (string, func(), error) {
	container, containerCleanup, err := openContainer(opts)
	defer func() {
		if err != nil {
			containerCleanup()
		}
	}()
	if err != nil {
		return "", nil, xerrors.Errorf("open container: %w", err)
	}
	dbURL := ConnectionParams{
		Username: "postgres",
		Password: "postgres",
		Host:     container.Host,
		Port:     container.Port,
		DBName:   "postgres",
	}.DSN()

	// Docker should hard-kill the container after 120 seconds.
	err = container.Resource.Expire(120)
	if err != nil {
		return "", nil, xerrors.Errorf("expire resource: %w", err)
	}

	container.Pool.MaxWait = 120 * time.Second

	// Record the error that occurs during the retry.
	// The 'pool' pkg hardcodes a deadline error devoid
	// of any useful context.
	var retryErr error
	err = container.Pool.Retry(func() error {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			retryErr = xerrors.Errorf("open postgres: %w", err)
			return retryErr
		}
		defer db.Close()

		err = db.Ping()
		if err != nil {
			retryErr = xerrors.Errorf("ping postgres: %w", err)
			return retryErr
		}

		err = migrations.Up(db)
		if err != nil {
			retryErr = xerrors.Errorf("migrate db: %w", err)
			// Only try to migrate once.
			return backoff.Permanent(retryErr)
		}

		return nil
	})
	if err != nil {
		return "", nil, retryErr
	}

	return dbURL, containerCleanup, nil
}
