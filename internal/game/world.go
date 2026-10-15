package game

import (
	"math"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/spacing"
)

var bootTime = time.Now()

func timeNowMs() int64 { return time.Since(bootTime).Milliseconds() + bootTime.UnixMilli() }

// randToken 对齐 Math.random().toString(36).slice(2) 的形态（随机 token，无需逐位对齐——对拍走注入 RNG 的 UUID 之外的确定性字段）。
func randToken(n int) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, n)
	for i := range out {
		out[i] = digits[rand.Intn(36)]
	}
	return string(out)
}

// unitIndexView 承载 update 期间的单位空间索引（对齐 this.unitIndex）。
type unitIndexView struct {
	idx *spacing.SpatialIndex
	byID map[int]*Unit
}

func (v *unitIndexView) Nearby(u *Unit) []*Unit {
	var out []*Unit
	for _, e := range v.idx.Nearby(unitView{u}) {
		if uv, ok := e.(unitView); ok {
			out = append(out, uv.u)
		}
	}
	return out
}

// unitView 把 *Unit 适配为 spacing.Entity。
type unitView struct{ u *Unit }

func (v unitView) ID() int          { return v.u.ID }
func (v unitView) Type() string     { return v.u.Type }
func (v unitView) X() float64       { return v.u.X }
func (v unitView) Y() float64       { return v.u.Y }
func (v unitView) HP() float64      { return v.u.HP }
func (v unitView) StunUntil() float64 { return v.u.StunUntil }

// unitMobile 可移动适配器（SeparateUnits 用）。
type unitMobile struct{ u *Unit }

func (m unitMobile) ID() int            { return m.u.ID }
func (m unitMobile) Type() string       { return m.u.Type }
func (m unitMobile) X() float64         { return m.u.X }
func (m unitMobile) Y() float64         { return m.u.Y }
func (m unitMobile) HP() float64        { return m.u.HP }
func (m unitMobile) StunUntil() float64 { return m.u.StunUntil }
func (m unitMobile) SetPos(x, y float64) { m.u.X, m.u.Y = x, y }

var _ = atomic.Int64{}
var _ = strconv.Itoa
var _ = math.Sqrt
var _ = gpdata.HasTag

// initBases 对齐 game.js L149-175 initBases。
func (g *Game) initBases() {
	for _, side := range g.Sides() {
		mirror := func(x, y float64) (float64, float64) {
			if g.PlayerCount == 6 {
				p := gpdata.SixPoint(side, x-205, y-720)
				return p.X, p.Y
			}
			if side == 0 {
				return x, y + g.HomeY - 720
			}
			return g.World.Width - x, g.World.Height - y - g.HomeY + 720
		}
		type spot struct {
			typ string
			x, y float64
		}
		var spots []spot
		add := func(t string, x, y float64) { spots = append(spots, spot{t, x, y}) }
		add("hq", 205, 720)
		add("power", 320, 595)
		add("refinery", 335, 825)
		add("barracks", 195, 875)
		add("factory", 460, 705)
		if g.Map.Water != nil {
			dx := 850.0
			if g.MapID == "ocean" || g.MapID == "archipelago" {
				dx = 950
			}
			add("dock", dx, 720)
		}
		for _, s := range spots {
			bx, by := mirror(s.x, s.y)
			if s.typ == "dock" && g.BattlefieldScale != 1 {
				z := g.Map.DockZones[side]
				bx = (z.X1 + z.X2) / 2
				by = g.HomeY
			}
			g.AddBuilding(side, s.typ, bx, by)
		}
		var units []spot
		if g.EconomyMode == "mining" {
			units = append(units, spot{"harvester", 405, 940}, spot{"harvester", 440, 485})
		}
		units = append(units,
			spot{"rifle", 575, 630}, spot{"rifle", 575, 680}, spot{"rifle", 575, 730},
			spot{"tank", 605, 790},
		)
		for _, s := range units {
			ux, uy := mirror(s.x, s.y)
			g.AddUnit(side, s.typ, ux, uy, false)
		}
		if g.MapID == "ocean" || g.MapID == "archipelago" {
			x, y := mirror(1160, 720)
			if g.BattlefieldScale != 1 {
				if side != 0 {
					x = g.Map.Water.X2 - 120
				} else {
					x = g.Map.Water.X1 + 120
				}
				y = g.HomeY
			}
			g.AddUnit(side, "patrol", x, y, false)
			g.Players[side].Credits += 800
		}
	}
	g.RecalculatePower()
	if g.Events.Notice != nil {
		if g.EconomyMode == "mining" {
			g.Events.Notice("月表基地已部署，采矿车开始就地取材。")
		} else {
			g.Events.Notice("战场已部署，外部后勤运输航线已建立。")
		}
	}
}

