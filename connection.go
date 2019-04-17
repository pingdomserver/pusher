// Package pusher implements client library for pusher.com socket
package pusher

import (
	"github.com/pingdomserver/go.net/websocket"
	"fmt"
	"log"
	"time"
)
// https://pusher.com/docs/channels/library_auth_reference/pusher-websockets-protocol#websocket-connection
const (
	pusherUrl = "ws://ws-%s.pusher.com:80/app/%s?protocol=7"
)

type Connection struct {
	key      string
	conn     *websocket.Conn
	logger 	 *log.Logger
	channels []*Channel
}

func New(key string, cluster string, logger *log.Logger) (*Connection, error) {
	ws, err := websocket.Dial(fmt.Sprintf(pusherUrl, cluster, key), "", "http://localhost/")
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
	lastLogTime := time.Now()
	for {
		var msg Message
		err := websocket.JSON.Receive(c.conn, &msg)
		if err != nil {
			delta := time.Now().Sub(lastLogTime)
			if delta > 1 * time.Minute {
				lastLogTime = time.Now()

				c.logger.Println("Error reading data from socket")
			}
			time.Sleep(500 * time.Millisecond)

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
