package dbauthz

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/xerrors"

	"cdr.dev/slog"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/rbac"
	"github.com/google/uuid"
)

var _ database.Store = (*AuthzQuerier)(nil)

var (
	// NoActorError wraps ErrNoRows for the api to return a 404. This is the correct
	// response when the user is not authorized.
	NoActorError = xerrors.Errorf("no authorization actor in context: %w", sql.ErrNoRows)
)

// NotAuthorizedError is a sentinel error that unwraps to sql.ErrNoRows.
// This allows the internal error to be read by the caller if needed. Otherwise
// it will be handled as a 404.
type NotAuthorizedError struct {
	Err error
}

func (e NotAuthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", e.Err.Error())
}

// Unwrap will always unwrap to a sql.ErrNoRows so the API returns a 404.
// So 'errors.Is(err, sql.ErrNoRows)' will always be true.
func (NotAuthorizedError) Unwrap() error {
	return sql.ErrNoRows
}

func LogNotAuthorizedError(ctx context.Context, logger slog.Logger, err error) error {
	// Only log the errors if it is an UnauthorizedError error.
	internalError := new(rbac.UnauthorizedError)
	if err != nil && xerrors.As(err, internalError) {
		logger.Debug(ctx, "unauthorized",
			slog.F("internal", internalError.Internal()),
			slog.F("input", internalError.Input()),
			slog.Error(err),
		)
	}
	return NotAuthorizedError{
		Err: err,
	}
}

// AuthzQuerier is a wrapper around the database store that performs authorization
// checks before returning data. All AuthzQuerier methods expect an authorization
// subject present in the context. If no subject is present, most methods will
// fail.
//
// Use WithAuthorizeContext to set the authorization subject in the context for
// the common user case.
type AuthzQuerier struct {
	db   database.Store
	auth rbac.Authorizer
	log  slog.Logger
}

func New(db database.Store, authorizer rbac.Authorizer, logger slog.Logger) *AuthzQuerier {
	return &AuthzQuerier{
		db:   db,
		auth: authorizer,
		log:  logger,
	}
}

// authorizeContext is a helper function to authorize an action on an object.
func (q *AuthzQuerier) authorizeContext(ctx context.Context, action rbac.Action, object rbac.Objecter) error {
	act, ok := ActorFromContext(ctx)
	if !ok {
		return NoActorError
	}

	err := q.auth.Authorize(ctx, act, action, object.RBACObject())
	if err != nil {
		return LogNotAuthorizedError(ctx, q.log, err)
	}
	return nil
}

type authContextKey struct{}

// ActorFromContext returns the authorization subject from the context.
// All authentication flows should set the authorization subject in the context.
// If no actor is present, the function returns false.
func ActorFromContext(ctx context.Context) (rbac.Subject, bool) {
	a, ok := ctx.Value(authContextKey{}).(rbac.Subject)
	return a, ok
}

func WithAuthorizeContext(ctx context.Context, actor rbac.Subject) context.Context {
	return context.WithValue(ctx, authContextKey{}, actor)
}

func WithAuthorizeSystemContext(ctx context.Context, roles rbac.ExpandableRoles) context.Context {
	// TODO: Add protections to search for user roles. If user roles are found,
	// this should panic. That is a developer error that should be caught
	// in unit tests.
	return context.WithValue(ctx, authContextKey{}, rbac.Subject{
		ID:     uuid.Nil.String(),
		Roles:  roles,
		Scope:  rbac.ScopeAll,
		Groups: []string{},
	})
}

//
// Generic functions used to implement the database.Store methods.
//

// insert runs an rbac.ActionCreate on the rbac object argument before
// running the insertFunc. The insertFunc is expected to return the object that
// was inserted.
func insert[
	ObjectType any,
	ArgumentType any,
	Insert func(ctx context.Context, arg ArgumentType) (ObjectType, error),
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	object rbac.Objecter,
	insertFunc Insert,
) Insert {
	return func(ctx context.Context, arg ArgumentType) (empty ObjectType, err error) {
		// Fetch the rbac subject
		act, ok := ActorFromContext(ctx)
		if !ok {
			return empty, NoActorError
		}

		// Authorize the action
		err = authorizer.Authorize(ctx, act, rbac.ActionCreate, object.RBACObject())
		if err != nil {
			return empty, LogNotAuthorizedError(ctx, logger, err)
		}

		// Insert the database object
		return insertFunc(ctx, arg)
	}
}

func deleteQ[
	ObjectType rbac.Objecter,
	ArgumentType any,
	Fetch func(ctx context.Context, arg ArgumentType) (ObjectType, error),
	Delete func(ctx context.Context, arg ArgumentType) error,
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	fetchFunc Fetch,
	deleteFunc Delete,
) Delete {
	return fetchAndExec(logger, authorizer,
		rbac.ActionDelete, fetchFunc, deleteFunc)
}

func updateWithReturn[
	ObjectType rbac.Objecter,
	ArgumentType any,
	Fetch func(ctx context.Context, arg ArgumentType) (ObjectType, error),
	UpdateQuery func(ctx context.Context, arg ArgumentType) (ObjectType, error),
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	fetchFunc Fetch,
	updateQuery UpdateQuery,
) UpdateQuery {
	return fetchAndQuery(logger, authorizer, rbac.ActionUpdate, fetchFunc, updateQuery)
}

