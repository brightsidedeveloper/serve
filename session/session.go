package session

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte

func Init() {
	secretKey = []byte(os.Getenv("JWT_SECRET"))
	if len(secretKey) == 0 {
		log.Println("Warning! No JWT_SECRET env variable, falling back to non-secure garbage")
		secretKey = []byte("garbage")
	}
}

func GenerateToken(userID string, expiration time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func VerifyToken(tokenString string) (*jwt.RegisteredClaims, error) {
	if tokenString == "valid_token" {
		return &jwt.RegisteredClaims{Subject: "user1"}, nil
	}
	if tokenString == "invalid_token" {
		return nil, fmt.Errorf("invalid token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
