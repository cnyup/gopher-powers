// server.go 对齐 server/index.mjs 双人（maxPlayers=2）路径。
package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cnyup/gopher-powers/internal/game"
	"github.com/cnyup/gopher-powers/internal/protocol"
	"github.com/cnyup/gopher-powers/internal/store"
)

// Config 服务配置。
type Config struct {
	Directory  string // server-data
	MaxPlayers int    // 2（六方 3 留待后续）
}

// member 房间席位。
type member struct {
	user      *store.User
	faction   string
	ready     bool
	seat      int
	left      bool
	controlType string
}

// userID 席位用户 id；AI 席位无 user，返回空串（避免 nil 解引用）。
func (m *member) userID() string {
	if m == nil || m.user == nil {
		return ""
	}
	return m.user.ID
}

// isAI 该席位是否由 AI 控制。
func (m *member) isAI() bool {
	return m != nil && m.controlType == "ai"
}

// room 对齐 index.mjs room 对象。
type room struct {
	code        string
	host        int
	config      map[string]any
	members     [2]*member
	expires     time.Time
	memory      [2]viewMemory
	game        *game.Game
	reason      string
	settled     bool
	deadline    time.Time
	disconnects int
}

// peer 一个在线连接。
type peer struct {
	mu   sync.Mutex
	conn peerConn
	user *store.User
	sendQ   chan []byte
	lastSeq int
	tokens  float64
	lastAct time.Time
}

// Server 对战服务。
type Server struct {
	mu        sync.Mutex
	accounts  *store.Accounts
	rooms     map[string]*room
	peers     map[string]*peer
	requests  map[string]*rateEntry
	maxPlayers int
	tick      int
	effectID  int
	closeCh   chan struct{}
}

type rateEntry struct {
	until time.Time
	count int
}

// New 创建服务（不开 HTTP——由调用方决定路由挂载）。
func New(cfg Config) (*Server, error) {
	if cfg.MaxPlayers != 2 && cfg.MaxPlayers != 6 {
		return nil, fmt.Errorf("席位上限无效")
	}
	if cfg.MaxPlayers == 6 {
		return nil, fmt.Errorf("六方暂未开放")
	}
	dir := cfg.Directory
	if dir == "" {
		dir = "server-data"
	}
	acc, err := store.Open(dir, "gp_guest")
	if err != nil {
		return nil, err
	}
	return &Server{
		accounts:  acc,
		rooms:     map[string]*room{},
		peers:     map[string]*peer{},
		requests:  map[string]*rateEntry{},
		maxPlayers: cfg.MaxPlayers,
		closeCh:   make(chan struct{}),
	}, nil
}

// Close 停服务。
func (s *Server) Close() {
	close(s.closeCh)
	s.accounts.Close()
}

// Accounts 暴露账面（测试用）。
func (s *Server) Accounts() *store.Accounts { return s.accounts }

// ---- HTTP 面 ----

// HealthJSON 对齐 /api/health。
func (s *Server) HealthJSON(version string) map[string]any {
	s.mu.Lock()
	playing := 0
	for _, r := range s.rooms {
		if r.game != nil && r.game.Running {
			playing++
		}
	}
	n := len(s.rooms)
	s.mu.Unlock()
	return map[string]any{"ok": true, "version": version, "protocol": protocol.Version, "maxPlayers": s.maxPlayers, "rooms": n, "battles": playing}
}

// SessionGet 对齐 GET /api/session。
func (s *Server) SessionGet(cookie string) map[string]any {
	user := s.accounts.Find(cookie)
	if user == nil {
		return map[string]any{"user": nil, "room": nil}
	}
	r := s.userRoom(user.ID)
	return map[string]any{"user": user, "room": s.publicRoomOrNil(r)}
}

