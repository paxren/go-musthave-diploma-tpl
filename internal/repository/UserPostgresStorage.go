package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/paxren/go-musthave-diploma-tpl/internal/models"
)

// ПОТОКО НЕБЕЗОПАСНО!

type UserPostgresStorage struct {
	db *PostgresConnection
}

func MakeUserPostgresStorage(pc *PostgresConnection) *UserPostgresStorage {
	return &UserPostgresStorage{
		db: pc,
	}
}

// hashPassword хеширует пароль с использованием bcrypt
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// checkPassword проверяет соответствие пароля его хешу
func checkPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (ps *UserPostgresStorage) GetUser(login string) *models.User {
	logger := slog.Default()
	logger.Debug("Получение пользователя", "login", login)

	var user models.User
	var passwordHash string

	query := "SELECT id, login, password_hash FROM gophermart_users WHERE login = $1"
	err := ps.db.db.QueryRow(query, login).Scan(&user.UserID, &user.Login, &passwordHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Debug("Пользователь не найден", "login", login)
			return nil
		}
		logger.Error("Ошибка при получении пользователя", "login", login, "error", err)
		return nil
	}

	// В поле Password не храним хеш, так как оно используется только для аутентификации
	user.Password = ""

	logger.Debug("Пользователь успешно получен", "login", login, "user_id", user.UserID)
	return &user
}

func (ps *UserPostgresStorage) RegisterUser(user models.User) error {
	logger := slog.Default()
	logger.Info("Попытка регистрации пользователя", "login", user.Login)

	// Проверяем, существует ли пользователь с таким логином
	existingUser := ps.GetUser(user.Login)
	if existingUser != nil {
		logger.Warn("Пользователь уже существует", "login", user.Login)
		return ErrUserExist
	}

	// Хешируем пароль
	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		logger.Error("Ошибка хеширования пароля", "login", user.Login, "error", err)
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	// Вставляем нового пользователя в базу данных
	var userID uint64
	query := "INSERT INTO gophermart_users (login, password_hash) VALUES ($1, $2) RETURNING id"
	err = ps.db.db.QueryRow(query, user.Login, hashedPassword).Scan(&userID)

	if err != nil {
		logger.Error("Ошибка при вставке пользователя", "login", user.Login, "error", err)
		// Проверяем ошибку уникальности ограничения (если пользователь уже существует)
		if isUniqueViolationError(err) {
			logger.Warn("Нарушение уникальности ограничения для пользователя", "login", user.Login)
			return ErrUserExist
		}
		return fmt.Errorf("ошибка при регистрации пользователя: %w", err)
	}

	logger.Info("Пользователь успешно зарегистрирован", "login", user.Login, "user_id", userID)
	// Устанавливаем ID пользователя
	//user.UserID = &userID

	return nil
}

func (ps *UserPostgresStorage) LoginUser(user models.User) error {
	logger := slog.Default()
	logger.Debug("Попытка входа пользователя", "login", user.Login)

	// Получаем пользователя из базы данных
	dbUser := ps.GetUser(user.Login)
	if dbUser == nil {
		logger.Warn("Пользователь не найден", "login", user.Login)
		return ErrBadLogin
	}

	// Получаем хеш пароля из базы данных
	var passwordHash string
	query := "SELECT password_hash FROM gophermart_users WHERE login = $1"
	err := ps.db.db.QueryRow(query, user.Login).Scan(&passwordHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("Пользователь не найден в базе данных", "login", user.Login)
			return ErrBadLogin
		}
		logger.Error("Ошибка при аутентификации пользователя", "login", user.Login, "error", err)
		return fmt.Errorf("ошибка при аутентификации пользователя: %w", err)
	}

	// Проверяем пароль
	err = checkPassword(user.Password, passwordHash)
	if err != nil {
		// Если пароль не совпадает, возвращаем ошибку аутентификации
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			logger.Warn("Неверный пароль", "login", user.Login)
			return ErrBadLogin
		}
		logger.Error("Ошибка при проверке пароля", "login", user.Login, "error", err)
		return fmt.Errorf("ошибка при проверке пароля: %w", err)
	}

	logger.Info("Пользователь успешно авторизован", "login", user.Login)
	return nil
}

// isUniqueViolationError проверяет, является ли ошибка ошибкой нарушения уникального ограничения
func isUniqueViolationError(err error) bool {
	// PostgreSQL код ошибки для нарушения уникального ограничения
	// Это может потребовать дополнительной настройки в зависимости от драйвера
	if err != nil && err.Error() == "pq: duplicate key value violates unique constraint" {
		return true
	}
	return false
}
