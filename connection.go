// Package pusher implements client library for pusher.com socket
package pusher

import (
	"github.com/pingdomserver/go.net/websocket"
	"fmt"
	"log"
	"time"
)

const (
	pusherUrl = "ws://ws.pusherapp.com:80/app/%s?protocol=7"
)

type Connection struct {
	key      string
	conn     *websocket.Conn
	logger 	 *log.Logger
	channels []*Channel
}

func New(key string, logger *log.Logger) (*Connection, error) {
	ws, err := websocket.Dial(fmt.Sprintf(pusherUrl, key), "", "http://localhost/")
	if err != nil {
		return nil, err
	}

	connection := &Connection{
		key:  key,
		conn: ws,
		logger: logger,
		channels: []*Channel{
			NewChannel(""),
		},
	}

	go connection.pong()
	go connection.poll()

	return connection, nil
}

func (c *Connection) pong() {
	tick := time.Tick(time.Minute)
	pong := NewPongMessage()
	for {
		<-tick
		websocket.JSON.Send(c.conn, pong)
	}
}

func (c *Connection) poll() {
	for {
		var msg Message
		err := websocket.JSON.Receive(c.conn, &msg)
		if err != nil {
			// 2019-04-05 hotfix: skip logging for now -- messages were flooding the log and filling up the disk.
			//                    This should be changed to log at most once per minute
			// c.logger.Println("Error reading data from socket")

			continue
		}

		c.processMessage(&msg)
	}
}

func (c *Connection) processMessage(msg *Message) {
	for _, channel := range c.channels {
		if channel.Name == msg.Channel {
			channel.processMessage(msg)
		}
	}
}

func (c *Connection) Disconnect() error {
	return c.conn.Close()
}

func (c *Connection) Channel(name string) *Channel {
	for _, channel := range c.channels {
		if channel.Name == name {
			return channel
		}
	}

	channel := NewChannel(name)
	c.channels = append(c.channels, channel)
	websocket.JSON.Send(c.conn, NewSubscribeMessage(name))

	return channel
}
