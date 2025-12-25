package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateToken(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret)

	userID := uint64(123)
	login := "testuser"

	token, err := jwtService.GenerateToken(userID, login)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Проверяем, что токен можно распарсить
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(*Claims)
	require.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, login, claims.Login)
}

func TestJWTService_ValidateToken(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret)

	userID := uint64(123)
	login := "testuser"

	// Генерируем токен
	token, err := jwtService.GenerateToken(userID, login)
	require.NoError(t, err)

	// Валидируем токен
	claims, err := jwtService.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, login, claims.Login)
}

func TestJWTService_ValidateToken_Invalid(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret)

	// Пробуем валидировать невалидный токен
	invalidToken := "invalid.token.string"

	claims, err := jwtService.ValidateToken(invalidToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateToken_WrongSecret(t *testing.T) {
	secret1 := "test-secret-key-1"
	secret2 := "test-secret-key-2"

	jwtService1 := NewJWTService(secret1)
	jwtService2 := NewJWTService(secret2)

	userID := uint64(123)
	login := "testuser"

	// Генерируем токен с одним секретом
	token, err := jwtService1.GenerateToken(userID, login)
	require.NoError(t, err)

	// Пытаемся валидировать с другим секретом
	claims, err := jwtService2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	// Создаем JWT сервис с очень коротким сроком действия токена для теста
	secret := "test-secret-key"
	jwtService := &JWTService{secretKey: secret}

	userID := uint64(123)
	login := "testuser"

	// Создаем токен с истекшим сроком действия
	claims := &Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Истек 1 час назад
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)), // Выпущен 2 часа назад
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	// Пытаемся валидировать истекший токен
	validatedClaims, err := jwtService.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}

func TestJWTService_ValidateToken_WrongSigningMethod(t *testing.T) {
	secret := "test-secret-key"

	userID := uint64(123)
	login := "testuser"

	// Создаем токен с правильным методом подписи
	claims := &Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	// Теперь создадим новый JWT сервис с другим секретом и попробуем валидировать
	wrongSecretService := NewJWTService("wrong-secret")
	validatedClaims, err := wrongSecretService.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}
