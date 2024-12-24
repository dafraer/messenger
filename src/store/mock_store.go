package store

import (
	"context"
	"golang.org/x/crypto/bcrypt"
)

type MockStore struct{}

func NewMockStore() *MockStore {
	return &MockStore{}
}

func (s *MockStore) NewUser(ctx context.Context, username, password string) error {
	return nil
}

func (s *MockStore) GetUser(ctx context.Context, username string) (*User, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte("passwordTest"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{Username: username, Password: string(hashPassword)}, nil
}

func (s *MockStore) NewChat(ctx context.Context, members []string, owner string) (interface{}, error) {
	return "1", nil
}

func (s *MockStore) GetChat(ctx context.Context, chatId string) (*Chat, error) {
	return &Chat{Id: chatId, Members: []string{"usernameTest"}}, nil
}

func (s *MockStore) GetChats(ctx context.Context, username string) ([]Chat, error) {
	return []Chat{{Owner: username}}, nil
}

func (s *MockStore) GetMessages(ctx context.Context, chatId string) ([]Message, error) {
	return []Message{{ChatId: chatId, Text: "hello world"}}, nil
}

func (s *MockStore) SaveMessage(ctx context.Context, msg Message) error {
	return nil
}

func (s *MockStore) RemoveUserFromChat(ctx context.Context, username, chatId string) error {
	return nil
}
