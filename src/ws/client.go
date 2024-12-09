package ws

import (
	"github.com/gorilla/websocket"
)

// ClientList is a map holding list of clients
type ClientList map[*Client]bool

// Client is a websocket client
type Client struct {
	connection *websocket.Conn
	manager    *Manager
	writer     chan []byte
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		writer:     make(chan []byte),
	}
}

// ReadMessages is run in a goroutine and is used for reading incoming messages over a websocket connection
func (c *Client) ReadMessages() {
	//Graceful close of the connection
	defer c.manager.RemoveClient(c)

	//Infinite loop
	for {
		mesageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			c.manager.logger.Errorw("error reading message", "err", err)
		}
		c.manager.logger.Infow("New message", "MessageType", mesageType, "Payload", payload)
		// Hack to test that WriteMessages works as intended
		// Will be replaced soon
		for wsclient := range c.manager.clients {
			wsclient.writer <- payload
		}
	}
}

// WriteMessages is run in a seperate goroutine and is used to write messages over websocket conneciton
func (c *Client) WriteMessages() {
	//Gracefully remove the client
	defer c.manager.RemoveClient(c)

	//Infinite loop
	for {
		select {
		case message, ok := <-c.writer:
			//Check if channel is closed
			if !ok {
				//Notify front end that connection channel is closed
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					c.manager.logger.Errorw("error writing message:", err)
				}
				return
			}
			//Write a regulat test message
			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				c.manager.logger.Errorw("error writing message:", err)
			}
			c.manager.logger.Infow("Message has been sent", "msg", message)
		}
	}

}
