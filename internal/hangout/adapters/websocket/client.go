package websocket

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	hangoutID uuid.UUID
	accountID uuid.UUID
}

func NewClient(conn *websocket.Conn, hangoutID, accountID uuid.UUID) *Client {
	return &Client{
		conn:      conn,
		send:      make(chan []byte, 16),
		hangoutID: hangoutID,
		accountID: accountID,
	}
}

func (c *Client) WritePump() {
	defer c.conn.Close()
	for msg := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