func update[
	ObjectType rbac.Objecter,
	ArgumentType any,
	Fetch func(ctx context.Context, arg ArgumentType) (ObjectType, error),
	Exec func(ctx context.Context, arg ArgumentType) error,
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	fetchFunc Fetch,
	updateExec Exec,
) Exec {
	return fetchAndExec(logger, authorizer, rbac.ActionUpdate, fetchFunc, updateExec)
}

// fetch is a generic function that wraps a database
// query function (returns an object and an error) with authorization. The
// returned function has the same arguments as the database function.
//
// The database query function will **ALWAYS** hit the database, even if the
// user cannot read the resource. This is because the resource details are
// required to run a proper authorization check.
func fetch[
	ArgumentType any,
	ObjectType rbac.Objecter,
	DatabaseFunc func(ctx context.Context, arg ArgumentType) (ObjectType, error),
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	f DatabaseFunc,
) DatabaseFunc {
	return func(ctx context.Context, arg ArgumentType) (empty ObjectType, err error) {
		// Fetch the rbac subject
		act, ok := ActorFromContext(ctx)
		if !ok {
			return empty, NoActorError
		}

		// Fetch the database object
		object, err := f(ctx, arg)
		if err != nil {
			return empty, xerrors.Errorf("fetch object: %w", err)
		}

		// Authorize the action
		err = authorizer.Authorize(ctx, act, rbac.ActionRead, object.RBACObject())
		if err != nil {
			return empty, LogNotAuthorizedError(ctx, logger, err)
		}

		return object, nil
	}
}

// fetchAndExec uses fetchAndQuery but only returns the error. The naming comes
// from SQL 'exec' functions which only return an error.
// See fetchAndQuery for more information.
func fetchAndExec[
	ObjectType rbac.Objecter,
	ArgumentType any,
	Fetch func(ctx context.Context, arg ArgumentType) (ObjectType, error),
	Exec func(ctx context.Context, arg ArgumentType) error,
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	action rbac.Action,
	fetchFunc Fetch,
	execFunc Exec,
) Exec {
	f := fetchAndQuery(logger, authorizer, action, fetchFunc, func(ctx context.Context, arg ArgumentType) (empty ObjectType, err error) {
		return empty, execFunc(ctx, arg)
	})
	return func(ctx context.Context, arg ArgumentType) error {
		_, err := f(ctx, arg)
		return err
	}
}

// fetchAndQuery is a generic function that wraps a database fetch and query.
// A query has potential side effects in the database (update, delete, etc).
// The fetch is used to know which rbac object the action should be asserted on
// **before** the query runs. The returns from the fetch are only used to
// assert rbac. The final return of this function comes from the Query function.
func fetchAndQuery[
	ObjectType rbac.Objecter,
	ArgumentType any,
	Fetch func(ctx context.Context, arg ArgumentType) (ObjectType, error),
	Query func(ctx context.Context, arg ArgumentType) (ObjectType, error),
](
	logger slog.Logger,
	authorizer rbac.Authorizer,
	action rbac.Action,
	fetchFunc Fetch,
	queryFunc Query,
) Query {
	return func(ctx context.Context, arg ArgumentType) (empty ObjectType, err error) {
		// Fetch the rbac subject
		act, ok := ActorFromContext(ctx)
		if !ok {
			return empty, NoActorError
		}

		// Fetch the database object
		object, err := fetchFunc(ctx, arg)
		if err != nil {
			return empty, xerrors.Errorf("fetch object: %w", err)
		}

		// Authorize the action
		err = authorizer.Authorize(ctx, act, action, object.RBACObject())
		if err != nil {
			return empty, LogNotAuthorizedError(ctx, logger, err)
		}

		return queryFunc(ctx, arg)
	}
}

// fetchWithPostFilter is like fetch, but works with lists of objects.
// SQL filters are much more optimal.
func fetchWithPostFilter[
	ArgumentType any,
	ObjectType rbac.Objecter,
	DatabaseFunc func(ctx context.Context, arg ArgumentType) ([]ObjectType, error),
](
	authorizer rbac.Authorizer,
	f DatabaseFunc,
) DatabaseFunc {
	return func(ctx context.Context, arg ArgumentType) (empty []ObjectType, err error) {
		// Fetch the rbac subject
		act, ok := ActorFromContext(ctx)
		if !ok {
			return empty, NoActorError
		}

		// Fetch the database object
		objects, err := f(ctx, arg)
		if err != nil {
			return nil, xerrors.Errorf("fetch object: %w", err)
		}

		// Authorize the action
		return rbac.Filter(ctx, authorizer, act, rbac.ActionRead, objects)
	}
}

// prepareSQLFilter is a helper function that prepares a SQL filter using the
// given authorization context.
func prepareSQLFilter(ctx context.Context, authorizer rbac.Authorizer, action rbac.Action, resourceType string) (rbac.PreparedAuthorized, error) {
	act, ok := ActorFromContext(ctx)
	if !ok {
		return nil, xerrors.Errorf("no authorization actor in context")
	}

	return authorizer.Prepare(ctx, act, action, resourceType)
}
