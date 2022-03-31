// Code generated by sqlc. DO NOT EDIT.
// source: organization_members.sql

package database

import (
	"context"
	"time"

	"github.com/lib/pq"
)

const getOrganizationMemberByUserID = `-- name: GetOrganizationMemberByUserID :one
SELECT
	organization_id, user_id, created_at, updated_at, roles
FROM
	organization_members
WHERE
	organization_id = $1
	AND user_id = $2
LIMIT
	1
`

type GetOrganizationMemberByUserIDParams struct {
	OrganizationID string `db:"organization_id" json:"organization_id"`
	UserID         string `db:"user_id" json:"user_id"`
}

func (q *sqlQuerier) GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error) {
	row := q.db.QueryRowContext(ctx, getOrganizationMemberByUserID, arg.OrganizationID, arg.UserID)
	var i OrganizationMember
	err := row.Scan(
		&i.OrganizationID,
		&i.UserID,
		&i.CreatedAt,
		&i.UpdatedAt,
		pq.Array(&i.Roles),
	)
	return i, err
}

const insertOrganizationMember = `-- name: InsertOrganizationMember :one
INSERT INTO
	organization_members (
		organization_id,
		user_id,
		created_at,
		updated_at,
		roles
	)
VALUES
	($1, $2, $3, $4, $5) RETURNING organization_id, user_id, created_at, updated_at, roles
`

type InsertOrganizationMemberParams struct {
	OrganizationID string    `db:"organization_id" json:"organization_id"`
	UserID         string    `db:"user_id" json:"user_id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	Roles          []string  `db:"roles" json:"roles"`
}

func (q *sqlQuerier) InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error) {
	row := q.db.QueryRowContext(ctx, insertOrganizationMember,
		arg.OrganizationID,
		arg.UserID,
		arg.CreatedAt,
		arg.UpdatedAt,
		pq.Array(arg.Roles),
	)
	var i OrganizationMember
	err := row.Scan(
		&i.OrganizationID,
		&i.UserID,
		&i.CreatedAt,
		&i.UpdatedAt,
		pq.Array(&i.Roles),
	)
	return i, err
}
