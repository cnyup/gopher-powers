package game

import (
	"math"
	"testing"
)

// Wave 1f-3 RED：战斗系。源锚点 game.js L767-795 canAttack、L841-878 unitDamage、L881-915 fire。

// canAttack：防空车只能打空中目标。
func TestCanAttackAA(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	aa := &Unit{Kind: "unit", Owner: 0, Type: "aa", HP: 100}
	fighter := &Unit{Kind: "unit", Owner: 1, Type: "fighter", HP: 100}
	tank := &Unit{Kind: "unit", Owner: 1, Type: "tank", HP: 100}
	if !g.CanAttack(aa, fighter) {
		t.Fatal("aa 应能打 fighter")
	}
	if g.CanAttack(aa, tank) {
		t.Fatal("aa 不应打 tank")
	}
}

// canAttack：潜艇只吃反潜/潜艇。
func TestCanAttackSubmarine(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{MapID: "ocean"}) // 舰艇交战需水图（源码语义）
	sub := &Unit{Kind: "unit", Owner: 1, Type: "submarine", HP: 100, X: 1500, Y: 1100} // ocean 水域 x∈[1040,2480]
	tank := &Unit{Kind: "unit", Owner: 0, Type: "tank", HP: 100, X: 1500, Y: 1100}
	frigate := &Unit{Kind: "unit", Owner: 0, Type: "frigate", HP: 100, X: 1600, Y: 1100}
	enemySub := &Unit{Kind: "unit", Owner: 0, Type: "submarine", HP: 100, X: 1700, Y: 1100}
	if g.CanAttack(tank, sub) {
		t.Fatal("tank 不应打潜艇")
	}
	if !g.CanAttack(frigate, sub) {
		t.Fatal("frigate 应反潜")
	}
	if !g.CanAttack(enemySub, sub) {
		t.Fatal("潜艇互打应允许")
	}
}

// unitDamage：核心克制表抽查（源数值锚点）。
func TestUnitDamage(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	tank := &Unit{Kind: "unit", Owner: 0, Type: "tank", HP: 380, X: 0, Y: 0}
	rifle := &Unit{Kind: "unit", Owner: 1, Type: "rifle", HP: 110, X: 0, Y: 0}
	// tank 打 rifle（armor 无朝向因子因 rifle 非 armor；tacticalDamage=1）→ 38 * 1.45。
	if d := g.UnitDamage(tank, rifle); math.Abs(d-38*1.45) > 1e-9 {
		t.Fatalf("tank→rifle = %v, want 55.1", d)
	}
	// rifle 打 tank → 12 * 0.52。
	if d := g.UnitDamage(rifle, tank); math.Abs(d-12*0.52) > 1e-9 {
		t.Fatalf("rifle→tank = %v, want 6.24", d)
	}
}

// damage：命中扣血 + 死亡清理 + 中国炮塔减伤 15%。
func TestDamage(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	u := g.Units[0]
	g.Damage(u, 10, 1)
	if u.HP != u.MaxHP-10 {
		t.Fatalf("HP = %v", u.HP)
	}
	// 华夏 turret 减伤：target faction=china & type=turret → ×0.85。
	gt, _ := New("china", "russia", nil, Options{})
	turret := gt.AddBuilding(0, "turret", 500, 500)
	gt.Damage(turret, 100, 1)
	if math.Abs(turret.HP-(turret.MaxHP-85)) > 1e-9 {
		t.Fatalf("china turret 减伤失效: %v", turret.HP)
	}
}

// 击杀触发：hp<=0 → 从列表清除（update 循环过滤）+ explosion 特效。
func TestDamageKill(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	u := g.Units[0]
	nEffects := len(g.Effects)
	g.Damage(u, 99999, 1)
	if u.HP != 0 {
		t.Fatal("应归零")
	}
	if len(g.Effects) != nEffects+1 {
		t.Fatal("应产生爆炸特效")
	}
	g.Update(0.05)
	for _, x := range g.Units {
		if x == u {
			t.Fatal("死亡单位应被清除")
		}
	}
}

// closestEnemy 优先级：fighter 优先空中（priority 0.3）。
func TestClosestEnemy(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	fighter := &Unit{Kind: "unit", Owner: 0, Type: "fighter", HP: 100, X: 500, Y: 500}
	groundTarget := &Unit{Kind: "unit", Owner: 1, Type: "tank", HP: 380, X: 550, Y: 500}
	airTarget := &Unit{Kind: "unit", Owner: 1, Type: "drone", HP: 130, X: 520, Y: 500} // 距离 20×0.3=6 < ground 50×1
	g.Units = append(g.Units, fighter, groundTarget, airTarget)
	g.UpdateFog()
	// 需可见才能索敌：把 side0 视野拉满（借卫星）。
	g.Players[0].SatelliteUntil = g.Time + 100
	g.UpdateFog()
	got := g.ClosestEnemy(fighter, 400, 0)
	if got != airTarget && got != groundTarget {
		t.Fatal("应索到目标")
	}
	// 空中优先：air 距离 20×0.3=6 < ground 50×1 → fighter 应选 airTarget。
	if got != airTarget {
		t.Fatalf("fighter 应优先空中目标，得 %v", got)
	}
}
