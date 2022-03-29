package tk

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
)

const DatabaseURL = "DATABASE_URL"

var (
	// ErrFailedGettingPoolConfig failed getting db pool config error
	ErrFailedGettingPoolConfig = errors.New("failed to get db pool config")
	// ErrFailedGettingPool failed getting db pool error
	ErrFailedGettingPool = errors.New("failed to get db pool")
)

// GetPool attempt to connect to DB
func GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(os.Getenv(DatabaseURL))
	if err != nil {
		return nil, ErrFailedGettingPoolConfig
	}

	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return pool, ErrFailedGettingPool
	}

	return pool, nil
}
