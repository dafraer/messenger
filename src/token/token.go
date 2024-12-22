package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenLiveSpan = time.Hour * 24 * 30
)

type JWTManager struct {
	signingKey string
}

// New creates new JWT token manager using provided signing key
func New(signingKey string) *JWTManager {
	return &JWTManager{
		signingKey: signingKey,
	}
}

// NewToken generates a JWT token using SHA512 algorithm
func (manager *JWTManager) NewToken(userId string) (string, error) {
	//Define the payload
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLiveSpan)),
		Subject:   userId,
	}

	//Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	accessToken, err := token.SignedString([]byte(manager.signingKey))
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

// Verify verifies JWT token and returns token's payload
func (manager *JWTManager) Verify(tokenString string) (*jwt.RegisteredClaims, error) {
	//Parse token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		//Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(manager.signingKey), nil
	})

	if err != nil {
		return nil, err
	}

	//Get token claims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
