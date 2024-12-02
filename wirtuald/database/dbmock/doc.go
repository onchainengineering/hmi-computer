// package dbmock contains a mocked implementation of the database.Store interface for use in tests
package dbmock

//go:generate mockgen -destination ./dbmock.go -package dbmock github.com/onchainengineering/hmi-computer/v2/coderd/database Store
