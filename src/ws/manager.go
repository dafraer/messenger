package ws

import (
	"context"
	"errors"
	"github.com/dafraer/messenger/src/store"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var ErrEventNotSupported = errors.New("this event type is not supported")

type Manager struct {
	WSUpgrader websocket.Upgrader
	clients    ClientList
	mu         sync.RWMutex
	logger     *zap.SugaredLogger
	chats      map[string]map[*Client]struct{}
	store      *store.Storage
}

func NewManager(logger *zap.SugaredLogger, store *store.Storage) *Manager {
	m := &Manager{
		WSUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(ClientList),
		mu:      sync.RWMutex{},
		logger:  logger,
		store:   store,
	}
	return m
}

func (m *Manager) AddClient(client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chats, err := m.store.GetChats(context.TODO(), client.username)
	if err != nil {
		return err
	}
	m.clients[client] = true
	for _, chat := range chats {
		m.chats[chat.Id][client] = struct{}{}
	}
	return nil
}

func (m *Manager) RemoveClient(client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chats, err := m.store.GetChats(context.TODO(), client.username)
	if err != nil {
		return err
	}
	for _, chat := range chats {
		delete(m.chats[chat.Id], client)
	}

	//Check if client exists then delete it
	if _, ok := m.clients[client]; ok {
		//Close connection
		if err := client.connection.Close(); err != nil {
			m.logger.Errorw("Error removing websocket client", "error", err)
			return err
		}
		//Remove client
		delete(m.clients, client)
	}
	return nil
}
