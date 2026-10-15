package game

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// transport.go 对齐 modern-combat.js 运输段（transportLoad/canBoardTransport/transportGrounded/boardTransport/unloadTransport/launchDeckAircraft）。

func (g *Game) TransportLoad(t *Unit) int {
	total := 0
	for _, id := range t.Passengers {
		u := g.GetUnit(id)
		if u != nil && u.HP > 0 {
			if t.Type == "carrier" || gpdata.HasTag(u.Type, "infantry") {
				total += 1
			} else {
				total += 4
			}
		}
	}
	return total
}

func (g *Game) CanBoardTransport(u *Unit, transport *Unit) bool {
	if u == nil || transport == nil || u == transport || u.HP <= 0 || transport.HP <= 0 ||
		u.Owner != transport.Owner || u.EmbarkedIn != 0 || transport.EmbarkedIn != 0 ||
		gpdata.UNITS[transport.Type].Capacity == 0 || gpdata.UNITS[u.Type].Capacity != 0 {
		return false
	}
	weight := 4
	if gpdata.HasTag(u.Type, "infantry") {
		weight = 1
	}
	if transport.Type == "carrier" {
		return gpdata.HasTag(u.Type, "deck") && (u.HomeCarrierID == transport.ID || len(g.carrierAircraft(transport)) < int(gpdata.UNITS["carrier"].Capacity)) && len(transport.Passengers) < int(gpdata.UNITS["carrier"].Capacity)
	}
	if gpdata.HasTag(u.Type, "air") || gpdata.HasTag(u.Type, "ship") || (transport.Type == "apc" && !gpdata.HasTag(u.Type, "infantry")) {
		return false
	}
	return g.TransportLoad(transport)+weight <= int(gpdata.UNITS[transport.Type].Capacity)
}

func (g *Game) transportGrounded(t *Unit) bool {
	if t.Type == "carrier" {
		return g.Time-t.LastMovedAt >= 1
	}
	if t.Type != "airlift" {
		return true
	}
	return !g.isGroundBlocked(t.X, t.Y, 35) && g.Time-t.LastMovedAt >= .5 && func() bool {
		for _, b := range g.Buildings {
			if b.HP > 0 && Distance(b.X, b.Y, t.X, t.Y) < b.Size*.5+35 {
				return false
			}
		}
		return true
	}()
}

func (g *Game) BoardTransport(u *Unit, dt float64) {
	if u.Order == nil {
		return
	}
	transport := g.GetUnit(u.Order.TargetID)
	if !g.CanBoardTransport(u, transport) {
		u.Order = nil
		return
	}
	if transport.Type == "landing" {
		shore := g.ResolveMoveGoal(u, transport.X, transport.Y)
		if Distance(shore.X, shore.Y, transport.X, transport.Y) > 145 || !g.canOccupyUnit(u, shore) {
			return
		}
		if Distance(u.X, u.Y, transport.X, transport.Y) > 150 {
			g.moveUnit(u, shore, dt, 12)
			return
		}
	} else {
		if !g.transportGrounded(transport) {
			if transport.Type == "carrier" {
				u.DeckApproach = nil
			}
			return
		}
		if Distance(u.X, u.Y, transport.X, transport.Y) > 65 {
			g.moveUnit(u, Point{transport.X, transport.Y}, dt, 45)
			return
		}
	}
	if transport.Type == "carrier" {
		if u.DeckApproach == nil || u.DeckApproach.CarrierID != transport.ID {
			u.DeckApproach = &DeckApproach{CarrierID: transport.ID, Start: g.Time, Until: g.Time + 2.4}
			transport.LandingID = u.ID
		}
		g.moveUnit(u, Point{transport.X, transport.Y}, dt, 16)
		if g.Time < u.DeckApproach.Until {
			return
		}
		u.HomeCarrierID = transport.ID
		u.DeckApproach = nil
		transport.LandingID = 0
	}
	transport.Passengers = append(transport.Passengers, u.ID)
	u.EmbarkedIn = transport.ID
	u.Order = nil
	u.Path = nil
	u.X, u.Y = transport.X, transport.Y
	g.Selected = filterInt(g.Selected, u.ID)
	g.FogTimer = 0
	if g.Events.Selection != nil {
		g.Events.Selection()
	}
}

