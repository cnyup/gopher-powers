package server

import (
	"testing"

	"github.com/cnyup/gopher-powers/internal/game"
	"github.com/cnyup/gopher-powers/internal/store"
)

// 单机模式：create(solo:true) → 1 号席位是 AI（无 user、恒在线、ready）
// → 房主直接 start 即可开战，模拟核心的 IsAI 生效。
func TestSoloRoomAI(t *testing.T) {
	s, u := newTestServerUser(t)

	if err := send(t, s, u, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation","solo":true}`); err != nil {
		t.Fatalf("create solo: %v", err)
	}
	r := s.userRoom(u.ID)
	if r == nil {
		t.Fatal("建房后应有房间")
	}
	if r.members[1] == nil {
		t.Fatal("单机模式应自动填 1 号席位")
	}
	if !r.members[1].isAI() {
		t.Fatalf("1 号席位应为 AI，实际 controlType=%q", r.members[1].controlType)
	}
	if r.members[1].user != nil {
		t.Fatal("AI 席位不应有 user")
	}
	if !r.members[1].ready {
		t.Fatal("AI 席位应默认 ready")
	}
	if r.members[1].faction == r.members[0].faction {
		t.Fatalf("AI 阵营应与玩家不同：%q vs %q", r.members[1].faction, r.members[0].faction)
	}

	// 快照不得因 AI 席位无 user 而 panic，且 connected=true。
	pr := s.publicRoom(r)
	members, _ := pr["members"].([]any)
	if len(members) < 2 || members[1] == nil {
		t.Fatalf("成员快照应含 2 席：%v", members)
	}
	m1 := members[1].(map[string]any)
	if m1["id"] != "" {
		t.Fatalf("AI 席位 id 应为空串，实际 %v", m1["id"])
	}
	if m1["connected"] != true {
		t.Fatalf("AI 席位应恒在线，实际 %v", m1["connected"])
	}
	if m1["controlType"] != "ai" {
		t.Fatalf("快照 controlType 应为 ai，实际 %v", m1["controlType"])
	}
}

// 单机开局：Multiplayer=false，模拟核心把 1 号席位认作 AI。
func TestSoloStartMarksAISide(t *testing.T) {
	s, u := newTestServerUser(t)
	if err := send(t, s, u, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation","solo":true}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := send(t, s, u, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if err := send(t, s, u, `{"type":"start"}`); err != nil {
		t.Fatalf("solo 开局应允许（AI 席位恒在线且 ready）：%v", err)
	}
	r := s.userRoom(u.ID)
	if r == nil || r.game == nil {
		t.Fatal("开局后应有对局")
	}
	if r.game.Multiplayer {
		t.Fatal("单机模式 Multiplayer 应为 false")
	}
	if !r.game.IsAI(1) {
		t.Fatal("1 号席位应被判为 AI")
	}
	if r.game.IsAI(0) {
		t.Fatal("0 号席位是玩家，不应判为 AI")
	}
}

// 联机房不受影响：无 solo 时不自动补席位，且 Multiplayer 仍为 true。
func TestMultiplayerRoomUnchanged(t *testing.T) {
	s, u := newTestServerUser(t)
	if err := send(t, s, u, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation"}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	r := s.userRoom(u.ID)
	if r.members[1] != nil {
		t.Fatal("联机房不应自动填 1 号席位")
	}
	if solo, _ := r.config["solo"].(bool); solo {
		t.Fatal("联机房 solo 应为 false")
	}
}

// 单机对局结算：AI 席位没有 user，end() 不得解引用 nil（曾触发 SIGSEGV 崩溃进程）。
// 红tests：结算路径把 r.members[1].user（AI 席位为 nil）直接交给 Settle。
func TestSoloEndWithoutAIUser(t *testing.T) {
	s, u := newTestServerUser(t)
	if err := send(t, s, u, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation","solo":true}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := send(t, s, u, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if err := send(t, s, u, `{"type":"start"}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	r := s.userRoom(u.ID)
	if r == nil || r.game == nil {
		t.Fatal("开局后应有对局")
	}
	// 结算：玩家认输（对手 1 号席位即 AI）。
	// 用 end() 直调模拟「掉线宽限结束 / 认输」两条真路径共同的下游。
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("单机结算 panic（AI 席位 user 为 nil）：%v", p)
		}
	}()
	s.end(r, 1, "测试结算")

	if !r.settled {
		t.Fatal("end 后 settled 应为 true")
	}
	if r.game.Running {
		t.Fatal("end 后 game.Running 应为 false")
	}
}

