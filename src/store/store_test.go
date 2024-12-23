package store

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
)

func TestNew(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, storage.db.Disconnect(context.Background()))
}

func TestNewUser(t *testing.T) {
	//Create ne mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)

	//Create new user
	assert.NoError(t, clearStorage(storage.db))
	assert.NoError(t, storage.NewUser(context.Background(), "testUsername", "testPassword"))
	assert.NoError(t, clearStorage(storage.db))
}

func TestGetUser(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new user
	assert.NoError(t, storage.NewUser(context.Background(), "testUsername", "testPassword"))

	//Get the user from the database
	user, err := storage.GetUser(context.Background(), "testUsername")
	assert.NoError(t, err)

	//Check that username is the same that we saved
	assert.Equal(t, "testUsername", user.Username)
	assert.NoError(t, clearStorage(storage.db))
}

func TestNewChat(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)
	assert.NoError(t, clearStorage(storage.db))
}

func TestGetChat(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId type into string
	chatIdString := chatId.(primitive.ObjectID).Hex()

	//Get chat from database
	chat, err := storage.GetChat(context.Background(), chatIdString)
	assert.NoError(t, err)

	//Check that we got same shat that we saved
	assert.Equal(t, chatIdString, chat.Id)
	assert.Equal(t, []string{"user1", "user2"}, chat.Members)
	assert.Equal(t, "user1", chat.Owner)

	assert.NoError(t, clearStorage(storage.db))
}

func TestGetChats(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId type into string
	chatIdString1 := chatId.(primitive.ObjectID).Hex()

	//Create another chat
	chatId, err = storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId type into string
	chatIdString2 := chatId.(primitive.ObjectID).Hex()

	//Get chats we created from the database
	chats, err := storage.GetChats(context.Background(), "user2")
	assert.NoError(t, err)

	//Check that received chats are the same chats we saved
	assert.Equal(t, len(chats), 2)
	assert.Equal(t, chatIdString1, chats[0].Id)
	assert.Equal(t, chatIdString2, chats[1].Id)
	assert.NoError(t, clearStorage(storage.db))
}

func TestSaveMessage(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId to string
	chatIdString := chatId.(primitive.ObjectID).Hex()

	//Save message
	assert.NoError(t, storage.SaveMessage(context.Background(), Message{
		ChatId: chatIdString,
		From:   "user1",
		Text:   "Hello World",
		Time:   1,
	}))
	assert.NoError(t, clearStorage(storage.db))
}

func TestGetMessages(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId to string
	chatIdString := chatId.(primitive.ObjectID).Hex()

	//Save message
	assert.NoError(t, storage.SaveMessage(context.Background(), Message{
		ChatId: chatIdString,
		From:   "user1",
		Text:   "Hello World",
		Time:   1,
	}))

	//Get message
	messages, err := storage.GetMessages(context.Background(), chatIdString)
	assert.NoError(t, err)

	//Check that received messages is the same as the saved message
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, "user1", messages[0].From)
	assert.Equal(t, "Hello World", messages[0].Text)
	assert.Equal(t, int64(1), messages[0].Time)

	assert.NoError(t, clearStorage(storage.db))
}

func TestRemoveUserFromChat(t *testing.T) {
	//Create new mongo client
	client, err := createDBConnection()
	assert.NoError(t, err)

	//Create new storage
	storage := New(client)
	assert.NoError(t, clearStorage(storage.db))

	//Create new chat
	chatId, err := storage.NewChat(context.Background(), []string{"user1", "user2"}, "user1")
	assert.NoError(t, err)

	//Check that chatId is not empty
	assert.NotEmpty(t, chatId)

	//Parse objectId to string
	chatIdString := chatId.(primitive.ObjectID).Hex()

	//Remove user from chat
	assert.NoError(t, storage.RemoveUserFromChat(context.Background(), "user1", chatIdString))

	//Get chat
	chat, err := storage.GetChat(context.Background(), chatIdString)
	assert.NoError(t, err)

	//Check that user has been removed
	assert.Equal(t, chatIdString, chat.Id)
	assert.Equal(t, []string{"user2"}, chat.Members)
	assert.Equal(t, "user1", chat.Owner)
	assert.NoError(t, clearStorage(storage.db))
}

func createDBConnection() (*mongo.Client, error) {
	//Create storage
	return mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
}

func clearStorage(client *mongo.Client) error {
	//Clear messages collection
	coll := client.Database("messenger").Collection("messages")
	if _, err := coll.DeleteMany(context.Background(), bson.D{}); err != nil {
		return err
	}

	//Clear chats collection
	coll = client.Database("messenger").Collection("chats")
	if _, err := coll.DeleteMany(context.Background(), bson.D{}); err != nil {
		return err
	}

	//Clear users collection
	coll = client.Database("messenger").Collection("users")
	if _, err := coll.DeleteMany(context.Background(), bson.D{}); err != nil {
		return err
	}
	return nil
}
