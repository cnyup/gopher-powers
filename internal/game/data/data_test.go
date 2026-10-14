package data

import (
	"slices"
	"testing"
)

// 数据规模快照：5 阵营 / 8 地图 / 33 单位 / 12 建筑 / 3 胜利模式（源项目 node -e 实测锚点）。
func TestScale(t *testing.T) {
	if len(FACTIONS) != 5 {
		t.Fatalf("FACTIONS = %d, want 5", len(FACTIONS))
	}
	if len(MAPS) != 8 {
		t.Fatalf("MAPS = %d, want 8", len(MAPS))
	}
	if len(UNITS) != 33 {
		t.Fatalf("UNITS = %d, want 33", len(UNITS))
	}
	if len(BUILDINGS) != 12 {
		t.Fatalf("BUILDINGS = %d, want 12", len(BUILDINGS))
	}
	if len(VICTORY_MODES) != 3 {
		t.Fatalf("VICTORY_MODES = %d, want 3", len(VICTORY_MODES))
	}
	if len(AI_DIFFICULTIES) != 3 {
		t.Fatalf("AI_DIFFICULTIES = %d, want 3", len(AI_DIFFICULTIES))
	}
}

// 弹药循环字段对齐 data.js:186-189。
func TestAmmoFields(t *testing.T) {
	want := map[string][3]float64{
		"rifle": {18, 1, 9}, "tank": {12, 6, 12}, "aa": {16, 4, 10}, "elite": {12, 6, 12},
		"drone": {8, 4, 8}, "patrol": {24, 3, 12}, "frigate": {16, 5, 12},
	}
	for typ, w := range want {
		u := UNITS[typ]
		if u.Ammo != w[0] || u.AmmoCost != w[1] || u.RearmTime != w[2] {
			t.Fatalf("%s ammo=%v/%v rearm=%v, want %v", typ, u.Ammo, u.AmmoCost, u.RearmTime, w)
		}
	}
	if UNITS["rifle"].Desc == "经济、可靠的基础作战单位。" {
		t.Fatal("rifle desc 未追加弹药说明")
	}
	// carrier 覆写对齐 data.js:194。
	if UNITS["carrier"].Capacity != 3 {
		t.Fatalf("carrier capacity = %v, want 3", UNITS["carrier"].Capacity)
	}
}

// UNIT_ORDER 顺序与计数对齐 data.js:192-196。
func TestUnitOrder(t *testing.T) {
	want := []string{"rifle", "engineer", "scout", "tank", "apc", "supply", "harvester", "drone", "ghost", "aa",
		"loiterer", "jammer", "laser", "rocket", "elite", "fighter", "strike", "patrol", "frigate",
		"destroyer", "carrier", "submarine", "railgun", "aegis", "relay", "landing", "bomber", "airlift",
		"navalFighter", "navalStrike", "ewPlane"}
	if !slices.Equal(UNIT_ORDER, want) {
		t.Fatalf("UNIT_ORDER 不匹配: %v", UNIT_ORDER)
	}
}

// 阵营修正对齐 unit-specs.js。注：IEEE754 运行时 380*1.15 = 436.99999999999994
// （JS 与 Go 逐位一致）。Go 未类型化常量会算出精确 437，故期望值必须走运行时乘法。
func rtMul(a, b float64) float64 { return a * b }

func TestFactionModifiers(t *testing.T) {
	if got := UnitCostFor("asia", "drone"); got != 221 {
		t.Fatalf("UnitCostFor(asia,drone) = %v, want 221", got)
	}
	if got := UnitHealthFor("nato", "tank"); got != rtMul(380, 1.15) {
		t.Fatalf("UnitHealthFor(nato,tank) = %v, want %v", got, rtMul(380, 1.15))
	}
	if got := UnitHealthFor("nato", "elite"); got != 580 {
		t.Fatalf("UnitHealthFor(nato,elite) = %v, want 580", got)
	}
	if got := UnitHealthFor("asia", "elite"); got != 250 {
		t.Fatalf("UnitHealthFor(asia,elite) = %v, want 250", got)
	}
	if got := UnitHealthFor("china", "aa"); got != rtMul(230, 1.15) {
		t.Fatalf("UnitHealthFor(china,aa) = %v, want %v", got, rtMul(230, 1.15))
	}
	if got := UnitCostFor("china", "tank"); got != 480 {
		t.Fatalf("UnitCostFor(china,tank) = %v, want 480", got)
	}
}

