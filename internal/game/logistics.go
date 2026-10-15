package game

import "fmt"

// Logistics 对齐 logistics-economy.js 的 logisticsEconomy mixin（挂到 Game）。

// supplyRoute 对齐 SUPPLY_ROUTES。
type supplyRoute struct {
	Interval float64
	Value    float64
	Unload   float64
}

var supplyRoutes = map[string]supplyRoute{
	"air": {Interval: 36, Value: 720, Unload: 6},
	"sea": {Interval: 60, Value: 900, Unload: 9},
}

var freightTypes = map[string]bool{"freightPlane": true, "containerShip": true}

// initLogistics 对齐 initLogistics。
func (g *Game) initLogistics() {
	g.LogisticsRoutes = make([]*LogisticsRoute, len(g.Players))
	for i, p := range g.Players {
		g.LogisticsRoutes[i] = &LogisticsRoute{Side: p.Side, NextAir: g.Time + 3, NextSea: g.Time + 10}
	}
}

// FreightDestination 对齐 freightDestination(side, sea)。
func (g *Game) FreightDestination(side int, sea bool) *Building {
	typ := "refinery"
	if sea {
		typ = "dock"
	}
	var best *Building
	for _, b := range g.OwnedBuildings(side, typ) {
		if best == nil || b.ID < best.ID {
			best = b
		}
	}
	if best != nil || sea {
		return best
	}
	bs := g.OwnedBuildings(side, "hq")
	if len(bs) > 0 {
		return bs[0]
	}
	return nil
}

// FreightEntry 对齐 freightEntry(side, sea)。
func (g *Game) FreightEntry(side int, sea bool) Point {
	margin := 90.0
	if sea {
		margin = 88 + 12 // unitRadius(containerShip)+12
	}
	x := g.World.Width / 2
	if sea {
		x = (g.Map.Water.X1 + g.Map.Water.X2) / 2
	}
	y := margin
	if side != 0 {
		y = g.World.Height - margin
	}
	return Point{X: x, Y: y}
}

// LaunchFreight 对齐 launchFreight(side, sea)。
func (g *Game) LaunchFreight(side int, sea bool) *Unit {
	home := g.FreightDestination(side, sea)
	if home == nil {
		return nil
	}
	entry := g.FreightEntry(side, sea)
	typ := "freightPlane"
	if sea {
		typ = "containerShip"
	}
	u := g.AddUnit(side, typ, entry.X, entry.Y, false)
	value := supplyRoutes["air"].Value
	if sea {
		value = supplyRoutes["sea"].Value
	}
	if home.Type == "hq" {
		value = 180
	}
	u.Freight = &Freight{HomeID: home.ID, Phase: "inbound", Entry: entry, Value: value}
	return u
}

// UpdateLogistics 对齐 updateLogistics(dt)。
func (g *Game) UpdateLogistics(dt float64) {
	if g.EconomyMode != "convoy" {
		return
	}
	for _, route := range g.LogisticsRoutes {
		for _, ch := range []struct {
			name string
			sea  bool
			typ  string
		}{{"Air", false, "freightPlane"}, {"Sea", true, "containerShip"}} {
			if ch.sea && g.Map.Water == nil {
				continue
			}
			var next *float64
			if ch.name == "Air" {
				next = &route.NextAir
			} else {
				next = &route.NextSea
			}
			if g.Time < *next {
				continue
			}
			if len(g.ActiveUnits(route.Side, ch.typ)) > 0 {
				continue
			}
			interval := supplyRoutes["air"].Interval
			if ch.sea {
				interval = supplyRoutes["sea"].Interval
			}
			*next = g.Time + interval
			g.LaunchFreight(route.Side, ch.sea)
		}
	}
}

// ActivateSatellite 对齐 activateSatellite(side)。
func (g *Game) ActivateSatellite(side int) bool {
	p := g.Players[side]
	if !g.Running || g.Paused || !g.HasBuilding(side, "radar") || !g.HasBuilding(side, "lab") ||
		!g.HasPower(side) || p.Credits < 1000 || p.SatelliteReadyAt > g.Time {
		return false
	}
	p.Credits -= 1000
	p.SatelliteUntil = g.Time + 8
	p.SatelliteReadyAt = g.Time + 120
	g.UpdateFog()
	if g.Events.Notice != nil {
		if side == 0 {
			g.Events.Notice("侦察卫星过境：全图可见 8 秒，潜航与隐身仍需专用探测")
		} else {
			g.Events.Notice("敌方侦察卫星正在过境")
		}
	}
	return true
}

var _ = fmt.Sprintf