// AddBuilding 对齐 game.js L176-183 addBuilding。
func (g *Game) AddBuilding(owner int, typ string, x, y float64) *Building {
	d := gpdata.BUILDINGS[typ]
	b := &Building{
		ID: int(nextID.Add(1)), Kind: "building", Owner: owner, Type: typ, X: x, Y: y,
		HP: d.HP, MaxHP: d.HP, Size: d.Size, SpawnAt: g.Time,
		Queue: []string{},
	}
	angle := math.Pi
	if owner == 0 {
		angle = 0
	}
	if g.Map.Spawns != nil && owner < len(g.Map.Spawns) {
		b.Angle = g.Map.Spawns[owner].Angle
		sa := g.Map.Spawns[owner].Angle
		b.SpawnAngle = &sa
	} else {
		b.Angle = angle
	}
	g.Buildings = append(g.Buildings, b)
	g.RecalculatePower()
	return b
}

// AddUnit 对齐 game.js L184-198 addUnit。
func (g *Game) AddUnit(owner int, typ string, x, y float64, temporary bool) *Unit {
	d := gpdata.UNITS[typ]
	p := g.Players[owner]
	hp := gpdata.UnitHealthFor(p.Faction, typ)
	u := &Unit{
		ID: int(nextID.Add(1)), Kind: "unit", Owner: owner, Type: typ, X: x, Y: y,
		HP: hp, MaxHP: hp, HarvestState: "find",
		FireTimer: g.Random(0, 0.4),
	}
	if owner != 0 {
		u.Angle, u.TurretAngle = math.Pi, math.Pi
	}
	if d.Ammo != 0 {
		a := d.Ammo
		u.Ammo = &a
	}
	if d.Stock != 0 {
		s := d.Stock
		u.Stock = &s
	}
	if d.Wing != 0 {
		w := d.Wing
		u.Wing = &w
	}
	if typ == "supply" {
		t := true
		u.AutoSupply = &t
	}
	if g.IsRobot(typ) {
		bat := 100.0
		u.Battery = &bat
		u.Ammo = nil
	}
	if temporary {
		u.TemporaryUntil = g.Time + 27
	}
	g.Units = append(g.Units, u)
	if g.IsAI(owner) && typ == "strike" {
		g.Players[owner].AIFirstStrikeProduced = true
	}
	g.Players[owner].UnitCount++
	return u
}

// RecalculatePower 对齐 game.js L199-214 recalculatePower。
func (g *Game) RecalculatePower() {
	for _, p := range g.Players {
		p.PowerIn, p.PowerOut = 0, 0
		for _, b := range g.Buildings {
			if b.Owner != p.Side || b.HP <= 0 {
				continue
			}
			power := gpdata.BUILDINGS[b.Type].Power
			if power > 0 {
				p.PowerIn += power
			} else {
				p.PowerOut -= power
			}
		}
		if g.Map.Future {
			n := 0
			for _, s := range g.Beacons {
				if s.Owner == p.Side {
					n++
				}
			}
			p.PowerIn += float64(n) * 40
		}
		depot := 0
		for _, s := range g.ResourceSites {
			if s.Type == "depot" && s.Owner == p.Side {
				depot++
			}
		}
		p.PowerIn += float64(depot) * 40
		load := 0
		for _, u := range g.Units {
			if u.Owner == p.Side && u.HP > 0 && u.EmbarkedIn == 0 && g.IsRobot(u.Type) &&
				u.Order != nil && u.Order.Type == "rearm" && g.RobotChargeSite(u) != nil {
				load++
			}
		}
		p.RobotChargeLoad = float64(load) * 4 // ROBOT_ENERGY.chargeLoad
		p.PowerOut += p.RobotChargeLoad
	}
}

// RobotChargeSite 对齐 lunar-robots.js robotChargeSite（就近充电点）。
func (g *Game) RobotChargeSite(u *Unit) any {
	var cands []struct {
		ent       any
		x, y      float64
		id        int
		lastDamage float64
	}
	for _, b := range g.OwnedBuildings(u.Owner, "") {
		if b.Type != "power" && b.Type != "barracks" {
			continue
		}
		if Distance(u.X, u.Y, b.X, b.Y) <= b.Size*.55+40 {
			cands = append(cands, struct {
				ent        any
				x, y       float64
				id         int
				lastDamage float64
			}{b, b.X, b.Y, b.ID, b.LastDamageAt})
		}
	}
	for _, t := range g.ActiveUnits(u.Owner, "supply") {
		if t.Stock != nil && *t.Stock >= 8 && g.Time-t.LastMovedAt >= .5 &&
			t.Order != nil && t.Order.Type != "move" && t.Order.Type != "attackMove" && t.Order.Type != "restock" &&
			Distance(u.X, u.Y, t.X, t.Y) <= 125 {
			cands = append(cands, struct {
				ent        any
				x, y       float64
				id         int
				lastDamage float64
			}{t, t.X, t.Y, t.ID, t.LastDamageAt})
		}
	}
	var best any
	bestDist := math.Inf(1)
	for _, c := range cands {
		if g.Time-c.lastDamage <= 3 {
			continue
		}
		d := Distance(u.X, u.Y, c.x, c.y)
		if d < bestDist || (d == bestDist && c.id < 0) {
			bestDist, best = d, c.ent
		}
	}
	return best
}

