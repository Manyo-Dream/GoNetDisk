package util

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey            string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

func NewJWTManager(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey, accessTokenDuration, refreshTokenDuration}
}

func (jm *JWTManager) GenerateAccessToken(userID, username, email string) (string, error) {
	return jm.generateToken(userID, username, email, jm.accessTokenDuration)
}

func (jm *JWTManager) GenerateRefreshToken(userID, username, email string) (string, error) {
	return jm.generateToken(userID, username, email, jm.refreshTokenDuration)
}

func (jm *JWTManager) generateToken(userID, username, email string, duration time.Duration) (string, error) {
	claims := Claims{
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jm.secretKey))
}

func (jm *JWTManager) GetAccessExpiresSeconds() int {
	return int(jm.accessTokenDuration.Seconds())
}

func (jm *JWTManager) GetRefreshTokenDuration() time.Duration {
	return jm.refreshTokenDuration
}

func (jm *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("无效的签名算法")
			}
			return []byte(jm.secretKey), nil
		})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的token")
	}

	return claims, nil
}
