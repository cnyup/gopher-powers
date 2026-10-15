package game

import (
	"strings"
	"testing"
)

// 对齐 game.test.js 首测：双方以同等经济、产能和部队开局。
func TestNewEqualStart(t *testing.T) {
	g, err := New("china", "russia", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if g.Players[0].Credits != g.Players[1].Credits {
		t.Fatalf("credits %v vs %v", g.Players[0].Credits, g.Players[1].Credits)
	}
	u0, u1 := g.OwnedUnits(0, ""), g.OwnedUnits(1, "")
	if len(u0) != len(u1) {
		t.Fatalf("units %d vs %d", len(u0), len(u1))
	}
	b0, b1 := g.OwnedBuildings(0, ""), g.OwnedBuildings(1, "")
	if len(b0) != len(b1) {
		t.Fatalf("buildings %d vs %d", len(b0), len(b1))
	}
	for i := range b0 {
		if b0[i].Type != b1[i].Type {
			t.Fatalf("building %d: %s vs %s", i, b0[i].Type, b1[i].Type)
		}
	}
	if g.Players[0].PowerIn != g.Players[1].PowerIn || g.Players[0].PowerOut != g.Players[1].PowerOut {
		t.Fatalf("power %v/%v vs %v/%v", g.Players[0].PowerIn, g.Players[0].PowerOut, g.Players[1].PowerIn, g.Players[1].PowerOut)
	}
}

// 对齐 game.test.js 次测：三档难度不改变开局资源与单位属性。
func TestDifficultyNeutralStart(t *testing.T) {
	for _, diff := range []string{"recruit", "standard", "veteran"} {
		g, err := New("china", "russia", nil, Options{Difficulty: diff})
		if err != nil {
			t.Fatal(err)
		}
		if g.Players[0].Credits != g.Players[1].Credits {
			t.Fatalf("%s: credits 不等", diff)
		}
		u0, u1 := g.OwnedUnits(0, ""), g.OwnedUnits(1, "")
		if len(u0) != len(u1) {
			t.Fatalf("%s: units 不等", diff)
		}
		for i := range u0 {
			if u0[i].Type != u1[i].Type || u0[i].MaxHP != u1[i].MaxHP {
				t.Fatalf("%s: unit %d 属性漂移", diff, i)
			}
		}
	}
}

// 难度参数确实生效（waveStart/thinkInterval 差异）。
func TestDifficultyParams(t *testing.T) {
	g1, _ := New("china", "russia", nil, Options{Difficulty: "recruit"})
	g2, _ := New("china", "russia", nil, Options{Difficulty: "veteran"})
	if g1.AIWaveTimer == g2.AIWaveTimer {
		t.Fatal("不同难度的首波时间应不同")
	}
}

// 迷雾：开局后 HQ 周围可见（400 视野 > fog 网格 80）。
func TestFogVisibleAtBase(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	hq := g.OwnedBuildings(0, "hq")[0]
	if !g.IsVisibleFor(0, hq.X, hq.Y) {
		t.Fatal("HQ 位置应在己方视野内")
	}
	// 敌方角落（默认无单位处）不可见。
	if g.IsVisibleFor(0, g.World.Width-30, 30) {
		t.Fatal("敌方右上角落不应直接可见")
	}
}

// 电子经济（convoy 模式）：首班运输机 3 秒后起飞。
func TestConvoyLaunch(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	if g.EconomyMode != "convoy" {
		t.Fatalf("valley 默认应为 convoy，得 %s", g.EconomyMode)
	}
	// 直接推时间：NextAir = 3，update 由 Wave 1f 主循环补；此处验证布局函数。
	if g.LogisticsRoutes[0].NextAir != 3 {
		t.Fatalf("nextAir = %v, want 3", g.LogisticsRoutes[0].NextAir)
	}
	if g.FreightDestination(0, false) == nil {
		t.Fatal("开局应有 refinery 作为货运终点")
	}
}

// 席位配置校验对齐 constructor 抛错路径。
func TestInvalidPlayerConfigs(t *testing.T) {
	if _, err := New("china", "russia", nil, Options{Players: []PlayerConfig{{Faction: "china", ControlType: "human"}}}); err == nil {
		t.Fatal("单席配置应报错（双人图）")
	}
	if _, err := New("china", "russia", nil, Options{Players: []PlayerConfig{{Faction: "xx", ControlType: "human"}, {Faction: "russia", ControlType: "ai"}}}); err == nil {
		t.Fatal("非法阵营应报错")
	}
	if _, err := New("china", "russia", nil, Options{Players: []PlayerConfig{{Faction: "china", ControlType: "robot"}, {Faction: "russia", ControlType: "ai"}}}); err == nil {
		t.Fatal("非法控制类型应报错")
	}
}

// 战区 08 六方图：六席 + 采矿经济。
func TestSixFrontSetup(t *testing.T) {
	cfg := make([]PlayerConfig, 6)
	factions := []string{"china", "russia", "nato", "asia", "middleeast"}
	for i := range cfg {
		cfg[i] = PlayerConfig{Faction: factions[i%5], ControlType: map[bool]string{true: "human", false: "ai"}[i == 0]}
	}
	g, err := New("china", "russia", nil, Options{MapID: "sixfront", Players: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if g.PlayerCount != 6 || g.EconomyMode != "mining" {
		t.Fatalf("playerCount=%d economy=%s", g.PlayerCount, g.EconomyMode)
	}
	if len(g.Ore) != 30 {
		t.Fatalf("ore = %d, want 30", len(g.Ore))
	}
	// 每席镜像模板开局：建筑 5 座（无水无 dock）。
	for side := 0; side < 6; side++ {
		if n := len(g.OwnedBuildings(side, "")); n != 5 {
			t.Fatalf("side %d buildings = %d, want 5", side, n)
		}
	}
}

// battleId 形态。
func TestBattleIDShape(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	if !strings.HasPrefix(g.BattleID, "battle-") || len(g.BattleID) < 12 {
		t.Fatalf("battleId 形态异常: %s", g.BattleID)
	}
}
