package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS instances (
            id          TEXT        PRIMARY KEY,
            first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
            last_seen   TIMESTAMPTZ NOT NULL,
            version     TEXT        NOT NULL
        );

        CREATE TABLE IF NOT EXISTS pings (
            id          BIGSERIAL   PRIMARY KEY,
            instance_id TEXT        NOT NULL REFERENCES instances(id),
            pinged_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
            version     TEXT        NOT NULL
        );

        CREATE INDEX IF NOT EXISTS idx_pings_pinged_at   ON pings (pinged_at);
        CREATE INDEX IF NOT EXISTS idx_pings_instance_id ON pings (instance_id);
    `)
	return err
}

func New(ctx context.Context) (*DB, error) {
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) RecordPing(ctx context.Context, instanceID, version string) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var lastSeen *time.Time
	err = tx.QueryRow(ctx,
		`SELECT last_seen FROM instances WHERE id = $1`, instanceID,
	).Scan(&lastSeen)

	if err == nil && lastSeen != nil && time.Since(*lastSeen) < 24*time.Hour {
		return nil
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO instances (id, last_seen, version)
        VALUES ($1, now(), $2)
        ON CONFLICT (id) DO UPDATE
            SET last_seen = now(),
                version   = EXCLUDED.version
    `, instanceID, version)
	if err != nil {
		return fmt.Errorf("upsert instance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO pings (instance_id, version) VALUES ($1, $2)`,
		instanceID, version,
	)
	if err != nil {
		return fmt.Errorf("insert ping: %w", err)
	}

	return tx.Commit(ctx)
}

func (d *DB) GetStats(ctx context.Context) (total, active int, err error) {
	err = d.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM instances`,
	).Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("count total: %w", err)
	}

	err = d.pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM instances
        WHERE last_seen > now() - interval '48 hours'
    `).Scan(&active)
	if err != nil {
		return 0, 0, fmt.Errorf("count active: %w", err)
	}

	return total, active, nil
}
