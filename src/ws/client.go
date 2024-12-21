package ws

import (
	"context"
	"encoding/json"
	"github.com/dafraer/messenger/src/store"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	pongWait     = time.Second * 10
	pingInterval = (pongWait * 9) / 10
)

type Message struct {
	From   string `json:"from"`
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

// ClientList is a map holding list of clients
type ClientList map[*Client]bool

// Client is a websocket client
type Client struct {
	username   string
	connection *websocket.Conn
	manager    *Manager
	writer     chan Message
	logger     *zap.SugaredLogger
}

func NewClient(conn *websocket.Conn, manager *Manager, username string) *Client {
	return &Client{
		username:   username,
		connection: conn,
		manager:    manager,
		writer:     make(chan Message),
		logger:     manager.logger,
	}
}

// ReadMessages is run in a goroutine and is used for reading incoming messages over a websocket connection
func (c *Client) ReadMessages() {
	//Graceful close of the connection
	defer func() {
		if err := c.manager.RemoveClient(c); err != nil {
			c.logger.Errorw("Error removing client", "error", err)
		}
	}()

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
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			c.logger.Errorw("error reading message", "err", err)
		}

		var request Message
		c.logger.Debugw("raw data", "data", string(payload))
		if err := json.Unmarshal(payload, &request); err != nil {
			c.logger.Errorw("error unmarshalling event", "err", err)
			break
		}

		c.logger.Infow("New message", "MessageType", messageType, "Payload", request)

		//set message author to the actual client to prevent malicious use
		request.From = c.username

		//Iterate through chat members and send message
		//Do I really need to have a map here?
		for _, client := range c.manager.chats[request.ChatId] {
			if client != c && client != nil {
				client.writer <- request
			}
		}

		//Save message to the database
		if err := c.manager.store.SaveMessage(context.TODO(), store.Message{ChatId: request.ChatId, From: c.username, Text: request.Text, Time: time.Now().UTC().Unix()}); err != nil {
			c.logger.Errorw("error saving message", "err", err)
		}
	}
}

// WriteMessages is run in a separate goroutine and is used to write messages over websocket connection
func (c *Client) WriteMessages() {
	//Create new ticker
	ticker := time.NewTicker(pingInterval)

	//Gracefully remove the client
	defer func() {
		if err := c.manager.RemoveClient(c); err != nil {
			c.logger.Errorw("error removing client", "error", err)
		}
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
			//Write a regular text message
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
