package ws

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var ErrEventNotSupported = errors.New("this event type is not supported")

type Manager struct {
	WSUpgrader    websocket.Upgrader
	clients       ClientList
	mu            sync.RWMutex
	logger        *zap.SugaredLogger
	eventHandlers map[string]EventHandler
}

func NewManager(logger *zap.SugaredLogger) *Manager {
	m := &Manager{
		WSUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:       make(ClientList),
		mu:            sync.RWMutex{},
		logger:        logger,
		eventHandlers: make(map[string]EventHandler),
	}
	m.setUpEventHandlers()
	return m
}

// for now doesnt makes sense will fix later
func (m *Manager) setUpEventHandlers() {
	m.eventHandlers[EventSendMessage] = func(e Event, c *Client) error {
		fmt.Println("skibidi op op")
		return nil
	}
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	//If event is not present in the map throw an error
	if handler, ok := m.eventHandlers[event.Type]; ok {
		//Execute the handler
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	}
	return ErrEventNotSupported
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
