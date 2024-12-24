package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dafraer/messenger/src/store"
	"github.com/dafraer/messenger/src/token"
	"github.com/dafraer/messenger/src/ws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testUser = store.User{
	Id:       "1",
	Username: "usernameTest",
	Password: "passwordTest",
}

func TestServeWS(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Create test server
	srv := httptest.NewServer(http.HandlerFunc(s.serveWS))

	//Convert http:// to ws://
	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Connect to the server
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	assert.NoError(t, wsConn.Close())
}

func TestRegister(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Create test server
	srv := httptest.NewServer(http.HandlerFunc(s.handleRegister))

	//Marshal request body
	body, err := json.Marshal(authRequest{Username: testUser.Username, Password: testUser.Password})
	assert.NoError(t, err)

	//Make test request
	resp, err := http.Post(srv.URL, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer assert.NoError(t, resp.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("expected 200 but got %d", resp.StatusCode))
}

func TestLogin(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Create test server
	srv := httptest.NewServer(http.HandlerFunc(s.handleLogin))

	//Marshal request body
	body, err := json.Marshal(authRequest{Username: testUser.Username, Password: testUser.Password})
	assert.NoError(t, err)

	//Make test request
	resp, err := http.Post(srv.URL, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer assert.NoError(t, resp.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("expected 200 but got %d", resp.StatusCode))
}

func TestHandleUser(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Make test request
	r := httptest.NewRequest(http.MethodGet, "/user", nil)
	r.SetPathValue("username", testUser.Username)
	w := httptest.NewRecorder()
	s.handleUser(w, r)
	res := w.Result()
	defer assert.NoError(t, res.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected 200 but got %d", res.StatusCode))
	assert.NoError(t, err)

	//Decode json response
	var user store.User
	assert.NotNil(t, res.Body)
	assert.NoError(t, json.NewDecoder(res.Body).Decode(&user))

	//Check that we got a correct response
	assert.Equal(t, testUser.Username, user.Username)
}

func TestHandleChats(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Make test request
	r := httptest.NewRequest(http.MethodGet, "/chats", nil)
	r.SetPathValue("username", testUser.Username)

	//Put username in context so user is authorized
	r = r.WithContext(context.WithValue(context.Background(), "username", testUser.Username))
	w := httptest.NewRecorder()
	s.handleChats(w, r)
	res := w.Result()
	defer assert.NoError(t, res.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected 200 but got %d", res.StatusCode))

	//Decode json response
	var chats []store.Chat
	assert.NotNil(t, res.Body)
	assert.NoError(t, json.NewDecoder(res.Body).Decode(&chats))

	//Check that we got a correct response
	assert.Equal(t, testUser.Username, chats[0].Owner)
}

func TestHandleMessages(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Make test request
	r := httptest.NewRequest(http.MethodGet, "/messages", nil)
	r.SetPathValue("chatId", "1")

	//Put username in context so user is authorized
	r = r.WithContext(context.WithValue(context.Background(), "username", testUser.Username))
	w := httptest.NewRecorder()
	s.handleMessages(w, r)
	res := w.Result()
	defer assert.NoError(t, res.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected 200 but got %d", res.StatusCode))

	//Decode json response
	var msgs []store.Message
	assert.NotNil(t, res.Body)
	assert.NoError(t, json.NewDecoder(res.Body).Decode(&msgs))

	//Check that we got a correct response
	assert.Equal(t, "1", msgs[0].ChatId)
	assert.Equal(t, "hello world", msgs[0].Text)
}

func TestHandleNewChat(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Create request body
	body, err := json.Marshal(store.Chat{Owner: testUser.Username, Members: []string{testUser.Username}})
	assert.NoError(t, err)

	//Make test request
	r := httptest.NewRequest(http.MethodGet, "/newChat", bytes.NewBuffer(body))

	//Put username in context so user is authorized
	r = r.WithContext(context.WithValue(context.Background(), "username", testUser.Username))
	w := httptest.NewRecorder()
	s.handleNewChat(w, r)
	res := w.Result()
	defer assert.NoError(t, res.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected 200 but got %d", res.StatusCode))

	//Decode json response
	var chatId string
	assert.NotNil(t, res.Body)
	assert.NoError(t, json.NewDecoder(res.Body).Decode(&chatId))

	//Check that we got a correct response
	assert.Equal(t, "1", chatId)
}

func TestHandleRemove(t *testing.T) {
	//Create server
	s, err := createTestService()
	assert.NoError(t, err)

	//Make test request
	r := httptest.NewRequest(http.MethodGet, "/remove", nil)
	r.SetPathValue("chatId", "1")
	r.SetPathValue("username", testUser.Username)

	//Put username in context so user is authorized
	r = r.WithContext(context.WithValue(context.Background(), "username", testUser.Username))
	w := httptest.NewRecorder()
	s.handleMessages(w, r)
	res := w.Result()
	defer assert.NoError(t, res.Body.Close())

	//Check that status code is OK
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected 200 but got %d", res.StatusCode))
	assert.NoError(t, err)
}

func createTestService() (*Server, error) {
	//Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	var sugar *zap.SugaredLogger

	//Create sugared logger
	if logger != nil {
		sugar = logger.Sugar()
	}

	//Create WSManager
	WSManager := ws.NewManager(sugar, store.NewMockStore())

	//Create a new service for testing
	return New(WSManager, sugar, token.NewMockManager(), store.NewMockStore()), nil
}
