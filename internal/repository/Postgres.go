package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type PostgresConnection struct {
	db *sql.DB
}

func MakePostgresStorage(con string) (*PostgresConnection, error) {

	fmt.Println("1")
	db, err := sql.Open("pgx", con)
	if err != nil {
		fmt.Printf("err=%v", err)
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	fmt.Println("2")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Printf("driver err! err=%v", err)
		return nil, err
	}

	fmt.Println("3")
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		fmt.Printf("migration err! err=%v", err)
		return nil, err
	}
	fmt.Println("4")
	m.Up()

	fmt.Println("5")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		fmt.Printf("err=%v", err)
		return nil, err
	}

	return &PostgresConnection{db: db}, nil
}

func (ps *PostgresConnection) Close() error {

	ps.db.Close()
	return nil
}
