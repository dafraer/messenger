package ws

import (
	"errors"
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
}

func NewManager(logger *zap.SugaredLogger) *Manager {
	m := &Manager{
		WSUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(ClientList),
		mu:      sync.RWMutex{},
		logger:  logger,
	}
	return m
}

func (m *Manager) AddClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client] = true
}

func (m *Manager) RemoveClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//Check if client exists then delete it
	if _, ok := m.clients[client]; ok {
		//Close connection
		if err := client.connection.Close(); err != nil {
			m.logger.Errorw("Error removing websocket client", "error", err)
			return
		}
		//Remove client
		delete(m.clients, client)
	}
}
