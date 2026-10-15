package game

// freight.go：updateFreight 完整版（对齐 logistics-economy.js updateFreight 全量）。
// 替换 stubs.go 中的最小闭环版。

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

func (g *Game) updateFreightFull(u *Unit, dt float64) {
	f := u.Freight
	if f == nil {
		u.HP = 0
		return
	}
	sea := u.Type == "containerShip"
	rule := supplyRoutes["air"]
	if sea {
		rule = supplyRoutes["sea"]
	}
	if g.Time < u.StunUntil {
		return
	}
	if f.Phase == "outbound" {
		exit := f.Entry
		if !sea {
			dx := 220.0
			if u.Owner != 0 {
				dx = -220
			}
			exit.X = clamp(f.Entry.X+dx, 90, g.World.Width-90)
		}
		if Distance(u.X, u.Y, exit.X, exit.Y) < 24 {
			u.HP = 0
			return
		}
		g.moveUnit(u, Point{exit.X, exit.Y}, dt, 18)
		return
	}
	home := g.GetBuilding(f.HomeID)
	if home == nil || home.HP <= 0 || home.Owner != u.Owner {
		home = g.FreightDestination(u.Owner, sea)
		if home != nil {
			f.HomeID = home.ID
		} else {
			f.HomeID = 0
		}
		f.Progress = 0
		f.OrbitAngle = nil
		if home != nil && home.Type == "hq" && f.Value > 180 {
			f.Value = 180
		}
	}
	if home == nil {
		f.Phase = "outbound"
		return
	}
	if !sea {
		// 空运：串行泊位。首班（id 最小）卸货，其余等待航线绕圈。
		var first *Unit
		for _, other := range g.ActiveUnits(u.Owner, "freightPlane") {
			if other.Freight != nil && other.Freight.HomeID == home.ID {
				if first == nil || other.ID < first.ID {
					first = other
				}
			}
		}
		waiting := first != u
		arrived := g.flyFreightCircuit(u, home, dt, waiting)
		if waiting || !arrived {
			f.Phase = "inbound"
			if !waiting {
				f.Progress = 0
			}
			return
		}
	} else {
		var first *Unit
		for _, other := range g.ActiveUnits(u.Owner, "containerShip") {
			if other.Freight != nil && other.Freight.HomeID == home.ID {
				if first == nil || other.ID < first.ID {
					first = other
				}
			}
		}
		f.Holding = first != u
		if f.Holding {
			goalX := (g.Map.Water.X1 + g.Map.Water.X2) / 2
			dir := 1.0
			if u.Owner != 0 {
				dir = -1
			}
			goal := g.navalGoal(goalX, home.Y+dir*360, unitRadiusOf(u.Type)+12)
			f.Phase = "inbound"
			g.moveUnit(u, goal, dt, 45)
			return
		}
		goal := g.navalGoal(home.X, home.Y, unitRadiusOf(u.Type)+12)
		if Distance(u.X, u.Y, goal.X, goal.Y) > 95 {
			f.Phase = "inbound"
			f.Progress = 0
			g.moveUnit(u, goal, dt, 66.5)
			return
		}
		if u.Owner != 0 {
			u.Angle = -1.5707963267948966
		} else {
			u.Angle = 1.5707963267948966
		}
	}
	f.Phase = "unloading"
	if !g.HasPower(u.Owner) || (sea && g.Time-u.LastMovedAt < .5) || g.Time-u.LastDamageAt <= 3 || g.Time-home.LastDamageAt <= 3 {
		return
	}
	if !sea && g.Time >= f.DropFXAt {
		f.DropFXAt = g.Time + 2
		dx := -1.0
		if u.Owner == 0 {
			dx = 1
		}
		fx := home.X + dx*home.Size*.65
		g.Effects = append(g.Effects, &Effect{Type: "supplyDrop", X: fx, Y: home.Y + 18, Owner: u.Owner, Age: 0, Duration: 3})
	}
	f.Progress += dt
	if f.Progress+1e-6 < rule.Unload {
		return
	}
	g.Players[u.Owner].Credits += f.Value
	g.LogisticsRoutes[u.Owner].Delivered += f.Value
	g.Effects = append(g.Effects, &Effect{Type: "capture", X: home.X, Y: home.Y, Owner: u.Owner, Age: 0, Duration: 1.2})
	if u.Owner == 0 && g.Events.Notice != nil {
		kind := "后勤运输机"
		if sea {
			kind = "集装箱船"
		}
		g.Events.Notice(kind + "卸货完成：+" + itoa(int(f.Value)) + " 资金")
	}
	f.Value = 0
	f.Phase = "outbound"
	f.Progress = 0
}

// flyFreightCircuit 对齐 logistics-economy.js airFreightPattern + flyFreightCircuit。
func (g *Game) flyFreightCircuit(u *Unit, home *Building, dt float64, waiting bool) bool {
	f := u.Freight
	radius := unitRadiusOf(u.Type) * 2.4
	if radius < 150 {
		radius = 150
	}
	margin := radius + 25
	clampV := func(v, size float64) float64 { return clamp(v, margin, size-margin) }
	center := Point{clampV(home.X, g.World.Width), clampV(home.Y, g.World.Height)}
	if waiting {
		gap := radius*2 + unitRadiusOf(u.Type)*2 + 60
		dir := 1.0
		if center.X >= g.World.Width/2 {
			dir = -1
		}
		center.X = clampV(center.X+dir*gap, g.World.Width)
	}
	if f.Holding != waiting {
		f.Holding = waiting
		f.OrbitAngle = nil
	}
	if Distance(u.X, u.Y, center.X, center.Y) > radius+45 {
		g.moveUnit(u, center, dt, radius)
		return false
	}
	if f.OrbitAngle == nil {
		a := atan2(u.Y-center.Y, u.X-center.X)
		f.OrbitAngle = &a
	}
	*f.OrbitAngle += dt * 128 / radius
	g.moveUnit(u, Point{center.X + cos(*f.OrbitAngle+.35)*radius, center.Y + sin(*f.OrbitAngle+.35)*radius}, dt, 0)
	return true
}

var _ = gpdata.HasTag
