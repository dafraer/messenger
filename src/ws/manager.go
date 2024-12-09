package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Manager struct {
	WSUpgrader websocket.Upgrader
	clients    ClientList
	mu         sync.RWMutex
	logger     *zap.SugaredLogger
}

func NewManager(logger *zap.SugaredLogger) *Manager {
	return &Manager{
		WSUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(ClientList),
		mu:      sync.RWMutex{},
		logger:  logger,
	}
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
		client.connection.Close()
		//Remove client
		delete(m.clients, client)
	}
}
