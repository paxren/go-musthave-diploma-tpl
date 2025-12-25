package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет структуру утверждений в JWT токене
type Claims struct {
	UserID uint64 `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// JWTService представляет сервис для работы с JWT токенами
type JWTService struct {
	secretKey string
}

// NewJWTService создает новый экземпляр JWTService
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey: secretKey,
	}
}

// GenerateToken генерирует JWT токен для пользователя
func (s *JWTService) GenerateToken(userID uint64, login string) (string, error) {
	// Создаем утверждения с данными пользователя и временем истечения токена
	claims := &Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Токен действителен 24 часа
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Создаем токен с методом подписи HS256 и утверждениями
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен секретным ключом
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken валидирует JWT токен и возвращает утверждения
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// Парсим токен с проверкой подписи
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что метод подписи - HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("неверный метод подписи")
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Проверяем, что токен валиден
	if !token.Valid {
		return nil, errors.New("невалидный токен")
	}

	// Извлекаем утверждения из токена
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("невозможно извлечь утверждения из токена")
	}

	return claims, nil
}