func filterInt(xs []int, v int) []int {
	out := xs[:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func (g *Game) UnloadTransport(transport *Unit, emergency bool) int {
	if transport == nil || gpdata.UNITS[transport.Type].Capacity == 0 {
		return 0
	}
	if transport.Type == "carrier" {
		return g.launchDeckAircraft(transport, emergency, 0)
	}
	if !emergency && (!g.transportGrounded(transport) || (transport.Order != nil && (transport.Order.Type == "move" || transport.Order.Type == "attackMove"))) {
		if transport.Owner == 0 && g.Events.Notice != nil {
			g.Events.Notice("运输单位需要在安全位置停驻后卸载")
		}
		return 0
	}
	unloaded := 0
	for _, id := range append([]int{}, transport.Passengers...) {
		passenger := g.GetUnit(id)
		if passenger == nil || passenger.HP <= 0 {
			transport.Passengers = filterInt(transport.Passengers, id)
			continue
		}
		var spot *Point
		for n := 0; n < 180; n++ {
			angle := float64(n) * 2.399963
			reach := 50 + float64(n/12)*12
			cand := Point{transport.X + cos(angle)*reach, transport.Y + sin(angle)*reach}
			if !g.canOccupyUnit(passenger, cand) {
				continue
			}
			blocked := false
			for _, v := range g.Units {
				if v.HP > 0 && v.EmbarkedIn == 0 && unitLayerOf(v.Type) == "ground" &&
					Distance(v.X, v.Y, cand.X, cand.Y) < unitRadiusOf(v.Type)+unitRadiusOf(passenger.Type)+4 {
					blocked = true
					break
				}
			}
			if !blocked {
				sp := cand
				spot = &sp
				break
			}
		}
		if spot == nil && !emergency {
			continue
		}
		passenger.EmbarkedIn = 0
		if spot != nil {
			passenger.X, passenger.Y = spot.X, spot.Y
			if emergency {
				passenger.HP *= .5
				passenger.LastDamageAt = g.Time
			}
			unloaded++
		} else {
			g.Damage(passenger, passenger.MaxHP*3, transport.LastAttacker)
		}
		transport.Passengers = filterInt(transport.Passengers, id)
	}
	if unloaded > 0 {
		g.FogTimer = 0
		if g.Events.Selection != nil {
			g.Events.Selection()
		}
	}
	if !emergency && transport.Owner == 0 && g.Events.Notice != nil && unloaded > 0 {
		g.Events.Notice(itoa(unloaded) + " 个作战单位已卸载")
	}
	return unloaded
}

func (g *Game) launchDeckAircraft(carrier *Unit, emergency bool, onlyID int) int {
	if !emergency && (!g.transportGrounded(carrier) || !g.HasPower(carrier.Owner)) {
		return 0
	}
	if !emergency {
		for _, p := range g.Projectiles {
			if !p.Finished && p.Kind == "wing" && p.SourceID == carrier.ID {
				return 0
			}
		}
	}
	count := 0
	for _, id := range append([]int{}, carrier.Passengers...) {
		if onlyID != 0 && id != onlyID {
			continue
		}
		plane := g.GetUnit(id)
		if plane == nil || plane.HP <= 0 {
			carrier.Passengers = filterInt(carrier.Passengers, id)
			continue
		}
		if !emergency && (plane.Ammo == nil || *plane.Ammo < gpdata.UNITS[plane.Type].Ammo || plane.HP < plane.MaxHP-.01) {
			continue
		}
		angle := carrier.Angle + float64(count)*.32
		goal := g.ResolveMoveGoal(plane, carrier.X+cos(angle)*230, carrier.Y+sin(angle)*230)
		plane.EmbarkedIn = 0
		plane.X, plane.Y = carrier.X, carrier.Y
		plane.Angle, plane.TurretAngle = angle, angle
		plane.Deployment = &Deployment{BuildingID: carrier.ID, Start: g.Time, Until: g.Time + 2.6, FromX: plane.X, FromY: plane.Y, X: goal.X, Y: goal.Y}
		if emergency {
			plane.HP *= .5
			plane.HomeCarrierID = 0
			plane.LastDamageAt = g.Time
		}
		g.RestoreCombatOrder(plane)
		carrier.Passengers = filterInt(carrier.Passengers, id)
		count++
	}
	if count > 0 {
		g.FogTimer = 0
		if g.Events.Selection != nil {
			g.Events.Selection()
		}
	}
	return count
}

// RestoreCombatOrder 对齐 modern-combat.js:275。
func (g *Game) RestoreCombatOrder(u *Unit) {
	gmetric(g, u.Owner, "resupplies", 1)
	u.Order = u.ResumeOrder
	u.ResumeOrder = nil
	u.RearmProgress = 0
}
