package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	pongWait     = time.Second * 10
	pingInterval = (pongWait * 9) / 10
)

// ClientList is a map holding list of clients
type ClientList map[*Client]bool

// Client is a websocket client
type Client struct {
	connection *websocket.Conn
	manager    *Manager
	writer     chan Event
	logger     *zap.SugaredLogger
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		writer:     make(chan Event),
		logger:     manager.logger,
	}
}

// ReadMessages is run in a goroutine and is used for reading incoming messages over a websocket connection
func (c *Client) ReadMessages() {
	//Graceful close of the connection
	defer c.manager.RemoveClient(c)

	//Set message size limit to 512 bytes
	c.connection.SetReadLimit(512)

	//Configure wait time for pong responses
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Errorw("error setting pong wait time", "err", err)
		return
	}
	//Configure handling pong responses
	c.connection.SetPongHandler(c.PongHandler)

	//Infinite loop
	for {
		mesageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			c.logger.Errorw("error reading message", "err", err)
		}

		var request Event
		c.logger.Debugw("raw data", "data", string(payload))
		if err := json.Unmarshal(payload, &request); err != nil {
			c.logger.Errorw("error unmarshaling event", "err", err)
			break
		}

		c.logger.Infow("New message", "MessageType", mesageType, "Payload", payload)

		//Route the event
		if err := c.manager.routeEvent(request, c); err != nil {
			c.logger.Errorw("error handling message", "err", err)
		}
	}
}

// WriteMessages is run in a seperate goroutine and is used to write messages over websocket conneciton
func (c *Client) WriteMessages() {
	//Create new ticker
	ticker := time.NewTicker(pingInterval)

	//Gracefully remove the client
	defer func() {
		c.manager.RemoveClient(c)
		ticker.Stop()
	}()

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

			data, err := json.Marshal(message)
			if err != nil {
				c.logger.Errorw("error marshaling event", "err", err)
				return
			}
			//Write a regular test message
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Errorw("error writing message:", err)
			}
			c.logger.Infow("Message has been sent", "msg", message)

		case <-ticker.C:
			//Send the ping
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				c.logger.Errorw("error writing ping message", "err", err)
				return
			}

		}
	}

}

func (c *Client) PongHandler(pongMsg string) error {
	//Current time + pong.Wait time
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