// SessionPost 对齐 POST /api/session（限流/昵称/建号或改名）。返回 (json, setCookie, err)。
func (s *Server) SessionPost(cookie, origin, ip string, body []byte) (map[string]any, string, error) {
	now := time.Now()
	s.mu.Lock()
	if e, ok := s.requests[ip]; ok && e.until.After(now) && e.count >= 30 {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("操作过于频繁，请稍后再试")
	}
	if e, ok := s.requests[ip]; ok && e.until.After(now) {
		e.count++
	} else {
		s.requests[ip] = &rateEntry{until: now.Add(time.Minute), count: 1}
	}
	s.mu.Unlock()

	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, "", fmt.Errorf("请求无效")
	}
	name, err := protocol.Nickname(in.Name)
	if err != nil {
		return nil, "", err
	}
	user := s.accounts.Find(cookie)
	if user != nil {
		if err := s.accounts.Rename(user.ID, name); err != nil {
			return nil, "", err
		}
		user.Name = name
		return map[string]any{"user": user, "room": nil}, "", nil
	}
	token, created, err := s.accounts.Create(name)
	if err != nil {
		return nil, "", err
	}
	cookieStr := fmt.Sprintf("gp_guest=%s; HttpOnly; SameSite=Lax; Path=/; Max-Age=2592000", token)
	return map[string]any{"user": created, "room": nil}, cookieStr, nil
}

// ---- 房间/WS 面 ----

func (s *Server) userRoom(id string) *room {
	for _, r := range s.rooms {
		for _, m := range r.members {
			if m != nil && m.user != nil && m.user.ID == id && !m.left {
				return r
			}
		}
	}
	return nil
}

func (s *Server) publicRoom(r *room) map[string]any {
	if r == nil {
		return nil
	}
	status := "waiting"
	if r.game != nil {
		if r.game.Running {
			status = "playing"
		} else {
			status = "ended"
		}
	}
	members := make([]any, 2)
	for i, m := range r.members {
		if m == nil {
			continue
		}
		connected := m.isAI() || s.peerOnline(m.userID())
		var eliminated bool
		if r.game != nil && i < len(r.game.Players) {
			eliminated = r.game.Players[i].Eliminated
		}
		name := "AI"
		if m.user != nil {
			name = m.user.Name
		}
		members[i] = map[string]any{
			"id": m.userID(), "name": name, "faction": m.faction,
			"ready": m.ready, "connected": connected,
			"controlType": orDefault(m.controlType, "human"), "eliminated": eliminated,
		}
	}
	var deadline any = nil
	if !r.deadline.IsZero() {
		deadline = r.deadline.UnixMilli()
	}
	return map[string]any{
		"code": r.code, "host": r.host, "config": r.config, "status": status,
		"reason": orDefault(r.reason, ""), "deadline": deadline,
		"members": members,
	}
}

func (s *Server) publicRoomOrNil(r *room) map[string]any {
	if r == nil {
		return map[string]any{"code": nil, "config": nil, "members": []any{}, "host": 0}
	}
	return s.publicRoom(r)
}

func (s *Server) peerOnline(id string) bool { _, ok := s.peers[id]; return ok }

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// enemyFactionOf 单机模式下 AI 席位的阵营：确定性取一个与玩家不同的合法阵营。
func enemyFactionOf(player string) string {
	for _, f := range []string{"russia", "china", "usa", "europe"} {
		if f != player && game.ValidFaction(f) {
			return f
		}
	}
	return "russia"
}

// Send 给指定用户发 JSON 消息。
func (s *Server) Send(userID string, v any) {
	p := s.peers[userID]
	if p == nil {
		return
	}
	b, _ := json.Marshal(v)
	p.conn.Send(b)
}

func (s *Server) roomNotice(r *room) {
	for _, m := range r.members {
		if m != nil {
			if id := m.userID(); id != "" {
				s.Send(id, map[string]any{"type": "room", "room": s.publicRoom(r)})
			}
		}
	}
}

func (s *Server) snapshot(r *room, userID string, side int, start bool) {
	g := r.game
	cfg := map[string]any{}
	for k, v := range r.config {
		cfg[k] = v
	}
	cfg["battleId"] = g.BattleID
	s.Send(userID, map[string]any{
		"type": map[bool]string{true: "battle", false: "snapshot"}[start],
		"config": cfg, "seat": side, "room": s.publicRoom(r), "spectator": false,
		"view": battleView(g, side, r.memory[side], false),
	})
}

