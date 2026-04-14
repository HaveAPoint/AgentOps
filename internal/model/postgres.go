package model

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"agentops/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresDB(c config.PostgresConf) (*sql.DB, error) {
	dsn := buildPostgresDSN(c)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetimeSeconds > 0 {
		db.SetConnMaxLifetime(time.Duration(c.ConnMaxLifetimeSeconds) * time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func buildPostgresDSN(c config.PostgresConf) string {
	user := url.User(c.User)
	if c.Password != "" {
		user = url.UserPassword(c.User, c.Password)
	}

	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   user,
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.DBName,
	}

	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	return u.String()
}
