package server

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/cnyup/gopher-powers/internal/store"
)

// fakeConn 测试用 WS 连接。
type fakeConn struct {
	mu   sync.Mutex
	msgs []map[string]any
}

func (c *fakeConn) Send(b []byte) {
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	c.mu.Lock()
	c.msgs = append(c.msgs, m)
	c.mu.Unlock()
}
func (c *fakeConn) Close(code int, reason string) {}

func (c *fakeConn) last(t string) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.msgs) - 1; i >= 0; i-- {
		if c.msgs[i]["type"] == t {
			return c.msgs[i]
		}
	}
	return nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{Directory: t.TempDir(), MaxPlayers: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func connectUser(t *testing.T, s *Server, name string) (*fakeConn, *store.User) {
	t.Helper()
	token, user, err := s.Accounts().Create(name)
	if err != nil {
		t.Fatal(err)
	}
	_ = token
	conn := &fakeConn{}
	s.OnConnect(user.ID, user, conn)
	if hello := conn.last("hello"); hello == nil {
		t.Fatalf("%s 未收到 hello", name)
	}
	return conn, user
}

func send(t *testing.T, s *Server, u *store.User, msg string) error {
	t.Helper()
	return s.HandleMessage(u.ID, []byte(msg))
}

// 集成：双人建房→加入→准备→开战→指令→tick 快照。
func TestFullBattleFlow(t *testing.T) {
	s := newTestServer(t)
	connA, userA := connectUser(t, s, "玩家甲")
	connB, userB := connectUser(t, s, "玩家乙")
	_ = connB

	// create。
	if err := send(t, s, userA, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation"}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	roomMsg := connA.last("room")
	if roomMsg == nil || roomMsg["room"] == nil {
		t.Fatal("create 后无 room 广播")
	}
	code, _ := roomMsg["room"].(map[string]any)["code"].(string)
	if len(code) != 6 {
		t.Fatalf("房间码 = %q", code)
	}

	// join。
	if err := send(t, s, userB, `{"type":"join","code":"`+code+`","faction":"russia"}`); err != nil {
		t.Fatalf("join: %v", err)
	}

	// ready。
	if err := send(t, s, userA, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready A: %v", err)
	}
	if err := send(t, s, userB, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready B: %v", err)
	}

	// start（非房主应拒）。
	if err := send(t, s, userB, `{"type":"start"}`); err == nil {
		t.Fatal("非房主 start 应报错")
	}
	if err := send(t, s, userA, `{"type":"start"}`); err != nil {
		t.Fatalf("start: %v", err)
	}

	battleA := connA.last("battle")
	if battleA == nil {
		t.Fatal("未收到 battle")
	}
	if battleA["seat"] != float64(0) {
		t.Fatalf("seat = %v", battleA["seat"])
	}
	view, _ := battleA["view"].(map[string]any)
	if view == nil {
		t.Fatal("battle 无 view")
	}
	// 席位重排锚点：seat 0 的玩家 credits 非零、seat 1 credits=0。
	players, _ := view["players"].([]any)
	if len(players) != 2 {
		t.Fatalf("players = %d", len(players))
	}
	p0 := players[0].(map[string]any)
	if c, _ := p0["credits"].(float64); c <= 0 {
		t.Fatalf("seat0 credits = %v（应可见己方资金）", p0["credits"])
	}
	p1 := players[1].(map[string]any)
	if c, _ := p1["credits"].(float64); c != 0 {
		t.Fatalf("seat1 credits = %v（敌方应剥离）", p1["credits"])
	}
	if _, has := view["fog"]; !has {
		t.Fatal("快照缺 fog")
	}

	// 指令：seq 单调。
	if err := send(t, s, userA, `{"type":"command","seq":1,"action":"build","unitType":"power"}`); err != nil {
		t.Fatalf("command: %v", err)
	}
	ack := connA.last("ack")
	if ack == nil || ack["ok"] != true {
		t.Fatalf("ack = %+v", ack)
	}

	// tick 推进 + 快照（偶数 tick）。
	for i := 0; i < 2; i++ {
		s.Tick()
	}
	if snap := connA.last("snapshot"); snap == nil {
		t.Fatal("tick 后无 snapshot")
	}

	// leave → 对手胜。
	if err := send(t, s, userB, `{"type":"leave"}`); err != nil {
		t.Fatalf("leave: %v", err)
	}
	endSnap := connA.last("snapshot")
	if endSnap == nil {
		t.Fatal("结算后无终态快照")
	}
	ev, _ := endSnap["view"].(map[string]any)
	if ev == nil || ev["running"] != false {
		t.Fatalf("终态 running = %+v", ev)
	}
}

// 限流：连续 45 条消息触发 1008（__RATE__ 错误标记）。
func TestRateLimit(t *testing.T) {
	s := newTestServer(t)
	_, user := connectUser(t, s, "刷屏者")
	var lastErr error
	for i := 0; i < 45; i++ {
		lastErr = send(t, s, user, `{"type":"ping","at":1}`)
	}
	if lastErr == nil || !strings.Contains(lastErr.Error(), "RATE") {
		t.Fatalf("限流未触发: %v", lastErr)
	}
}

// 昵称校验（session 面）。
func TestSessionPost(t *testing.T) {
	s := newTestServer(t)
	out, cookie, err := s.SessionPost("", "http://x", "1.2.3.4:1", []byte(`{"name":"测试者"}`))
	if err != nil || out == nil {
		t.Fatalf("session post: %v", err)
	}
	if !strings.Contains(cookie, "gp_guest=") {
		t.Fatalf("cookie = %q", cookie)
	}
	// 改名。
	out2, _, err := s.SessionPost(cookie, "http://x", "1.2.3.4:1", []byte(`{"name":"新名"}`))
	if err != nil || out2 == nil {
		t.Fatalf("rename: %v", err)
	}
	// 非法名。
	if _, _, err := s.SessionPost(cookie, "http://x", "1.2.3.4:1", []byte(`{"name":""}`)); err == nil {
		t.Fatal("空名应报错")
	}
	// find 回读。
	got := s.SessionGet(cookie)
	if got["user"] == nil {
		t.Fatal("cookie 回读失败")
	}
}
