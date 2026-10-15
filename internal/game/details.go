package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// battlefield-details 移植（game.js 引用的三件套）。

func atan2(y, x float64) float64 { return math.Atan2(y, x) }
func cos(v float64) float64      { return math.Cos(v) }
func sin(v float64) float64      { return math.Sin(v) }

// productionExit 对齐 battlefield-details.js:4-11。
func (g *Game) productionExit(b *Building, typ string) (out struct {
	X, Y, Angle           float64
	Side                  float64
	ForwardX, ForwardY    float64
	Distance              float64
}) {
	side := -1.0
	if b.Owner == 0 {
		side = 1
	}
	tagsInf := hasTag(typ, "infantry")
	reach := b.Size*.45 + 21
	if tagsInf {
		reach = b.Size*.45 + 10
	}
	dist := math.Max(82, unitRadiusOf(typ)*2+22)
	if tagsInf {
		dist = 62
	}
	if b.SpawnAngle != nil {
		angle := *b.SpawnAngle
		c, s := cos(angle), sin(angle)
		out.X = b.X + c*reach - s*b.Size*.2
		out.Y = b.Y + s*reach + c*b.Size*.2
		out.Angle = angle
		out.Side = side
		out.ForwardX, out.ForwardY = c, s
		out.Distance = dist
		return
	}
	out.X = b.X + side*reach
	out.Y = b.Y + side*b.Size*.2
	if side > 0 {
		out.Angle = 0
	} else {
		out.Angle = math.Pi
	}
	out.Side = side
	out.Distance = dist
	return
}

// productionDuration 对齐 battlefield-details.js:13-15。
func (g *Game) productionDuration(b *Building, typ string) float64 {
	f := 1.0
	if g.Players[b.Owner].Faction == "nato" {
		f = 1.06
	}
	if !g.HasPower(b.Owner) && (b.Type == "armory" || b.Type == "airfield" || b.Type == "dock") {
		f *= 2
	}
	return gpdata.UNITS[typ].Time * f
}

// unitCost 对齐 game.js unitCost。
func (g *Game) unitCost(side int, typ string) float64 {
	return gpdata.UnitCostFor(g.Players[side].Faction, typ)
}

// findSpawn 对齐 game.js L609-619（向日葵螺旋找空位）。
func (g *Game) findSpawn(x, y float64, typ string) *Point {
	radius := unitRadiusOf(typ)
	layer := unitLayerOf(typ)
	for n := 0; n < 128; n++ {
		angle := float64(n) * 2.399963
		reach := 0.0
		if n > 0 {
			reach = 18 * math.Sqrt(float64(n))
		}
		px := clamp(x+cos(angle)*reach, 50, g.World.Width-50)
		py := clamp(y+sin(angle)*reach, 50, g.World.Height-50)
		p := Point{px, py}
		if !g.canOccupyUnitType(typ, p) {
			continue
		}
		blocked := false
		for _, other := range g.Units {
			if other.HP > 0 && unitLayerOf(other.Type) == layer &&
				Distance(other.X, other.Y, px, py) < unitRadiusOf(other.Type)+radius+4 {
				blocked = true
				break
			}
		}
		if !blocked {
			return &p
		}
	}
	return nil
}

// canOccupyUnitType 与 canOccupyUnit 同语义但以类型代单位（findSpawn 用）。
func (g *Game) canOccupyUnitType(typ string, p Point) bool {
	if p.X < 18 || p.X > g.World.Width-18 || p.Y < 18 || p.Y > g.World.Height-18 {
		return false
	}
	if hasTag(typ, "air") {
		return true
	}
	layer := unitLayerOf(typ)
	if layer == "naval" || layer == "submerged" {
		return !g.isNavalBlocked(p.X, p.Y, maxF(18, unitRadiusOf(typ)))
	}
	radius := 14.0
	if hasTag(typ, "infantry") {
		radius = 6
	}
	return !g.isGroundBlocked(p.X, p.Y, radius)
}

// findNavalSpawn 对齐 game.js L632-642。
func (g *Game) findNavalSpawn(dock *Building, typ string) *Point {
	water := g.Map.Water
	padding := maxF(38, unitRadiusOf(typ))
	x := water.X1 + padding
	if dock.X >= g.World.Width/2 {
		x = water.X2 - padding
	}
	for n := 0; n < 32; n++ {
		span := 35.0 + float64(n)*16
		y := clamp(dock.Y+g.Random(-span, span), padding, g.World.Height-padding)
		blocked := false
		for _, u := range g.Units {
			if u.HP > 0 && unitLayerOf(u.Type) == unitLayerOf(typ) &&
				Distance(u.X, u.Y, x, y) < unitRadiusOf(u.Type)+unitRadiusOf(typ)+4 {
				blocked = true
				break
			}
		}
		if !blocked {
			return &Point{x, y}
		}
	}
	return &Point{x, dock.Y}
}
