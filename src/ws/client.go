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

// Message struct is the message that we receive over websocket
type Message struct {
	From   string `json:"from"`
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

// Client is a websocket client
type Client struct {
	username   string
	connection *websocket.Conn
	manager    *Manager
	//writer is a channel over which we send messages
	writer chan Message
	logger *zap.SugaredLogger
}

// NewClient creates new websocket client
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
func (c *Client) ReadMessages(ctx context.Context) {
	//Graceful close of the connection
	defer func() {
		if err := c.manager.RemoveClient(ctx, c); err != nil {
			c.logger.Errorw("Error removing client", "error", err)
		}
	}()

	//Set message size limit to 512 bytes
	c.connection.SetReadLimit(512)

	//Configure wait time for pong responses
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Errorw("Error setting pong wait time", "error", err)
		return
	}

	//Configure handling pong responses
	c.connection.SetPongHandler(func(string) error {
		if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.logger.Errorw("Error setting pong wait time", "error", err)
			return err
		}
		return nil
	})

	//Infinite loop in which we read messages from the websocket connection
	for {
		//Read message from websocket connection
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			c.logger.Errorw("Error reading message", "error", err)
			return
		}

		//Unmarshal message into message struct
		var request Message
		if err := json.Unmarshal(payload, &request); err != nil {
			c.logger.Errorw("Error unmarshalling event", "error", err)
			continue
		}

		//Set message author to the actual client to prevent impersonation
		request.From = c.username

		//Iterate through chat members and send message
		for _, client := range c.manager.chats[request.ChatId] {
			if client != c && client != nil {
				client.writer <- request
			}
		}

		//Save message to the database
		if err := c.manager.store.SaveMessage(context.TODO(), store.Message{ChatId: request.ChatId, From: c.username, Text: request.Text, Time: time.Now().UTC().Unix()}); err != nil {
			c.logger.Errorw("Error saving message", "error", err)
		}
	}
}

// WriteMessages is run in a separate goroutine and is used to write messages over websocket connection
func (c *Client) WriteMessages(ctx context.Context) {
	//Create new ticker
	ticker := time.NewTicker(pingInterval)

	//Gracefully remove the client
	defer func() {
		if err := c.manager.RemoveClient(ctx, c); err != nil {
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
					c.manager.logger.Errorw("Error writing message:", "error", err)
				}
				return
			}

			//Marshal message into json
			data, err := json.Marshal(message)
			if err != nil {
				c.logger.Errorw("Error marshaling message", "error", err)
				return
			}

			//Write text message to the websocket connection
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Errorw("Error writing message:", "error", err)
			}
		case <-ticker.C:
			//Send the ping
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				c.logger.Errorw("Error writing ping message", "error", err)
				return
			}

		}
	}

}
