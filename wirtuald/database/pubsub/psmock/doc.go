// package psmock contains a mocked implementation of the pubsub.Pubsub interface for use in tests
package psmock

//go:generate mockgen -destination ./psmock.go -package psmock github.com/onchainengineering/hmi-computer/v2/coderd/database/pubsub Pubsub
