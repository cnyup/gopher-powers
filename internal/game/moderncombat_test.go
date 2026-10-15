package game

import (
	"math"
	"testing"
)

// Wave 1d：弹道飞行与拦截。
// 对齐 modern-combat.js launchProjectile/updateProjectiles 数值锚点。

// torpedo 出管：沿艇首方向偏移 6.2*8，角度 = 艇身角。
func TestTorpedoLaunch(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{MapID: "ocean"})
	sub := g.AddUnit(0, "submarine", 1500, 1100, false)
	sub.Angle = 0 // 朝东
	enemy := g.AddUnit(1, "patrol", 1800, 1100, false)
	if !g.launchProjectile(sub, enemy, 110, "submarine") {
		t.Fatal("应成功发射鱼雷")
	}
	p := g.Projectiles[len(g.Projectiles)-1]
	if p.Kind != "torpedo" {
		t.Fatalf("kind = %s", p.Kind)
	}
	if math.Abs(p.X-(1500+6.2*8)) > 1e-9 {
		t.Fatalf("torpedo 起点 X = %v（应偏移 49.6）", p.X)
	}
	if p.Angle != 0 {
		t.Fatalf("torpedo 初始角 = %v（应继承艇身角）", p.Angle)
	}
	if sub.ExposedUntil <= g.Time {
		t.Fatal("发射后应暴露 4 秒")
	}
}

// 拦截：aa 对来袭 missile 造成 damage*1.8。
func TestInterceptMissile(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	aa := g.AddUnit(0, "aa", 500, 500, false)
	src := g.AddUnit(1, "loiterer", 700, 500, false) // 距 aa 200 ≤ range 250
	tgt := g.AddUnit(0, "tank", 520, 500, false)
	if !g.launchProjectile(src, tgt, 95, "loiterer") {
		t.Fatal("发射失败")
	}
	p := g.Projectiles[len(g.Projectiles)-1]
	if p.Kind != "loitering" {
		t.Fatalf("kind = %s", p.Kind)
	}
	hp0 := p.HP
	aa.FireTimer = 0
	g.Players[0].SatelliteUntil = g.Time + 100
	g.UpdateFog()
	g.updateProjectilesFull(0.05)
	if p.HP >= hp0 {
		t.Fatalf("aa 应拦截（hp %v → %v，期望 -%v）", hp0, p.HP, 22*1.8)
	}
}

// laser 只打 wing（不含 rocket/torpedo/bomb 与 missile）。
func TestLaserInterceptRules(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	laser := g.AddUnit(0, "laser", 500, 500, false)
	laser.FireTimer = 0
	src := g.AddUnit(1, "rocket", 560, 500, false)
	tgt := g.AddUnit(0, "tank", 510, 500, false)
	g.launchProjectile(src, tgt, 85, "rocket")
	p := g.Projectiles[len(g.Projectiles)-1]
	hp0 := p.HP
	g.updateProjectilesFull(0.05)
	if p.HP != hp0 {
		t.Fatal("laser 不应拦截 rocket（源码 kind 过滤）")
	}
}

// wing 返航：命中后 Returning=true 而非销毁。
func TestWingReturning(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{MapID: "ocean"})
	carrier := g.AddUnit(0, "carrier", 1500, 1100, false)
	w := carrier.Wing
	if w == nil || *w <= 0 {
		t.Fatal("carrier 应有初始 wing")
	}
	enemy := g.AddUnit(1, "patrol", 1600, 1100, false)
	if !g.launchProjectile(carrier, enemy, 95, "carrier") {
		t.Fatal("舰载机编组应可出动")
	}
	p := g.Projectiles[len(g.Projectiles)-1]
	if p.Kind != "wing" {
		t.Fatalf("kind = %s", p.Kind)
	}
	p.X, p.Y = p.ToX-1, p.ToY // 逼近命中点
	g.projectileHit(p, enemy)
	if !p.Returning {
		t.Fatal("wing 命中后应返航")
	}
	if p.Finished {
		t.Fatal("返航中不应 finished")
	}
}

// 电子战：storm 无中继覆盖时空中单位被压制（jam 0.3s）。
func TestStormJam(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{MapID: "meridian"})
	// 推到 storm 相位（time ∈ [100,125)）。
	g.Time = 110
	jet := g.AddUnit(0, "fighter", 500, 500, false)
	g.updateElectronicWarfareFull(0.05)
	if jet.JammedUntil <= g.Time {
		t.Fatal("storm 中无中继的战机应被压制")
	}
	// 中继覆盖范围内不受压制。
	relay := g.AddUnit(0, "relay", 510, 500, false)
	_ = relay // relay 存在即提供 260 信号覆盖（hasSignalCover 扫描 ActiveUnits）
	jet.JammedUntil = 0
	g.updateElectronicWarfareFull(0.05)
	if jet.JammedUntil > g.Time {
		t.Fatal("中继 260 范围内不应被压制")
	}
}

// 激光过热：+34/发，≥95 过热，<28 恢复。
func TestLaserHeat(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	laser := g.AddUnit(0, "laser", 500, 500, false)
	g.laserHeat(laser)
	if laser.Heat != 34 {
		t.Fatalf("heat = %v, want 34", laser.Heat)
	}
	laser.Heat = 96
	g.laserHeat(laser)
	if !laser.Overheated {
		t.Fatal("≥95 应过热")
	}
	laser.Heat = 27
	g.updateElectronicWarfareFull(0.05)
	if laser.Overheated {
		t.Fatal("<28 应解除过热")
	}
}
