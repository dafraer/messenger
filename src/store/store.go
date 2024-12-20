package store

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrUserExists = fmt.Errorf("user exists")

type Storage struct {
	db *mongo.Client
}

type User struct {
	Id       string `bson:"_id,omitempty" json:"id,omitempty"`
	Username string `bson:"username"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
}

type Chat struct {
	Id      *string  `bson:"_id, omitempty" json:"id,omitempty"`
	Members []string `bson:"members" json:"members"`
	Owner   string   `bson:"owner" json:"owner"`
}

type Message struct {
	Id     string `bson:"_id,omitempty"`
	ChatId string `bson:"chat_id"`
	From   string `bson:"from"`
	Text   string `bson:"text"`
	//unix utc time
	Time int64 `bson:"time"`
	Read bool  `bson:"read"`
}

func New(client *mongo.Client) *Storage {
	return &Storage{
		db: client,
	}
}

func (s *Storage) Init(ctx context.Context) error {
	coll := s.db.Database("messenger").Collection("users")
	model := mongo.IndexModel{Keys: bson.D{{"username", "text"}}}
	_, err := coll.Indexes().CreateOne(ctx, model)
	if err != nil {
		return err
	}
	return nil
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

func (s *Storage) NewChat(ctx context.Context, chat *Chat) (interface{}, error) {
	coll := s.db.Database("messenger").Collection("chats")
	res, err := coll.InsertOne(ctx, bson.D{{Key: "members", Value: chat.Members}, {Key: "owner", Value: chat.Owner}})
	if res != nil {
		return res.InsertedID, err
	}
	return nil, err
}

func (s *Storage) DeleteChat(ctx context.Context, chatId string) error {
	coll := s.db.Database("messenger").Collection("chats")
	_, err := coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: chatId}})
	return err
}

func (s *Storage) GetChat(ctx context.Context, chatId string) (*Chat, error) {
	coll := s.db.Database("messenger").Collection("chats")
	var chat Chat
	objID, err := primitive.ObjectIDFromHex(chatId)
	if err != nil {
		return nil, err
	}
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: objID}}).Decode(&chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetChats returns chats of the user
func (s *Storage) GetChats(ctx context.Context, username string) ([]Chat, error) {
	coll := s.db.Database("messenger").Collection("chats")
	var chats []Chat
	cursor, err := coll.Find(ctx, bson.D{{"owner", username}})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &chats); err != nil {
		return nil, err
	}
	return chats, nil
}

// GetMessages returns list of messages in a specific chat
func (s *Storage) GetMessages(ctx context.Context, chatId string) ([]Message, error) {
	coll := s.db.Database("messenger").Collection("messages")
	var messages []Message
	cursor, err := coll.Find(ctx, bson.D{{"chat_id", chatId}})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// FindSimilarUsers returns a list of usernames that are similar to the given username
// TODO fix
func (s *Storage) FindSimilarUsers(ctx context.Context, username string) ([]User, error) {
	coll := s.db.Database("messenger").Collection("users")
	filter := bson.D{{"$text", bson.D{{"$search", username}}}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var users []User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
