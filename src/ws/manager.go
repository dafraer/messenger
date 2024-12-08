package ws

import (
	"github.com/gorilla/websocket"
)

type Manager struct {
	WSUpgrader websocket.Upgrader
}

func NewManager() *Manager {
	return &Manager{
		WSUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}
