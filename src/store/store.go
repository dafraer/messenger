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
	Username string `bson:"username"`
	Password string `bson:"password"`
}

type Chat struct {
	Id int
}

func New(client *mongo.Client) *Storage {
	return &Storage{
		db: client,
	}
}

func (s *Storage) NewUser(ctx context.Context, user *User) error {
	coll := s.db.Database("messenger").Collection("users")
	//Check if user exists
	var u User
	err := coll.FindOne(ctx, bson.D{{Key: "username", Value: user.Username}}).Decode(&u)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return ErrUserExists
	}
	_, err = coll.InsertOne(ctx, bson.D{{Key: "username", Value: user.Username}, {Key: "password", Value: user.Password}})
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
