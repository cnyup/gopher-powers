package game

import "testing"

// Wave 1f RED：update 主循环驱动经济（油井收入）。
// 源语义：p.credits += oil 数 × 11 × dt（game.js update L495）。
func TestUpdateOilIncome(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	start := g.Players[0].Credits
	// 无油井开局：credits 不涨（只掉电力无关）。
	g.Update(0.05)
	if g.Players[0].Credits != start {
		t.Fatalf("无油井时 credits 变化: %v → %v", start, g.Players[0].Credits)
	}
	// 给 side0 一口油井：每 tick +11*0.05=0.55。
	g.Oil[0].Owner = 0
	g.Update(0.05)
	want := 11.0
	want *= 0.05 // 运行时乘法，对齐 IEEE754（常量折叠会得 0.55 尾差）
	if got := g.Players[0].Credits - start; got != want {
		t.Fatalf("油井收入 = %v, want %v", got, want)
	}
}

func rt(n int) float64 { s := 0.0; for i := 0; i < n; i++ { s += 11 * 0.05 }; return s }

// update 被 paused 抑制。
func TestUpdatePaused(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	g.Paused = true
	g.Time = 0
	g.Update(0.05)
	if g.Time != 0 {
		t.Fatal("paused 时不应推进时间")
	}
}

// dt 钳制 [0, 0.05]（game.js L481）。
func TestUpdateDtClamp(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	g.Update(10) // 应被钳到 0.05
	if g.Time != 0.05 {
		t.Fatalf("Time = %v, want 0.05（钳制）", g.Time)
	}
}

// 信标控制分（control 模式）。
func TestUpdateControlScore(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{VictoryMode: "control"})
	g.Beacons[0].Owner = 0
	start := g.Players[0].ControlScore
	g.Update(0.05)
	if g.Players[0].ControlScore <= start {
		t.Fatal("control 模式信标应涨分")
	}
	if g.Players[1].ControlScore != 0 {
		t.Fatal("敌方无信标不应涨分")
	}
}

// 技能冷却推进。
func TestUpdateAbilityCooldown(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	g.Players[0].AbilityCooldown = 5
	g.Update(0.05)
	if g.Players[0].AbilityCooldown >= 5 {
		t.Fatal("冷却应随时间递减")
	}
}
