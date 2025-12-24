package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
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
	logger := slog.Default()

	logger.Info("Подключение к PostgreSQL", "step", 1)
	db, err := sql.Open("pgx", con)
	if err != nil {
		logger.Error("Ошибка при открытии соединения с PostgreSQL", "error", err)
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	logger.Info("Создание конфигурации для миграций", "step", 2)
	// Создаем конфигурацию для драйвера PostgreSQL с кастомным именем таблицы миграций
	postgresConfig := &postgres.Config{
		MigrationsTable: "schema_migrations_gophermart",
	}

	driver, err := postgres.WithInstance(db, postgresConfig)
	if err != nil {
		logger.Error("Ошибка при создании конфигурации для миграций", "error", err)
		return nil, err
	}

	logger.Info("Инициализация миграций", "step", 3)
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		logger.Error("Ошибка при инициализации миграций", "error", err)
		return nil, err
	}

	logger.Info("Применение миграций", "step", 4)
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("Ошибка при применении миграций", "error", err)
		return nil, err
	}

	logger.Info("Проверка соединения с базой данных", "step", 5)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		logger.Error("Ошибка при проверке соединения с базой данных", "error", err)
		return nil, err
	}

	logger.Info("PostgreSQL успешно инициализирован")
	return &PostgresConnection{db: db}, nil
}

func (ps *PostgresConnection) Close() error {
	logger := slog.Default()
	logger.Info("Закрытие соединения с PostgreSQL")

	err := ps.db.Close()
	if err != nil {
		logger.Error("Ошибка при закрытии соединения с PostgreSQL", "error", err)
		return err
	}

	logger.Info("Соединение с PostgreSQL успешно закрыто")
	return nil
}
