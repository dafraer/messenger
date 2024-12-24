package token

import (
	"github.com/golang-jwt/jwt/v5"
)

type MockManager struct{}

func NewMockManager() *MockManager {
	return &MockManager{}
}

func (manager *MockManager) NewToken(userId string) (string, error) {
	return "", nil
}

func (manager *MockManager) Verify(tokenString string) (*jwt.RegisteredClaims, error) {
	return &jwt.RegisteredClaims{}, nil
}
