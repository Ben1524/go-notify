package websockettools

import (
	"context"
	"go-notify/model"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketStream struct {
	clients     map[uint][]*Client
	lock        sync.RWMutex
	pingPeriod  time.Duration
	pongTimeout time.Duration
	upgrader    websocket.Upgrader
	ctx         context.Context
}

// 辅助函数
func unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := []T{}
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func isAllowedOrigin(r *http.Request, allowedOrigins []*regexp.Regexp) bool {
	// 如果origin为空（通常表示同域请求），直接返回true
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin) // 解析请求的origin
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) { // 如果请求源和页面源相同，允许连接
		return true
	}
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin.MatchString(strings.ToLower(u.Hostname())) {
			return true
		}
	}
	return false
}

var (
	userID uint = 0
)

func (ws *WebSocketStream) GinHandler(ctx *gin.Context) {
	conn, err := ws.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	log.Print("WebSocket connected: ", ctx.Request.RemoteAddr)
	c := newClient(conn, userID, "", func(client *Client) {
		ws.RemoveClient(client.userID)
	})
	// 一个用户ID自增，实际应用中应使用真实用户ID，并且一个用户可能有多个连接
	userID++
	ws.AddClient(c)
	type Message struct {
		Type string `json:"type"`
		Body string `json:"body"`
	}
	// 启动读写协程，监听该连接的读写
	go c.startReading(ws.ctx, ws.pongTimeout)
	go c.startWriting(ws.ctx, ws.pingPeriod)
}

func NewWebSocketStream(ctx context.Context, pingPeriod, pongTimeout time.Duration,
	allowedWebSocketOrigins []string) *WebSocketStream {
	return &WebSocketStream{
		clients:     make(map[uint][]*Client),
		pingPeriod:  pingPeriod,
		pongTimeout: pongTimeout,
		upgrader:    newWebSocketUpgrader(allowedWebSocketOrigins),
		ctx:         ctx,
	}
}

func (ws *WebSocketStream) AddClient(client *Client) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	ws.clients[client.userID] = append(ws.clients[client.userID], client)
}
func (a *WebSocketStream) CollectConnectedClientTokens() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var clients []string
	for _, cs := range a.clients {
		for _, c := range cs {
			clients = append(clients, c.token)
		}
	}
	return unique(clients)
}

func (ws *WebSocketStream) RemoveClient(uid uint) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	if clients, ok := ws.clients[uid]; ok {
		for i, c := range clients {
			c.Close()
			clients[i] = nil
		}
		delete(ws.clients, uid)
	}
}

func (ws *WebSocketStream) RemoveClientByToken(userID uint, tokens ...string) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	if clients, ok := ws.clients[userID]; ok {
		for i, c := range clients {
			for _, token := range tokens {
				if c.token == token {
					c.Close()
					clients[i] = nil
					clients = append(clients[:i], clients[i+1:]...)
				}
			}
		}
		ws.clients[userID] = clients
	}
}

// 把允许的origin字符串编译成正则表达式
func compileAllowedWebSocketOrigins(allowedOrigins []string) []*regexp.Regexp {
	var origins []*regexp.Regexp
	for _, origin := range allowedOrigins {
		origins = append(origins, regexp.MustCompile(origin))
	}
	return origins
}

func newWebSocketUpgrader(allowedOrigins []string) websocket.Upgrader {
	origins := compileAllowedWebSocketOrigins(allowedOrigins)
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r, origins)
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

func (ws *WebSocketStream) BroadcastMessage(userIDs []uint, message *model.MessageExternal) {
	ws.lock.RLock()
	defer ws.lock.RUnlock()
	for _, uid := range userIDs {
		if clients, ok := ws.clients[uid]; ok {
			for _, c := range clients {
				c.write <- message
			}
		}
	}
}

func (ws *WebSocketStream) Close() {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	for uid, clients := range ws.clients {
		for _, c := range clients {
			c.Close()
			c = nil
		}
		delete(ws.clients, uid)
	}
	ws.clients = make(map[uint][]*Client) // 清空map,交给GC
}

func (ws *WebSocketStream) SendMessage(userID uint, message *model.MessageExternal) {
	ws.lock.RLock()
	defer ws.lock.RUnlock()
	if clients, ok := ws.clients[userID]; ok {
		for _, c := range clients {
			c.write <- message
		}
	}
}
