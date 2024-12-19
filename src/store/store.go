package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrUserExists = fmt.Errorf("user exists")

type Storage struct {
	db *mongo.Client
}

type User struct {
	Id       string `bson:"_id,omitempty" json:"id"`
	Username string `bson:"username"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
	Chats    []Chat `bson:"chats,omitempty" json:"chats,omitempty"`
}

type Chat struct {
	Id      string   `bson:"_id, omitempty" json:"id"`
	Members []string `bson:"members" json:"members"`
	Owner   string   `bson:"owner" json:"owner"`
}

type Message struct {
	Id     string `bson:"_id,omitempty"`
	ChatId int    `bson:"chat_id"`
	From   string `bson:"from"`
	Text   string `bson:"text"`
	//unix utc time
	Time int64 `bson:"time"`
}

func New(client *mongo.Client) *Storage {
	return &Storage{
		db: client,
	}
}

func (s *Storage) NewUser(ctx context.Context, username, password string) error {
	coll := s.db.Database("messenger").Collection("users")
	//Check if user exists
	var u User
	err := coll.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&u)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return ErrUserExists
	}
	_, err = coll.InsertOne(ctx, bson.D{{Key: "username", Value: username}, {Key: "password", Value: password}})
	return err
}

func (s *Storage) GetUser(ctx context.Context, username string) (*User, error) {
	coll := s.db.Database("messenger").Collection("users")
	var user User
	if err := coll.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) updateUser(ctx context.Context, user *User) error {
	coll := s.db.Database("messenger").Collection("users")
	//Check if user exists
	var u User
	err := coll.FindOne(ctx, bson.D{{Key: "username", Value: user.Username}}).Decode(&u)
	if err != nil {
		return err
	}
	_, err = coll.UpdateOne(ctx, bson.D{{"username", user.Username}}, bson.D{{Key: "$set", Value: user}})
	return err
}

func (s *Storage) NewChat(ctx context.Context, chat *Chat) error {
	coll := s.db.Database("messenger").Collection("chats")
	_, err := coll.InsertOne(ctx, chat)
	return err
}

func (s *Storage) DeleteChat(ctx context.Context, chatId string) error {
	coll := s.db.Database("messenger").Collection("chats")
	_, err := coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: chatId}})
	return err
}

func (s *Storage) GetChat(ctx context.Context, chatId string) (*Chat, error) {
	coll := s.db.Database("messenger").Collection("chats")
	var chat Chat
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: chatId}}).Decode(&chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetChats returns chats of the user
func (s *Storage) GetChats(ctx context.Context, username string) ([]Chat, error) {
	return []Chat{}, nil
}

// GetMessages returns list of messages in a specific chat
func (s *Storage) GetMessages(ctx context.Context, chatId string) ([]Message, error) {
	return []Message{}, nil
}

// FindSimilarUsers returns a list of usernames that are similar to the given username
func (s *Storage) FindSimilarUsers(ctx context.Context, username string) ([]User, error) {
	return []User{}, nil
}
