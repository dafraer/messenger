package ws

import (
	"context"
	"github.com/dafraer/messenger/src/store"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ClientList is a map holding list of clients
type ClientList map[*Client]bool

type Manager struct {
	WSUpgrader websocket.Upgrader
	clients    ClientList
	mu         sync.RWMutex
	logger     *zap.SugaredLogger
	//chats field stores map of chats where client slice is stored as a value
	chats map[string][]*Client
	store store.Storer
}

// NewManager creates new websocket manager
func NewManager(logger *zap.SugaredLogger, store store.Storer) *Manager {
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

// AddClient adds new client to the websocket manager
func (m *Manager) AddClient(ctx context.Context, client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	//Get user chats
	chats, err := m.store.GetChats(ctx, client.username)
	if err != nil {
		return err
	}

	//Add clients to the client list
	m.clients[client] = true

	//add all user chats to the manager
	for _, chat := range chats {
		m.chats[chat.Id] = append(m.chats[chat.Id], client)
	}
	return nil
}

// RemoveClient removes websocket client from the manager and closes the connection
func (m *Manager) RemoveClient(ctx context.Context, client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	//Get user chats
	chats, err := m.store.GetChats(ctx, client.username)
	if err != nil {
		return err
	}

	//remove user from chats
	//Iterate through user chats
	for _, chat := range chats {
		//Iterate through chat's clients to delete this client from there
		for i, c := range m.chats[chat.Id] {
			//If we found our user delete them
			if c == client {
				if i < len(m.chats[chat.Id])-1 {
					m.chats[chat.Id] = append(m.chats[chat.Id][:i], m.chats[chat.Id][i+1:]...)
				} else {
					m.chats[chat.Id] = m.chats[chat.Id][:i]
				}
				//If chat has no clients connected set it to nil
				if len(m.chats[chat.Id]) == 0 {
					delete(m.chats, chat.Id)
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
