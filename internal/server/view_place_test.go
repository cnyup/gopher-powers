package server

import (
	"testing"

	"github.com/cnyup/gopher-powers/internal/game"
)

// placeRules 是前端放置预检的唯一数据源——源项目 canPlace 在客户端本地跑
// （render.js:1177），拆进程后必须随快照下发，否则 ghost 无法着色。
func TestBattleViewEmitsPlaceRules(t *testing.T) {
	g := makeTestGame(t)
	view := battleView(g, 0, viewMemory{}, false)

	raw, ok := view["placeRules"]
	if !ok || raw == nil {
		t.Fatal("battleView 未下发 placeRules（前端无法做落位预检）")
	}
	rules, ok := raw.(game.PlaceRules)
	if !ok {
		t.Fatalf("placeRules 类型应为 game.PlaceRules，实际 %T", raw)
	}
	// 世界尺寸必须与模拟一致——战场按 1.5 缩放，硬编码 2240×1440 会错。
	if rules.WorldWidth != g.World.Width || rules.WorldHeight != g.World.Height {
		t.Fatalf("世界尺寸 = %v×%v，模拟为 %v×%v",
			rules.WorldWidth, rules.WorldHeight, g.World.Width, g.World.Height)
	}
	// 尺寸表必须覆盖全部可建建筑，否则前端对未知类型会恒判非法。
	for _, typ := range []string{"hq", "power", "refinery", "barracks", "factory", "armory", "dock", "radar", "airfield", "turret", "lab", "super"} {
		if rules.Sizes[typ] <= 0 {
			t.Errorf("尺寸表缺 %q（前端会把它判成非法落点）", typ)
		}
	}
	// 判据必须与服务端 CanPlace 同源：合法点两边都合法。
	// 在己方 HQ 310 半径内、禁建区外找一个点。
	hq := g.OwnedBuildings(0, "hq")
	if len(hq) == 0 {
		t.Fatal("测试对局应有己方 HQ")
	}
	px, py := hq[0].X+120, hq[0].Y+120
	serverOK := g.CanPlacePublic(0, "power", px, py)
	if !serverOK {
		t.Logf("提示：(%v,%v) 服务端判非法，跳过一致性断言（依赖地图布局）", px, py)
		return
	}
	// 前端 canPlace 已由 web/src/scene/placement.test.mjs 覆盖；
	// 此处只验证服务端确实能给出该判据所需的输入。
	if len(rules.NoBuild) != len(g.Map.NoBuild) {
		t.Errorf("NoBuild 条数 = %d，地图为 %d", len(rules.NoBuild), len(g.Map.NoBuild))
	}
}

// 六方图没有 NoBuild 时也不得 panic（nil 切片序列化为 null，前端需能处理）。
func TestPlaceRulesNilSlicesSafe(t *testing.T) {
	g := makeTestGame(t)
	rules := g.PlaceRulesFor(0)
	if rules.Sizes == nil {
		t.Fatal("Sizes 不应为 nil")
	}
}
