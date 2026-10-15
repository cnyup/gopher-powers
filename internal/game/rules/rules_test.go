package rules

import (
	"testing"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// 对齐 tactical-rules.js:4-7：facing = cos(atan2(sy-ty,sx-tx) - target.angle)。
// 源在目标正西（sx<tx，目标朝东 angle=0）→ angle=π → 后部 1.18。
func TestArmorFacing(t *testing.T) {
	if f := ArmorFacing(0, 0, 100, 0, 0); f.Multiplier != 1.18 || f.Name != "后部装甲" {
		t.Errorf("后部判定错: %+v", f)
	}
	// 源在目标正东（sx>tx，目标朝东）→ angle=0 → 正面 0.82。
	if f := ArmorFacing(200, 0, 100, 0, 0); f.Multiplier != .82 || f.Name != "正面装甲" {
		t.Errorf("正面判定错: %+v", f)
	}
	// 源在目标正北 → angle=π/2 → 侧面 1。
	if f := ArmorFacing(100, 200, 100, 100, 0); f.Multiplier != 1 || f.Name != "侧面装甲" {
		t.Errorf("侧面判定错: %+v", f)
	}
}

// 对齐 tactical-rules.js:13-17 战术伤害因子。
func TestTacticalDamage(t *testing.T) {
	m := gpdata.MAPS["meridian"] // 带 cover
	covered := CombatTarget{Kind: "unit", Type: "rifle", X: 1000, Y: 800} // 落在 cover[0] (930-1110, 760-920)
	if !InCover(m, covered) {
		t.Fatal("rifle 应在掩体内")
	}
	// 普通源打掩体步兵 → 0.72。
	if f := TacticalDamage(m, "tank", 0, 0, covered, 0); f != .72 {
		t.Errorf("掩体减伤 = %v, want 0.72", f)
	}
	// 例外源不受掩体影响。
	for _, src := range []string{"rocket", "strike", "navalStrike", "bomber", "loiterer", "railgun"} {
		if f := TacticalDamage(m, src, 0, 0, covered, 0); f != 1 {
			t.Errorf("%s 打掩体 = %v, want 1", src, f)
		}
	}
	// tank 打 tank（非掩体）：源 (0,0) 在目标 (100,0) 正西 → 后部 1.18。
	tankTarget := CombatTarget{Kind: "unit", Type: "tank", X: 100, Y: 0}
	if f := TacticalDamage(m, "tank", 0, 0, tankTarget, 0); f != 1.18 {
		t.Errorf("tank 后部 = %v, want 1.18", f)
	}
	// 非 armor 目标不吃朝向。
	rifleTarget := CombatTarget{Kind: "unit", Type: "rifle", X: 5000, Y: 5000}
	if f := TacticalDamage(m, "tank", 0, 0, rifleTarget, 0); f != 1 {
		t.Errorf("tank 打 rifle = %v, want 1", f)
	}
}

// 对齐 tactical-rules.js:19-23 子午天气周期。
func TestWeatherState(t *testing.T) {
	meridian := gpdata.MAPS["meridian"]
	valley := gpdata.MAPS["valley"]
	if w := WeatherState(valley, 999); w.Phase != "clear" || w.Remaining != 0 {
		t.Errorf("非未来图应恒 clear/0: %+v", w)
	}
	cases := []struct {
		t         float64
		phase     string
		remaining float64
	}{
		{0, "clear", 80}, {80, "warning", 20},
		{100, "storm", 25},
		{125, "clear", 135}, {180, "clear", 80},
	}
	for _, c := range cases {
		w := WeatherState(meridian, c.t)
		if w.Phase != c.phase || w.Remaining != c.remaining {
			t.Errorf("t=%v → %+v, want %s/%v", c.t, w, c.phase, c.remaining)
		}
	}
}

// 对齐 tactical-rules.js:25-28 信号掩护。
func TestHasSignalCover(t *testing.T) {
	relays := []RelayView{{X: 100, Y: 100, StunUntil: 5}}
	beacons := []BeaconView{{X: 1000, Y: 1000, Owner: 0}}
	if !HasSignalCover(10, relays, beacons, 0, 200, 100) {
		t.Error("relay 100 范围内应受保护")
	}
	if HasSignalCover(4, relays, beacons, 0, 200, 100) {
		t.Error("被眩晕的 relay 不应保护（time=4 < stunUntil=5）")
	}
	// 敌方 owner=1：无己方 relay 列表传入时仅信标可保护（beacons 里 owner=0 的不保护）。
	enemyRelays := []RelayView{}
	if HasSignalCover(10, enemyRelays, beacons, 1, 200, 100) {
		t.Error("敌方无 relay 无信标不应受保护")
	}
	if !HasSignalCover(10, relays, beacons, 0, 1200, 1000) {
		t.Error("信标 330 内应受保护")
	}
}
