package wss

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Connection struct {
	ws   *websocket.Conn
	send chan *Message
}

// write writes a message with the given message type and payload.
func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Connection) writePump(s *Service) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			buf, err := json.Marshal(message)
			if err != nil {
				return
			}
			if err := c.write(websocket.TextMessage, buf); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Connection) Send(id MsgID, data interface{}) error {
	jdata, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return err
	}
	select {
	case c.send <- &Message{id, (*json.RawMessage)(&jdata)}:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// Read a JSON msg from the websocket
func (c *Connection) Receive() (*Message, error) {
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	c.ws.ReadMessage()
	var msg Message
	if err := c.ws.ReadJSON(&msg); err != nil {
		c.ws.Close()
		return nil, err
	}
	return &msg, nil
}
