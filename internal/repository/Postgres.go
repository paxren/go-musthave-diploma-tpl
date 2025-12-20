package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	fmt.Println("3")

	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Ошибка при получении рабочей директории: %v\n", err)
	} else {
		fmt.Printf("Текущая рабочая директория: %s\n", wd)
	}

	// Проверяем существование директории миграций
	migrationsPath := filepath.Join(wd, "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		fmt.Printf("Директория миграций не существует: %s\n", migrationsPath)
	} else {
		fmt.Printf("Директория миграций найдена: %s\n", migrationsPath)
	}

	// Создаем конфигурацию для драйвера PostgreSQL с кастомным именем таблицы миграций
	postgresConfig := &postgres.Config{
		MigrationsTable: "schema_migrations_gophermart",
	}

	driver, err := postgres.WithInstance(db, postgresConfig)
	if err != nil {
		fmt.Printf("driver err! err=%v", err)
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		fmt.Printf("migration err! err=%v", err)
		return nil, err
	}
	fmt.Println("4")

	// Проверяем текущую версию миграций
	version, dirty, err := m.Version()
	if err != nil {
		fmt.Printf("Ошибка при получении версии миграций: %v\n", err)
	} else {
		fmt.Printf("Текущая версия миграций: %d, грязное состояние: %v\n", version, dirty)
	}

	// Добавляем вывод о результате миграции
	err = m.Up()
	if err != nil {
		errStr := err.Error()
		fmt.Printf("Полная ошибка миграции: %s\n", errStr)
		if errStr == "no change" {
			fmt.Println("Миграции уже применены")
		} else if strings.Contains(errStr, "Dirty database version") {
			fmt.Println("Обнаружено грязное состояние базы данных, пытаемся исправить...")
			// Принудительно устанавливаем версию на 1, чтобы затем применить миграции 2 и 3
			err = m.Force(1)
			if err != nil {
				fmt.Printf("Ошибка при установке версии 1: %v\n", err)
				return nil, err
			}
			fmt.Println("Версия установлена на 1, применяем миграции заново...")
			err = m.Up()
			if err != nil {
				fmt.Printf("Ошибка при повторном применении миграций: %v\n", err)
				return nil, err
			}
			fmt.Println("Миграции успешно применены после исправления")
		} else {
			fmt.Printf("Ошибка при применении миграций: %v\n", err)
			return nil, err
		}
	} else {
		fmt.Println("Миграции успешно применены")
	}

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
