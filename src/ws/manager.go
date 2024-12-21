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
	chats      map[string][]*Client
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
		chats:   make(map[string][]*Client),
	}
	return m
}

func (m *Manager) AddClient(client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chats, err := m.store.GetChats(context.TODO(), client.username)
	m.logger.Debugw("Chats", "user", client.username, "chats", chats)
	if err != nil {
		return err
	}
	m.clients[client] = true
	for _, chat := range chats {
		m.chats[chat.Id] = append(m.chats[chat.Id], client)
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
	//remove user from chats
	for _, chat := range chats {
		for i, c := range m.chats[chat.Id] {
			if c == client {
				if i < len(m.chats[chat.Id])-1 {
					m.chats[chat.Id] = append(m.chats[chat.Id][:i], m.chats[chat.Id][i+1:]...)
				} else {
					m.chats[chat.Id] = m.chats[chat.Id][:i]
				}
			}
		}
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
