// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: schema.sql

package db

import (
	"context"
)

const pop = `-- name: Pop :one
select id, sendto, url, title, links, status from queue where status = ''
`

func (q *Queries) Pop(ctx context.Context) (Queue, error) {
	row := q.db.QueryRowContext(ctx, pop)
	var i Queue
	err := row.Scan(
		&i.ID,
		&i.Sendto,
		&i.Url,
		&i.Title,
		&i.Links,
		&i.Status,
	)
	return i, err
}

const push = `-- name: Push :exec
insert into queue(sendto, url) values (?, ?)
`

type PushParams struct {
	Sendto string
	Url    string
}

func (q *Queries) Push(ctx context.Context, arg PushParams) error {
	_, err := q.db.ExecContext(ctx, push, arg.Sendto, arg.Url)
	return err
}

const setFailed = `-- name: SetFailed :exec
update queue set status = ?
`

func (q *Queries) SetFailed(ctx context.Context, status string) error {
	_, err := q.db.ExecContext(ctx, setFailed, status)
	return err
}

const setSuccess = `-- name: SetSuccess :exec
update queue set
	title = ?,
	links = ?,
	status='success'
`

type SetSuccessParams struct {
	Title string
	Links string
}

func (q *Queries) SetSuccess(ctx context.Context, arg SetSuccessParams) error {
	_, err := q.db.ExecContext(ctx, setSuccess, arg.Title, arg.Links)
	return err
}