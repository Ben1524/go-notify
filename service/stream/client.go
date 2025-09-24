package websockettools

import (
	"context"
	"github.com/gorilla/websocket"
	"go-notify/model"
	"log"
	"sync"
	"time"
)

const (
	writeWait = 2 * time.Second
)

var ping = func(conn *websocket.Conn) error {
	return conn.WriteMessage(websocket.PingMessage, nil)
}

var writeJSON = func(conn *websocket.Conn, v interface{}) error {
	return conn.WriteJSON(v)
}

type Client struct {
	conn    *websocket.Conn
	onClose func(*Client)
	write   chan *model.MessageExternal // 写消息的管道，批量发送
	userID  uint                        // 用户ID
	token   string                      // 连接的令牌
	sync.Once
}

func newClient(conn *websocket.Conn, userID uint, token string, onClose func(*Client)) *Client {
	return &Client{
		conn:    conn,
		write:   make(chan *model.MessageExternal, 1),
		userID:  userID,
		token:   token,
		onClose: onClose,
	}
}

func (c *Client) Close() {
	c.Do(func() {
		close(c.write)
		_ = c.conn.Close()
	})
}

func (c *Client) NotifyClose() {
	if c.onClose != nil {
		c.Do(func() {
			c.conn.Close()
			close(c.write)
			c.onClose(c)
		})
	}
}

func (c *Client) startReading(ctx context.Context, pongWait time.Duration) {
	defer c.NotifyClose()
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetReadLimit(512)
	for {
		log.Print("Waiting for WebSocket message from ", c.conn.RemoteAddr())
		select {
		case <-ctx.Done():
			return
		default:
			msgType, msgData, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.CloseMessage {
				return
			}
			// 处理收到的消息 msgData
			log.Print("Received message: ", string(msgData))
		}
	}
}

func (c *Client) startWriting(ctx context.Context, pingPeriod time.Duration) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		c.NotifyClose()
		pingTicker.Stop()
	}()
	log.Print("WebSocket connection established: ", c.conn.RemoteAddr())
	for {
		select {
		case message, ok := <-c.write:
			if !ok {
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := writeJSON(c.conn, message); err != nil {
				printWebSocketError("WriteError", err)
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ping(c.conn); err != nil {
				printWebSocketError("PingError", err)
				return
			}
		case <-ctx.Done():
			return
		default:
			// No messages to send, just wait a bit
			time.Sleep(50 * time.Millisecond)

		}
	}
}

func printWebSocketError(s string, err error) {
	log.Printf("WebSocket %s: %v", s, err)
}