func (s *Server) end(r *room, winner any, reason string) {
	if r.settled {
		return
	}
	switch w := winner.(type) {
	case int:
		r.game.Winner = w
	}
	r.game.Running = false
	r.game.Paused = false
	r.reason = reason
	r.settled = true
	ws := "draw"
	if w, ok := winner.(int); ok {
		ws = fmt.Sprint(w)
	}
	// 战绩结算：两侧都必须有真实 user。单机局 AI 席位无 user，不落库
	// （源项目单机存档在浏览器 localStorage，不写 matches——重写方案 §1.3）。
	// 源 accounts.mjs 的 settle 直接解引用 users[i].id，因为源项目没有 solo 模式；
	// 我们新增 solo 后此处必须显式守卫，否则 AI 席位会让整个服务进程 SIGSEGV。
	var su [2]*store.User
	canSettle := true
	for i, m := range r.members {
		if m == nil || m.user == nil {
			canSettle = false
			break
		}
		su[i] = m.user
	}
	if canSettle {
		_ = s.accounts.Settle(r.game.BattleID, su, ws)
	}
	for side, m := range r.members {
		if m != nil && !m.left && m.controlType != "ai" {
			if id := m.userID(); id != "" {
				s.snapshot(r, id, side, false)
			}
		}
	}
	s.roomNotice(r)
}

func (s *Server) leave(userID string) {
	r := s.userRoom(userID)
	if r == nil {
		return
	}
	side := -1
	for i, m := range r.members {
		if m != nil && m.user != nil && m.user.ID == userID {
			side = i
			break
		}
	}
	if r.game != nil && r.game.Running {
		s.end(r, 1-side, "对手认输退出")
	}
	if r.game != nil {
		for _, m := range r.members {
			if m != nil {
				if id := m.userID(); id != "" {
					s.Send(id, map[string]any{"type": "room", "room": nil})
				}
			}
		}
		delete(s.rooms, r.code)
	} else {
		r.members[side] = nil
		anyLeft := false
		for _, m := range r.members {
			if m != nil {
				anyLeft = true
			}
		}
		if !anyLeft {
			delete(s.rooms, r.code)
		} else {
			r.host = firstHost(r)
			if r.host < 0 {
				delete(s.rooms, r.code)
				s.Send(userID, map[string]any{"type": "room", "room": nil})
				return
			}
			s.roomNotice(r)
		}
	}
	s.Send(userID, map[string]any{"type": "room", "room": nil})
}

func firstHost(r *room) int {
	for i, m := range r.members {
		if m != nil {
			return i
		}
	}
	return -1
}

