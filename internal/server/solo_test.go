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

// newTestServerUser 复用既有 newTestServer + connectUser（已处理 conn/hello）。
func newTestServerUser(t *testing.T) (*Server, *store.User) {
	t.Helper()
	s := newTestServer(t)
	_, u := connectUser(t, s, "单机玩家")
	return s, u
}

var _ = game.ValidFaction