// ActiveUnits 对齐 JS activeUnits(side, type)（未上舰且存活）。
func (g *Game) ActiveUnits(side int, typ string) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.Owner == side && u.HP > 0 && u.EmbarkedIn == 0 && (typ == "" || u.Type == typ) {
			out = append(out, u)
		}
	}
	return out
}

// UpdateFog 对齐 game.js updateFog。
func (g *Game) UpdateFog() {
	type src struct {
		x, y, sight float64
		overTerrain bool
	}
	for _, side := range g.Sides() {
		fog := g.Fogs[side]
		for i := range fog.Visible {
			fog.Visible[i] = false
		}
		if g.Players[side].SatelliteUntil > g.Time {
			for i := range fog.Visible {
				fog.Visible[i], fog.Explored[i] = true, true
			}
			continue
		}
		var sources []src
		for _, u := range g.ActiveUnits(side, "") {
			if g.IsRobot(u.Type) && (u.Battery == nil || *u.Battery <= 0) {
				continue
			}
			d := gpdata.UNITS[u.Type]
			sources = append(sources, src{u.X, u.Y, d.Sight,
				gpdata.HasTag(u.Type, "air") || gpdata.HasTag(u.Type, "ship") || u.Type == "aa"})
		}
		for _, b := range g.OwnedBuildings(side, "") {
			sight := 310.0
			over := false
			if b.Type == "radar" && g.HasPower(side) {
				sight, over = 630, true
			} else if b.Type == "hq" {
				sight = 400
			}
			sources = append(sources, src{b.X, b.Y, sight, over})
		}
		for _, beacon := range g.Beacons {
			if beacon.Owner == side {
				sources = append(sources, src{beacon.X, beacon.Y, 650, true})
			}
		}
		for _, s := range g.ResourceSites {
			if s.Type == "depot" && s.Owner == side {
				sources = append(sources, src{s.X, s.Y, 220, false})
			}
		}
		for _, s := range sources {
			minX := int(clamp(math.Floor((s.x-s.sight)/g.World.Fog), 0, float64(fog.Cols-1)))
			maxX := int(clamp(math.Floor((s.x+s.sight)/g.World.Fog), 0, float64(fog.Cols-1)))
			minY := int(clamp(math.Floor((s.y-s.sight)/g.World.Fog), 0, float64(fog.Rows-1)))
			maxY := int(clamp(math.Floor((s.y+s.sight)/g.World.Fog), 0, float64(fog.Rows-1)))
			for y := minY; y <= maxY; y++ {
				for x := minX; x <= maxX; x++ {
					cx := (float64(x) + 0.5) * g.World.Fog
					cy := (float64(y) + 0.5) * g.World.Fog
					if Distance(s.x, s.y, cx, cy) >= s.sight {
						continue
					}
					if !s.overTerrain {
						blocked := false
						for _, r := range g.Map.Barriers {
							if crossesRect(s.x, s.y, cx, cy, r) {
								blocked = true
								break
							}
						}
						if blocked {
							continue
						}
					}
					fog.Visible[y*fog.Cols+x] = true
					fog.Explored[y*fog.Cols+x] = true
				}
			}
		}
	}
}

// crossesRect 对齐 game.js L19-30 crossesRect（线段与矩形相交）。
func crossesRect(sx, sy, ex, ey float64, r gpdata.Rect) bool {
	near, far := 0.0, 1.0
	axes := []struct {
		origin, delta, min, max float64
	}{{sx, ex - sx, r.X1, r.X2}, {sy, ey - sy, r.Y1, r.Y2}}
	for _, a := range axes {
		if math.Abs(a.delta) < 0.0001 {
			if a.origin < a.min || a.origin > a.max {
				return false
			}
			continue
		}
		a1 := (a.min - a.origin) / a.delta
		a2 := (a.max - a.origin) / a.delta
		lo, hi := math.Min(a1, a2), math.Max(a1, a2)
		near = math.Max(near, lo)
		far = math.Min(far, hi)
		if near > far {
			return false
		}
	}
	return true
}

// IsVisibleFor 对齐 isVisibleFor(side, x, y)。
func (g *Game) IsVisibleFor(side int, x, y float64) bool {
	fog := g.Fogs[side]
	cx := int(clamp(math.Floor(x/g.World.Fog), 0, float64(fog.Cols-1)))
	cy := int(clamp(math.Floor(y/g.World.Fog), 0, float64(fog.Rows-1)))
	return fog.Visible[cy*fog.Cols+cx]
}

// HasExploredFor 对齐 hasExploredFor(side, x, y)。
func (g *Game) HasExploredFor(side int, x, y float64) bool {
	fog := g.Fogs[side]
	cx := int(clamp(math.Floor(x/g.World.Fog), 0, float64(fog.Cols-1)))
	cy := int(clamp(math.Floor(y/g.World.Fog), 0, float64(fog.Rows-1)))
	return fog.Explored[cy*fog.Cols+cx]
}

// intersectsRect 对齐 game.js L18（圆与矩形相交）。
func intersectsRect(x, y, radius float64, r gpdata.Rect) bool {
	cx := clamp(x, r.X1, r.X2)
	cy := clamp(y, r.Y1, r.Y2)
	return Distance(x, y, cx, cy) <= radius
}