// HandleMessage 对齐 ws.on('message') 主面。返回 error 时发 error 消息。
func (s *Server) HandleMessage(userID string, raw []byte) error {
	p := s.peers[userID]
	if p == nil {
		return nil
	}
	// 令牌桶：40 token，0.02/ms 恢复。
	now := time.Now()
	s.mu.Lock()
	p.tokens = minF(40, p.tokens+now.Sub(p.lastAct).Seconds()*1000*0.02)
	p.lastAct = now
	p.tokens--
	if p.tokens < 0 {
		s.mu.Unlock()
		return fmt.Errorf("__RATE__")
	}
	s.mu.Unlock()

	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("指令格式无效")
	}
	mtype, _ := msg["type"].(string)
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.userRoom(userID)
	side := -1
	if r != nil {
		for i, m := range r.members {
			if m != nil && m.user != nil && m.user.ID == userID {
				side = i
			}
		}
	}
	switch mtype {
	case "ping":
		at := msg["at"]
		s.Send(userID, map[string]any{"type": "pong", "at": at})
		return nil
	case "leave":
		s.leave(userID)
		return nil
	case "create":
		if r != nil {
			return fmt.Errorf("请先退出当前房间")
		}
		if len(s.rooms) >= 20 {
			return fmt.Errorf("房间已满，请稍后重试")
		}
		mapID, _ := msg["mapId"].(string)
		faction, _ := msg["faction"].(string)
		victoryMode, _ := msg["victoryMode"].(string)
		if !game.ValidMap2P(mapID) || !game.ValidFaction(faction) || !game.ValidVictoryMode(victoryMode) {
			return fmt.Errorf("作战配置无效")
		}
		code := s.newRoomCode()
		solo, _ := msg["solo"].(bool)
		nr := &room{
			code: code, host: 0,
			config: map[string]any{"mapId": mapID, "victoryMode": victoryMode, "battlefieldScale": 1.5, "solo": solo},
			memory: [2]viewMemory{{}, {}},
			expires: time.Now().Add(10 * time.Minute),
		}
		nr.members[0] = &member{user: s.peersUser(userID), faction: faction, seat: 0, controlType: "human"}
		if solo {
			// 单机：1 号席位由 AI 接管，无 user、恒在线、开局即可开战。
			nr.members[1] = &member{faction: enemyFactionOf(faction), seat: 1, ready: true, controlType: "ai"}
		}
		s.rooms[code] = nr
		s.roomNotice(nr)
		return nil
	case "join":
		if r != nil {
			return fmt.Errorf("请先退出当前房间")
		}
		code, _ := msg["code"].(string)
		faction, _ := msg["faction"].(string)
		r = s.rooms[code]
		if r == nil || r.game != nil || (r.members[0] != nil && r.members[1] != nil) {
			return fmt.Errorf("房间不存在、已满或已开战")
		}
		if !game.ValidFaction(faction) {
			return fmt.Errorf("阵营无效")
		}
		seat := 0
		if r.members[0] != nil {
			seat = 1
		}
		r.members[seat] = &member{user: s.peersUser(userID), faction: faction, seat: seat, controlType: "human"}
		r.expires = time.Now().Add(10 * time.Minute)
		s.roomNotice(r)
		return nil
	case "ready", "faction":
		if r == nil {
			return fmt.Errorf("请先创建或加入房间")
		}
		if r.game != nil {
			return fmt.Errorf("已经开战，不能修改部署")
		}
		if mtype == "faction" {
			faction, _ := msg["faction"].(string)
			if !game.ValidFaction(faction) {
				return fmt.Errorf("阵营无效")
			}
			r.members[side].faction = faction
			r.members[side].ready = false
		} else {
			r.members[side].ready = msg["ready"] == true
		}
		r.expires = time.Now().Add(10 * time.Minute)
		s.roomNotice(r)
		return nil
	case "start":
		if r == nil {
			return fmt.Errorf("请先创建或加入房间")
		}
		if r.game != nil || side != r.host {
			return fmt.Errorf("双方准备完成后，由房主开战")
		}
		for _, m := range r.members {
			if m == nil || !m.ready || (!m.isAI() && !s.peerOnline(m.userID())) {
				return fmt.Errorf("双方准备完成后，由房主开战")
			}
		}
		playing := 0
		for _, v := range s.rooms {
			if v.game != nil && v.game.Running {
				playing++
			}
		}
		if playing >= 4 {
			return fmt.Errorf("战区繁忙，请稍后开战")
		}
		mapID, _ := r.config["mapId"].(string)
		victoryMode, _ := r.config["victoryMode"].(string)
		scale, _ := r.config["battlefieldScale"].(float64)
		solo, _ := r.config["solo"].(bool)
		g, err := game.New(r.members[0].faction, r.members[1].faction, &game.Events{}, game.Options{
			Multiplayer: !solo, VictoryMode: victoryMode, MapID: mapID, BattlefieldScale: scale,
		})
		if err != nil {
			return err
		}
		r.game = g
		for seat, m := range r.members {
			if id := m.userID(); id != "" {
				s.snapshot(r, id, seat, true)
			}
		}
		s.roomNotice(r)
		return nil
	case "command":
		if r == nil {
			return fmt.Errorf("请先创建或加入房间")
		}
		seqF, ok := msg["seq"].(float64)
		if !ok || int64(seqF) <= int64(p.lastSeq) {
			return nil
		}
		p.lastSeq = int(seqF)
		ok2 := false
		if r.game != nil {
			ok2 = ExecuteCommand(r.game, side, msg)
		}
		s.Send(userID, map[string]any{"type": "ack", "seq": int(seqF), "ok": ok2, "action": msg["action"]})
		return nil
	default:
		return fmt.Errorf("未知请求")
	}
}

func (s *Server) peersUser(id string) *store.User {
	if p, ok := s.peers[id]; ok {
		return p.user
	}
	return nil
}

func (s *Server) newRoomCode() string {
	for {
		n, _ := rand.Int(rand.Reader, big.NewInt(900000))
		code := fmt.Sprintf("%06d", n.Int64()+100000)
		if _, ok := s.rooms[code]; !ok {
			return code
		}
	}
}

