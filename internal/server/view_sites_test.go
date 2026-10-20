package server

import (
	"testing"

	"github.com/cnyup/gopher-powers/internal/game"
)

// makeTestGame 构造一个最小对局：双方各一个 HQ，矿/油/信标/资源点各一。
func makeTestGame(t *testing.T) *game.Game {
	t.Helper()
	g, err := game.New("china", "russia", nil, game.Options{MapID: "meridian"})
	if err != nil {
		t.Fatalf("game.New: %v", err)
	}
	return g
}

func TestBattleViewSitesBasic(t *testing.T) {
	g := makeTestGame(t)
	if len(g.Ore) == 0 {
		t.Fatal("测试地图应有矿点")
	}
	view := battleView(g, 0, viewMemory{}, false)
	ore, _ := view["ore"].([]any)
	if len(ore) == 0 {
		t.Fatal("ore 数组为空")
	}
	// 记忆 stub 修通后：元素必须是完整对象（不再是 null）。
	for i, e := range ore {
		if e == nil {
			t.Fatalf("ore[%d] 为 nil——site 记忆 stub 未实现", i)
		}
		m, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("ore[%d] 非 map：%T", i, e)
		}
		if m["id"] == nil || m["x"] == nil {
			t.Fatalf("ore[%d] 缺 id/x 字段：%v", i, m)
		}
	}
}

func TestBattleViewSitesMemoryPersists(t *testing.T) {
	// 已看过的 site 进入记忆；随后被迷雾遮蔽，快照仍返回记忆中的条目（而非 null）。
	g := makeTestGame(t)
	memory := viewMemory{}
	view1 := battleView(g, 0, memory, false)
	ore1, _ := view1["ore"].([]any)
	if len(ore1) == 0 || ore1[0] == nil {
		t.Fatal("初次快照 ore[0] 应非 nil")
	}
	// 直接清空可见性：把 side 0 的 fog 全部置为不可见未探索。
	fog := g.Fogs[0]
	for i := range fog.Visible {
		fog.Visible[i] = false
		fog.Explored[i] = false
	}
	view2 := battleView(g, 0, memory, false)
	ore2, _ := view2["ore"].([]any)
	if len(ore2) != len(ore1) {
		t.Fatalf("记忆生效后 ore 长度应不变：%d → %d", len(ore1), len(ore2))
	}
	if ore2[0] == nil {
		t.Fatal("迷雾遮蔽后 ore[0] 应返回记忆条目，实际 nil——site 记忆未生效")
	}
}

func TestBattleViewSitesFogStripsOwner(t *testing.T) {
	// 敌方（side 1）占领的矿被迷雾遮蔽：owner 应剥为 null、amount 露 max。
	// 回归锚点：曾用原始 owner 重新映射写回，把剥离覆盖成真实归属。
	g := makeTestGame(t)
	memory := viewMemory{}
	if len(g.Ore) == 0 {
		t.Fatal("测试地图应有矿点")
	}
	// 矿点置于 (500,500)，归敌方。
	o := g.Ore[0]
	o.Owner = 1
	o.X, o.Y = 500, 500
	// side 0 视野外（默认初始视野在基地附近）。
	fog := g.Fogs[0]
	for i := range fog.Visible {
		fog.Visible[i] = false
		fog.Explored[i] = false
	}
	view := battleView(g, 0, memory, false)
	ore, _ := view["ore"].([]any)
	var target map[string]any
	for _, e := range ore {
		if m, ok := e.(map[string]any); ok {
			target = m
		}
	}
	if target == nil {
		t.Fatal("ore 无 map 条目")
	}
	if target["owner"] != nil {
		t.Fatalf("迷雾遮蔽的敌矿 owner 应为 null，实际 %v", target["owner"])
	}
	if amt, _ := target["amount"].(float64); amt != o.Max {
		t.Fatalf("迷雾遮蔽的敌矿 amount 应为 max=%v，实际 %v", o.Max, target["amount"])
	}
}
