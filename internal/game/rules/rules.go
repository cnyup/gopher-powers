// Package rules 对齐源项目 src/tactical-rules.js。
package rules

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// ArmorFacing 对齐 tactical-rules.js:4-7。
type Facing struct {
	Name       string
	Multiplier float64
}

func ArmorFacing(sx, sy, tx, ty, tAngle float64) Facing {
	angle := math.Atan2(sy-ty, sx-tx)
	facing := math.Cos(angle - tAngle)
	switch {
	case facing > .5:
		return Facing{Name: "正面装甲", Multiplier: .82}
	case facing < -.5:
		return Facing{Name: "后部装甲", Multiplier: 1.18}
	default:
		return Facing{Name: "侧面装甲", Multiplier: 1}
	}
}

// CombatTarget 战术伤害判定的最小目标视图。
type CombatTarget struct {
	Kind string // "unit" / "building"
	Type string
	X, Y float64
}

// InCover 对齐 tactical-rules.js:10-11：仅步兵可入掩体。
func InCover(m *gpdata.MapSpec, t CombatTarget) bool {
	if t.Kind != "unit" || !gpdata.HasTag(t.Type, "infantry") {
		return false
	}
	for _, zone := range m.Cover {
		if t.X >= zone.X1 && t.X <= zone.X2 && t.Y >= zone.Y1 && t.Y <= zone.Y2 {
			return true
		}
	}
	return false
}

// TacticalDamage 对齐 tactical-rules.js:13-17。
// sourceAngle 仅在 source 为 tank/railgun 攻击 armor 时使用（目标 angle 字段）。
func TacticalDamage(m *gpdata.MapSpec, sourceType string, sx, sy float64, target CombatTarget, targetAngle float64) float64 {
	factor := 1.0
	if target.Kind == "unit" && gpdata.HasTag(target.Type, "armor") &&
		(sourceType == "tank" || sourceType == "railgun") {
		factor *= ArmorFacing(sx, sy, target.X, target.Y, targetAngle).Multiplier
	}
	if InCover(m, target) {
		switch sourceType {
		case "rocket", "strike", "navalStrike", "bomber", "loiterer", "railgun":
		default:
			factor *= .72
		}
	}
	return factor
}

// Weather 对齐 tactical-rules.js:19-23 的 phase/remaining。
type Weather struct {
	Phase     string
	Remaining float64
}

// WeatherState 对齐 tactical-rules.js:19-23：子午图 180 秒周期。
func WeatherState(m *gpdata.MapSpec, time float64) Weather {
	if !m.Future {
		return Weather{Phase: "clear", Remaining: 0}
	}
	cycle := math.Mod(math.Max(0, time), 180)
	switch {
	case cycle < 80:
		return Weather{Phase: "clear", Remaining: 80 - cycle}
	case cycle < 100:
		return Weather{Phase: "warning", Remaining: 100 - cycle}
	case cycle < 125:
		return Weather{Phase: "storm", Remaining: 125 - cycle}
	default:
		return Weather{Phase: "clear", Remaining: 260 - cycle}
	}
}

// HasSignalCoverView 对齐 tactical-rules.js:25-28 的中继/信标保护判定输入。
type RelayView struct {
	X, Y, StunUntil float64
}
type BeaconView struct {
	X, Y float64
	Owner int
}

// HasSignalCover：relay 260 或己方信标 330 范围（game.time 比较）。
// relays 约定传入 owner 方的中继车列表（源码 activeUnits(owner,'relay') 已过滤）。
func HasSignalCover(time float64, relays []RelayView, beacons []BeaconView, owner int, px, py float64) bool {
	for _, r := range relays {
		if time >= r.StunUntil && math.Sqrt((r.X-px)*(r.X-px)+(r.Y-py)*(r.Y-py)) <= 260 {
			return true
		}
	}
	for _, b := range beacons {
		if b.Owner == owner && math.Sqrt((b.X-px)*(b.X-px)+(b.Y-py)*(b.Y-py)) <= 330 {
			return true
		}
	}
	return false
}