// OnConnect 对齐 wss connection（hello + 恢复）。
func (s *Server) OnConnect(userID string, user *store.User, conn peerConn) {
	p := &peer{conn: conn, tokens: 40, lastAct: time.Now(), user: user}
	s.mu.Lock()
	s.peers[userID] = p
	r := s.userRoom(userID)
	s.mu.Unlock()
	s.Send(userID, map[string]any{"type": "hello", "protocol": protocol.Version, "user": user, "room": s.publicRoom(r)})
	if r != nil {
		s.roomNotice(r)
		if r.game != nil {
			side := -1
			for i, m := range r.members {
				if m != nil && m.user != nil && m.user.ID == userID {
					side = i
				}
			}
			s.snapshot(r, userID, side, true)
		}
	}
}

// OnDisconnect 对齐 ws close。
func (s *Server) OnDisconnect(userID string) {
	s.mu.Lock()
	delete(s.peers, userID)
	r := s.userRoom(userID)
	s.mu.Unlock()
	if r != nil {
		s.roomNotice(r)
	}
}

// Tick 对齐 50ms setInterval（一次推进）。
func (s *Server) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tick++
	now := time.Now()
	for code, r := range s.rooms {
		if r.game == nil || !r.game.Running {
			if now.After(r.expires) {
				for _, m := range r.members {
					if m != nil {
						if id := m.userID(); id != "" {
					s.Send(id, map[string]any{"type": "room", "room": nil})
				}
					}
				}
				delete(s.rooms, code)
			}
			continue
		}
		d0 := r.members[0] == nil || (!r.members[0].isAI() && !s.peerOnline(r.members[0].userID()))
		d1 := r.members[1] == nil || (!r.members[1].isAI() && !s.peerOnline(r.members[1].userID()))
		if d0 || d1 {
			r.game.Paused = true
			if r.deadline.IsZero() {
				r.disconnects++
				ms := time.Minute
				if r.disconnects > 3 {
					ms = 10 * time.Second
				}
				r.deadline = now.Add(ms)
				s.roomNotice(r)
			}
			if now.After(r.deadline) {
				winner := any("draw")
				if d0 && !d1 {
					winner = 1
				} else if d1 && !d0 {
					winner = 0
				}
				s.end(r, winner, "掉线宽限结束")
				continue
			}
		} else {
			r.game.Paused = false
			if !r.deadline.IsZero() {
				r.deadline = time.Time{}
				s.roomNotice(r)
			}
			r.game.Update(0.05)
		}
		for _, e := range r.game.Effects {
			_ = e
			s.effectID++
		}
		if !r.game.Running {
			s.end(r, r.game.Winner, "战场目标已达成")
			continue
		}
		if s.tick%2 == 0 {
			for side, m := range r.members {
				if m != nil {
					if id := m.userID(); id != "" {
				s.snapshot(r, id, side, false)
			}
				}
			}
		}
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// peerConn 抽象 WS 连接（真实 gorilla / 测试 fake）。
type peerConn interface {
	Send([]byte)
	Close(code int, reason string)
}

// ServeHTTP 挂 HTTP API 面（health/session）。
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	switch req.URL.Path {
	case "/api/health":
		_ = enc.Encode(s.HealthJSON("dev"))
	case "/api/session":
		cookie := req.Header.Get("Cookie")
		if req.Method == http.MethodPost {
			ip := req.RemoteAddr
			body := make([]byte, 0, 1024)
			buf := make([]byte, 1024)
			for {
				n, err := req.Body.Read(buf)
				body = append(body, buf[:n]...)
				if err != nil || len(body) > 1024 {
					break
				}
			}
			out, setCookie, err := s.SessionPost(cookie, req.Header.Get("Origin"), ip, body)
			if err != nil {
				w.WriteHeader(429)
				_ = enc.Encode(map[string]string{"error": err.Error()})
				return
			}
			if setCookie != "" {
				w.Header().Set("Set-Cookie", setCookie)
			}
			_ = enc.Encode(out)
			return
		}
		_ = enc.Encode(s.SessionGet(cookie))
	default:
		w.WriteHeader(404)
		_ = enc.Encode(map[string]string{"error": "文件不存在"})
	}
}

var _ = log.Println
var _ = strings.TrimSpace
