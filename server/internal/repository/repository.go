package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"campus-fellow/server/internal/config"
	"github.com/go-sql-driver/mysql"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Repository struct{ db *sql.DB }

func Open(ctx context.Context, cfg config.Config) (*Repository, error) {
	c := mysql.NewConfig()
	c.User, c.Passwd, c.Net, c.Addr, c.DBName = cfg.DBUser, cfg.DBPassword, "tcp", net.JoinHostPort(cfg.DBHost, cfg.DBPort), cfg.DBName
	c.ParseTime, c.Loc, c.ClientFoundRows = true, time.UTC, true
	if err := c.Apply(mysql.Charset("utf8mb4", "utf8mb4_0900_ai_ci")); err != nil {
		return nil, err
	}
	c.Params = map[string]string{"time_zone": "'+00:00'"}
	c.Timeout, c.ReadTimeout, c.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	connector, err := mysql.NewConnector(c)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect MySQL (check DB_* settings): %w", err)
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error                   { return r.db.Close() }
func (r *Repository) Ping(ctx context.Context) error { return r.db.PingContext(ctx) }

// MySQL DDL auto-commits. Migrations must be restartable and recorded only after every statement succeeds.
func (r *Repository) Migrate(ctx context.Context) error {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(SHA2(CONCAT(DATABASE(), ':migrations'), 256), 30)").Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return errors.New("could not acquire migration lock")
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(SHA2(CONCAT(DATABASE(), ':migrations'), 256))")
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)) ENGINE=InnoDB`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		var count int
		if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", entry.Name()).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		for _, statement := range strings.Split(string(body), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s: %w", entry.Name(), err)
			}
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES (?)", entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

func Duplicate(err error) bool {
	var e *mysql.MySQLError
	return errors.As(err, &e) && e.Number == 1062
}

// scanner keeps row mapping identical for QueryRow and Query results.
type scanner interface{ Scan(dest ...any) error }

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func optionalTime(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := timestamp(t.Time)
	return &s
}
func optionalString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func (r *Repository) transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
