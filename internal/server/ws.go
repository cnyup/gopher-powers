// ws.go：gorilla/websocket 接线——升级 /battle-socket，桥接 Server 核心。
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WSServer 真实 WebSocket 层（对齐 index.mjs wss 语义：maxPayload 16KB、
// 心跳 15s、buffered 背压丢弃）。
type WSServer struct {
	core     *Server
	upgrader websocket.Upgrader
	allowed  func(r *http.Request) bool
}

// NewWSServer 建 WS 层。dev=true 时放行 localhost:4173 预览源。
func NewWSServer(core *Server, origin string, dev bool) *WSServer {
	return &WSServer{
		core: core,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		allowed: func(r *http.Request) bool {
			o := r.Header.Get("Origin")
			host := "http://" + r.Host
			return o == origin || o == host || o == "" ||
				(dev && (o == "http://localhost:4173" || o == "http://127.0.0.1:4173"))
		},
	}
}

type wsConn struct {
	ws     *websocket.Conn
	closed atomic.Bool
}

func (c *wsConn) Send(b []byte) {
	if c.closed.Load() {
		return
	}
	_ = c.ws.WriteControl(websocket.TextMessage, nil, time.Time{})
	// 直接写数据帧；写失败由读侧心跳/对端关闭兜底。
	if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
		c.closed.Store(true)
	}
}
func (c *wsConn) Close(code int, reason string) {
	if c.closed.Swap(true) {
		return
	}
	msg := websocket.FormatCloseMessage(code, reason)
	_ = c.ws.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
	_ = c.ws.Close()
}

// ServeWS 处理 /battle-socket 升级与消息泵。
func (w *WSServer) ServeWS(rw http.ResponseWriter, req *http.Request) {
	user := w.core.accounts.Find(req.Header.Get("Cookie"))
	if req.URL.Path != "/battle-socket" || !w.allowed(req) || user == nil {
		rw.WriteHeader(http.StatusForbidden)
		_, _ = rw.Write([]byte("HTTP/1.1 403 Forbidden"))
		return
	}
	ws, err := w.upgrader.Upgrade(rw, req, nil)
	if err != nil {
		return
	}
	conn := &wsConn{ws: ws}
	w.core.OnConnect(user.ID, user, conn)
	defer func() {
		w.core.OnDisconnect(user.ID)
		conn.Close(1000, "")
	}()
	ws.SetReadLimit(16384)
	const rateKick = "__RATE__"
	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if err := w.core.HandleMessage(user.ID, raw); err != nil {
			if strings.Contains(err.Error(), rateKick) {
				conn.Close(1008, "操作过于频繁")
				return
			}
			em := map[string]any{"type": "error"}
			if isSyntaxJSON(raw) {
				em["message"] = "指令格式无效"
			} else {
				em["message"] = err.Error()
			}
			b, _ := json.Marshal(em)
			conn.Send(b)
		}
	}
}

func isSyntaxJSON(raw []byte) bool {
	var v any
	return json.Unmarshal(raw, &v) != nil
}

// StartTicker 启 50ms 推进循环（阻塞直到 stop 关闭）。
func (s *Server) StartTicker(stop <-chan struct{}) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.Tick()
		}
	}
}
