// Code generated by sqlc. DO NOT EDIT.
// source: parameter_values.sql

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const deleteParameterValueByID = `-- name: DeleteParameterValueByID :exec
DELETE FROM
	parameter_values
WHERE
	id = $1
`

func (q *sqlQuerier) DeleteParameterValueByID(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteParameterValueByID, id)
	return err
}

const getParameterValueByScopeAndName = `-- name: GetParameterValueByScopeAndName :one
SELECT
	id, created_at, updated_at, scope, scope_id, name, source_scheme, source_value, destination_scheme
FROM
	parameter_values
WHERE
	scope = $1
	AND scope_id = $2
	AND NAME = $3
LIMIT
	1
`

type GetParameterValueByScopeAndNameParams struct {
	Scope   ParameterScope `db:"scope" json:"scope"`
	ScopeID string         `db:"scope_id" json:"scope_id"`
	Name    string         `db:"name" json:"name"`
}

func (q *sqlQuerier) GetParameterValueByScopeAndName(ctx context.Context, arg GetParameterValueByScopeAndNameParams) (ParameterValue, error) {
	row := q.db.QueryRowContext(ctx, getParameterValueByScopeAndName, arg.Scope, arg.ScopeID, arg.Name)
	var i ParameterValue
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Scope,
		&i.ScopeID,
		&i.Name,
		&i.SourceScheme,
		&i.SourceValue,
		&i.DestinationScheme,
	)
	return i, err
}

const getParameterValuesByScope = `-- name: GetParameterValuesByScope :many
SELECT
	id, created_at, updated_at, scope, scope_id, name, source_scheme, source_value, destination_scheme
FROM
	parameter_values
WHERE
	scope = $1
	AND scope_id = $2
`

type GetParameterValuesByScopeParams struct {
	Scope   ParameterScope `db:"scope" json:"scope"`
	ScopeID string         `db:"scope_id" json:"scope_id"`
}

func (q *sqlQuerier) GetParameterValuesByScope(ctx context.Context, arg GetParameterValuesByScopeParams) ([]ParameterValue, error) {
	rows, err := q.db.QueryContext(ctx, getParameterValuesByScope, arg.Scope, arg.ScopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ParameterValue
	for rows.Next() {
		var i ParameterValue
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Scope,
			&i.ScopeID,
			&i.Name,
			&i.SourceScheme,
			&i.SourceValue,
			&i.DestinationScheme,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertParameterValue = `-- name: InsertParameterValue :one
INSERT INTO
	parameter_values (
		id,
		"name",
		created_at,
		updated_at,
		scope,
		scope_id,
		source_scheme,
		source_value,
		destination_scheme
	)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at, scope, scope_id, name, source_scheme, source_value, destination_scheme
`

type InsertParameterValueParams struct {
	ID                uuid.UUID                  `db:"id" json:"id"`
	Name              string                     `db:"name" json:"name"`
	CreatedAt         time.Time                  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time                  `db:"updated_at" json:"updated_at"`
	Scope             ParameterScope             `db:"scope" json:"scope"`
	ScopeID           string                     `db:"scope_id" json:"scope_id"`
	SourceScheme      ParameterSourceScheme      `db:"source_scheme" json:"source_scheme"`
	SourceValue       string                     `db:"source_value" json:"source_value"`
	DestinationScheme ParameterDestinationScheme `db:"destination_scheme" json:"destination_scheme"`
}

func (q *sqlQuerier) InsertParameterValue(ctx context.Context, arg InsertParameterValueParams) (ParameterValue, error) {
	row := q.db.QueryRowContext(ctx, insertParameterValue,
		arg.ID,
		arg.Name,
		arg.CreatedAt,
		arg.UpdatedAt,
		arg.Scope,
		arg.ScopeID,
		arg.SourceScheme,
		arg.SourceValue,
		arg.DestinationScheme,
	)
	var i ParameterValue
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Scope,
		&i.ScopeID,
		&i.Name,
		&i.SourceScheme,
		&i.SourceValue,
		&i.DestinationScheme,
	)
	return i, err
}