// 六方出生点几何对齐 six-player-map.js（半径 1920、首席朝上）。
func TestSixSpawns(t *testing.T) {
	first := SixSpawns[0]
	if first.X != 2880 || first.Y != 960 {
		t.Fatalf("spawn0 = %v,%v want 2880,960", first.X, first.Y)
	}
	dx := SixSpawns[1].X - SixSpawns[0].X
	dy := SixSpawns[1].Y - SixSpawns[0].Y
	d := dx*dx + dy*dy
	if d < 1920*1920*0.999 || d > 1920*1920*1.001 {
		t.Fatalf("相邻席位弦长平方 = %v, want ≈%d", d, 1920*1920)
	}
	if len(SixSpawns) != 6 {
		t.Fatalf("SixSpawns = %d, want 6", len(SixSpawns))
	}
}

// 六方地图资源计数：4×6 模板矿 + 6 中央矿 = 30 矿 / 6 油 / 6 信标。
func TestSixFrontMap(t *testing.T) {
	m := MAPS["sixfront"]
	if m.MaxPlayers != 6 || m.Name != "六域会战" {
		t.Fatalf("sixfront 元数据错误: %+v", m)
	}
	if len(m.Ore) != 30 {
		t.Fatalf("sixfront ore = %d, want 30", len(m.Ore))
	}
	if len(m.Oil) != 6 || len(m.Beacons) != 6 {
		t.Fatalf("sixfront oil/beacons = %d/%d, want 6/6", len(m.Oil), len(m.Beacons))
	}
	if m.World.Width != 5760 || m.World.Height != 5760 {
		t.Fatal("sixfront world 尺寸错误")
	}
	if len(ResourceLayout(m)) != 18 {
		t.Fatalf("sixfront resources = %d, want 18", len(ResourceLayout(m)))
	}
}

// 战场缩放对齐 battlefield-expansion.js battlefieldMap。
func TestBattlefieldMapScale(t *testing.T) {
	scaled, err := BattlefieldMap("valley", 1.5)
	if err != nil {
		t.Fatal(err)
	}
	if scaled.World.Width != 2240*1.5 || scaled.World.Height != 1440*1.5 {
		t.Fatalf("缩放后世界 = %v×%v", scaled.World.Width, scaled.World.Height)
	}
	if scaled.Name != "灰谷地带 · 纵深" {
		t.Fatalf("缩放后名称 = %v", scaled.Name)
	}
	if scaled.Ore[0].X != 330*1.5 {
		t.Fatal("矿点未缩放")
	}
	identical, _ := BattlefieldMap("valley", 1)
	if identical.Name != "灰谷地带" {
		t.Fatal("scale=1 应原样返回")
	}
	six, _ := BattlefieldMap("sixfront", 1.5)
	if six.Name != "六域会战" {
		t.Fatal("六方图不受 scale 影响")
	}
	if _, err := BattlefieldMap("valley", 2); err == nil {
		t.Fatal("非法 scale 应报错")
	}
	// 双人图资源点：3 类 × 2 侧 = 6。
	valley := MAPS["valley"]
	if got := len(ResourceLayout(valley)); got != 6 {
		t.Fatalf("valley resources = %d, want 6", got)
	}
}

// SupportsMap 对齐 data.js:197。
func TestSupportsMap(t *testing.T) {
	cases := []struct {
		required, mapID string
		want            bool
	}{
		{"", "valley", true},
		{"ocean", "archipelago", true},
		{"ocean", "ocean", true},
		{"meridian", "valley", false},
		{"radar", "valley", false},
	}
	for _, c := range cases {
		if got := SupportsMap(c.required, c.mapID); got != c.want {
			t.Fatalf("SupportsMap(%q,%q) = %v, want %v", c.required, c.mapID, got, c.want)
		}
	}
}

// LegacyBattleID 对齐 battle-identity.js FNV 双哈希。
func TestLegacyBattleID(t *testing.T) {
	id := LegacyBattleID([]rune("test-save"))
	if !ValidBattleID(id) {
		t.Fatalf("id 不合 battleId 规范: %v", id)
	}
	if LegacyBattleID([]rune("test-save")) != id {
		t.Fatal("同输入应产生相同指纹")
	}
	if LegacyBattleID([]rune("test-save2")) == id {
		t.Fatal("不同输入应产生不同指纹")
	}
	if !ValidBattleID("abcABC123_-xyz") || len("abcABC123_-xyz") < 12 {
		t.Fatal("合法 id 误拒")
	}
	if ValidBattleID("short") {
		t.Fatal("过短 id 误收")
	}
}

// HasTag 委托验证。
func TestHasTag(t *testing.T) {
	if !HasTag("tank", "armor") || HasTag("tank", "air") {
		t.Fatal("HasTag 判定错误")
	}
	if !HasTag("submarine", "stealth") {
		t.Fatal("submarine 应带 stealth")
	}
}
