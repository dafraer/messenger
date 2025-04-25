package store

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrUserExists = fmt.Errorf("user exists")

type Storer interface {
	NewUser(ctx context.Context, username string, password string) error
	GetUser(ctx context.Context, username string) (*User, error)
	NewChat(ctx context.Context, members []string, owner string) (interface{}, error)
	GetChat(ctx context.Context, chatId string) (*Chat, error)
	GetChats(ctx context.Context, username string) ([]Chat, error)
	GetMessages(ctx context.Context, chatId string) ([]Message, error)
	SaveMessage(ctx context.Context, msg Message) error
	RemoveUserFromChat(ctx context.Context, username string, chatId string) error
}

type Storage struct {
	db *mongo.Client
}

type User struct {
	Id       string `bson:"_id,omitempty" json:"id,omitempty"`
	Username string `bson:"username" json:"username"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
}

type Chat struct {
	Id      string   `bson:"_id, omitempty" json:"id,omitempty"`
	Members []string `bson:"members" json:"members"`
	Owner   string   `bson:"owner" json:"owner"`
}

type Message struct {
	ChatId string `bson:"chat_id" json:"chat_id"`
	From   string `bson:"from"    json:"from"`
	Text   string `bson:"text"    json:"text"`
	//Unix utc time
	Time int64 `bson:"time"    json:"time"`
}

// New creates new storage instance with mongo client as the only field
func New(client *mongo.Client) *Storage {
	return &Storage{
		db: client,
	}
}

// NewUser adds new user to the database using username and password
func (s *Storage) NewUser(ctx context.Context, username, password string) error {
	//Get users collection
	coll := s.db.Database("messenger").Collection("users")

	//Check if user exists
	var u User
	err := coll.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&u)

	//Return error if user with the same username already exists
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return ErrUserExists
	}

	//Create new user in the database
	_, err = coll.InsertOne(ctx, bson.D{{Key: "username", Value: username}, {Key: "password", Value: password}})
	return err
}

// GetUser returns all user info
func (s *Storage) GetUser(ctx context.Context, username string) (*User, error) {
	//Get users collection
	coll := s.db.Database("messenger").Collection("users")

	//Get user from the database by username
	var user User
	if err := coll.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// NewChat creates new chat using members and owner fields. Returns chat id
func (s *Storage) NewChat(ctx context.Context, members []string, owner string) (interface{}, error) {
	//Get chats collection
	coll := s.db.Database("messenger").Collection("chats")

	//Create new chat in the database
	res, err := coll.InsertOne(ctx, bson.D{{Key: "members", Value: members}, {Key: "owner", Value: owner}})
	if err != nil {
		return nil, err
	}
	return res.InsertedID, err
}

// GetChat returns chat info by chat id
func (s *Storage) GetChat(ctx context.Context, chatId string) (*Chat, error) {
	//Get chats collection
	coll := s.db.Database("messenger").Collection("chats")

	//Convert chatId to objectId type
	objId, err := primitive.ObjectIDFromHex(chatId)
	if err != nil {
		return nil, err
	}

	//Get chat info from the database
	var chat Chat
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: objId}}).Decode(&chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetChats returns all chats where user is a member
func (s *Storage) GetChats(ctx context.Context, username string) ([]Chat, error) {
	//Get chats collection
	coll := s.db.Database("messenger").Collection("chats")

	//Find chats where user is a member
	var chats []Chat
	cursor, err := coll.Find(ctx, bson.D{{"members", bson.M{"$in": []string{username}}}})
	if err != nil {
		return nil, err
	}

	//Parse chats into chats struct
	if err = cursor.All(ctx, &chats); err != nil {
		return nil, err
	}

	return chats, nil
}

// GetMessages returns list of messages in a chat by chat id
func (s *Storage) GetMessages(ctx context.Context, chatId string) ([]Message, error) {
	//Get messages collection
	coll := s.db.Database("messenger").Collection("messages")

	//Find messages from a specific chat
	var messages []Message
	cursor, err := coll.Find(ctx, bson.D{{"chat_id", chatId}})
	if err != nil {
		return nil, err
	}

	//Parse messages into messages struct
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SaveMessage saves message to the database
func (s *Storage) SaveMessage(ctx context.Context, msg Message) error {
	//Get messages collection
	coll := s.db.Database("messenger").Collection("messages")

	//Create new message in the database
	_, err := coll.InsertOne(ctx, msg)
	return err
}

// RemoveUserFromChat removes user from a specific by deleting username from the members array
func (s *Storage) RemoveUserFromChat(ctx context.Context, username, chatId string) error {
	//Get chats collection
	coll := s.db.Database("messenger").Collection("chats")

	//Convert chatId to object id type
	objId, err := primitive.ObjectIDFromHex(chatId)
	if err != nil {
		return err
	}

	//Delete user from members array
	_, err = coll.UpdateOne(ctx, bson.D{{"_id", objId}}, bson.D{{"$pull", bson.D{{"members", username}}}})
	return err
}