// 单机对局不写战绩表：AI 席位无 user，matches 是双 human 语义。
// 对齐源项目——单机存档在浏览器 localStorage，不落 guests.sqlite（重写方案 §1.3）。
func TestSoloEndRecordsNothing(t *testing.T) {
	s := newTestServer(t)
	token, u, err := s.Accounts().Create("单机玩家")
	if err != nil {
		t.Fatal(err)
	}
	s.OnConnect(u.ID, u, &fakeConn{})
	if err := send(t, s, u, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation","solo":true}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := send(t, s, u, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if err := send(t, s, u, `{"type":"start"}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	r := s.userRoom(u.ID)
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("单机结算 panic：%v", p)
			}
		}()
		s.end(r, 0, "测试结算")
	}()
	got := s.Accounts().Find("gp_guest=" + token)
	if got == nil {
		t.Fatal("结算后应能读回玩家")
	}
	if got.Wins != 0 || got.Losses != 0 || got.Draws != 0 {
		t.Fatalf("单机局不应计入战绩，实际 胜%d 负%d 平%d", got.Wins, got.Losses, got.Draws)
	}
}

// 联机对局照常结算：两个真实 user 的胜负仍写入（守卫不得误伤正常路径）。
func TestMultiplayerEndStillSettles(t *testing.T) {
	s := newTestServer(t)
	tokA, uA, err := s.Accounts().Create("甲")
	if err != nil {
		t.Fatal(err)
	}
	connA := &fakeConn{}
	s.OnConnect(uA.ID, uA, connA)
	tokB, uB, err := s.Accounts().Create("乙")
	if err != nil {
		t.Fatal(err)
	}
	connB := &fakeConn{}
	s.OnConnect(uB.ID, uB, connB)

	if err := send(t, s, uA, `{"type":"create","mapId":"valley","faction":"china","victoryMode":"annihilation"}`); err != nil {
		t.Fatalf("create: %v", err)
	}
	code, _ := connA.last("room")["room"].(map[string]any)["code"].(string)
	if err := send(t, s, uB, `{"type":"join","code":"`+code+`","faction":"russia"}`); err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := send(t, s, uA, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready A: %v", err)
	}
	if err := send(t, s, uB, `{"type":"ready","ready":true}`); err != nil {
		t.Fatalf("ready B: %v", err)
	}
	if err := send(t, s, uA, `{"type":"start"}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	r := s.userRoom(uA.ID)
	if r == nil || r.game == nil {
		t.Fatal("开局后应有对局")
	}
	s.end(r, 0, "测试结算")
	a := s.Accounts().Find("gp_guest=" + tokA)
	b := s.Accounts().Find("gp_guest=" + tokB)
	if a == nil || b == nil {
		t.Fatal("结算后应能读回两名玩家")
	}
	if a.Wins != 1 || b.Losses != 1 {
		t.Fatalf("联机结算应为 甲胜1/乙负1，实际 甲胜%d负%d / 乙胜%d负%d",
			a.Wins, a.Losses, b.Wins, b.Losses)
	}
}

// newTestServerUser 复用既有 newTestServer + connectUser（已处理 conn/hello）。
func newTestServerUser(t *testing.T) (*Server, *store.User) {
	t.Helper()
	s := newTestServer(t)
	_, u := connectUser(t, s, "单机玩家")
	return s, u
}

var _ = game.ValidFaction
