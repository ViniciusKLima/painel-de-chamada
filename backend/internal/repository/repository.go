package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{Pool: pool}
}

func (r *Repo) Ping(ctx context.Context) error {
	return r.Pool.Ping(ctx)
}
