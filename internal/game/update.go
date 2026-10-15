package game

import (
	gpreport "github.com/cnyup/gopher-powers/internal/game/report"
	"github.com/cnyup/gopher-powers/internal/game/spacing"
)

// metricStats 对齐 battleMetric 所需视图。
func (g *Game) IsRunning() bool   { return g.Running }
func (g *Game) Stats() *gpreport.Stats { return g.BattleStats }
func (g *Game) TimeNow() float64  { return g.Time }

// Update 对齐 game.js L479-524 主循环。
// mixin 阶段（updateIntelligence/updateElectronicWarfare/updateProjectiles 等）
// 由各分批补齐；此处主序与钳制先落。
func (g *Game) Update(dt float64) {
	if !g.Running || g.Paused {
		return
	}
	if dt < 0 {
		dt = 0
	}
	if dt > 0.05 {
		dt = 0.05
	}
	g.Time += dt
	g.RecalculatePower()
	g.UpdateIntelligence(dt)
	g.UpdateLogistics(dt)
	for _, p := range g.Players {
		oilCount := 0
		for _, o := range g.Oil {
			if o.Owner == p.Side {
				oilCount++
			}
		}
		gpreport.Metric(g, p.Side, "oilSeconds", float64(oilCount)*dt)
		beaconCount := 0
		for _, o := range g.Beacons {
			if o.Owner == p.Side {
				beaconCount++
			}
		}
		gpreport.Metric(g, p.Side, "beaconSeconds", float64(beaconCount)*dt)
		p.Credits += float64(oilCount) * 11 * dt
		if g.VictoryMode == "control" {
			p.ControlScore += float64(beaconCount) * dt
		}
		if p.AbilityCooldown > 0 {
			p.AbilityCooldown -= dt
			if p.AbilityCooldown < 0 {
				p.AbilityCooldown = 0
			}
		}
		if g.HasBuilding(p.Side, "super") && g.HasPower(p.Side) && p.AbilityCharge < 100 {
			p.AbilityCharge += dt * 0.85
			if p.AbilityCharge > 100 {
				p.AbilityCharge = 100
			}
		}
		g.updateBuildQueue(p, dt)
	}
	g.PendingHits = g.PendingHits[:0]
	g.PendingMoves = g.PendingMoves[:0]
	g.CollectingHits = true
	g.CollectingMoves = true
	g.updateElectronicWarfare(dt)
	g.updateProjectiles(dt)
	for _, b := range g.Buildings {
		if b.HP > 0 {
			g.updateBuilding(b, dt)
		}
	}
	for _, u := range g.Units {
		if u.HP > 0 && u.EmbarkedIn == 0 {
			g.updateUnit(u, dt)
		}
	}
	g.CollectingHits = false
	g.CollectingMoves = false
	g.rebuildUnitIndex()
	for _, m := range g.PendingMoves {
		if m.unit.HP > 0 {
			g.moveUnitNow(m.unit, m.goal, m.dt, m.stopDistance)
		}
	}
	g.separateUnits(dt)
	g.UnitIndex = nil
	for _, h := range g.PendingHits {
		g.applyHit(h)
	}
	// 过滤死亡实体。
	liveU := g.Units[:0]
	for _, u := range g.Units {
		if u.HP > 0 {
			liveU = append(liveU, u)
		}
	}
	g.Units = liveU
	liveB := g.Buildings[:0]
	for _, b := range g.Buildings {
		if b.HP > 0 {
			liveB = append(liveB, b)
		}
	}
	g.Buildings = liveB
	for _, e := range g.Effects {
		e.Age += dt
	}
	liveE := g.Effects[:0]
	for _, e := range g.Effects {
		if e.Age < e.Duration {
			liveE = append(liveE, e)
		}
	}
	g.Effects = liveE
	g.FogTimer -= dt
	if g.FogTimer <= 0 {
		g.UpdateFog()
		g.FogTimer = 0.22
	}
	// AI 双计时器。
	hasAI := false
	for _, p := range g.Players {
		if g.IsAI(p.Side) && !p.Eliminated {
			hasAI = true
		}
	}
	if hasAI {
		g.AIThinkTimer -= dt
		if g.AIThinkTimer <= 0 {
			for _, p := range g.Players {
				if g.IsAI(p.Side) && !p.Eliminated {
					g.updateAI(p.Side)
				}
			}
			g.AIThinkTimer = g.Difficulty.ThinkInterval
		}
		g.AIWaveTimer -= dt
		if g.AIWaveTimer <= 0 {
			for _, p := range g.Players {
				if g.IsAI(p.Side) && !p.Eliminated {
					g.launchAIWave(p.Side)
				}
			}
			g.AIWaveTimer = g.Difficulty.WaveInterval + g.Random(-4, 5)
		}
	}
	// 选中列表清理。
	changed := false
	liveS := g.Selected[:0]
	for _, id := range g.Selected {
		e := g.getEntityAny(id)
		if e != nil && !e.embarked() {
			liveS = append(liveS, id)
		} else {
			changed = true
		}
	}
	g.Selected = liveS
	if changed && g.Events.Selection != nil {
		g.Events.Selection()
	}
	g.CheckVictory()
}

// getEntityAny 数字 id 到统一视图（含 oil/beacon/resource 的字符串 id 不在此列）。
type entityAny interface {
	embarked() bool
}

func (g *Game) getEntityAny(id int) entityAny {
	for _, u := range g.Units {
		if u.ID == id {
			return unitView2{u}
		}
	}
	for _, b := range g.Buildings {
		if b.ID == id {
			return buildingView2{b}
		}
	}
	return nil
}

type unitView2 struct{ u *Unit }

func (v unitView2) embarked() bool { return v.u.EmbarkedIn != 0 }

type buildingView2 struct{ b *Building }

func (v buildingView2) embarked() bool { return false }

func (g *Game) rebuildUnitIndex() {
	ents := make([]spacing.Entity, 0, len(g.Units))
	for _, u := range g.Units {
		ents = append(ents, unitView{u})
	}
	g.UnitIndex = &unitIndexView{idx: spacing.NewSpatialIndex(ents)}
}

func (g *Game) separateUnits(dt float64) {
	mobs := make([]spacing.Mobile, 0, len(g.Units))
	for _, u := range g.Units {
		if u.HP > 0 && u.EmbarkedIn == 0 {
			mobs = append(mobs, unitMobile{u})
		}
	}
	spacing.SeparateUnits(mobs, dt, func(m spacing.Mobile, p spacing.Point) bool {
		return g.canOccupyUnit(m.(unitMobile).u, Point{p.X, p.Y})
	}, g.Time)
}

// CheckVictory 对齐 game.js L1350-1375。
func (g *Game) CheckVictory() {
	if g.PlayerCount == 6 {
		for _, p := range g.Players {
			if p.Eliminated {
				continue
			}
			lost := false
			if g.VictoryMode == "annihilation" {
				lost = len(g.OwnedBuildings(p.Side, "")) == 0 && len(g.nonFreightUnits(p.Side)) == 0
			} else {
				lost = !g.hasCoreBuilding(p.Side)
			}
			if lost {
				p.Eliminated = true
				p.BuildQueue = nil
				p.ReadyBuilding = ""
				g.removeSideEntities(p.Side)
			}
		}
		alive := 0
		var aliveSide any = "draw"
		for _, p := range g.Players {
			if !p.Eliminated {
				alive++
				aliveSide = p.Side
			}
		}
		if alive <= 1 {
			g.endGame(aliveSide)
			return
		}
		if g.VictoryMode == "control" {
			var winners []*Player
			for _, p := range g.Players {
				if !p.Eliminated && p.ControlScore >= 240 {
					winners = append(winners, p)
				}
			}
			if len(winners) == 1 {
				g.endGame(winners[0].Side)
			} else if len(winners) > 1 {
				g.endGame("draw")
			}
		}
		return
	}
	defeated := [2]bool{}
	for side := 0; side < 2; side++ {
		if g.VictoryMode == "annihilation" {
			defeated[side] = len(g.OwnedBuildings(side, "")) == 0 && len(g.nonFreightUnits(side)) == 0
		} else {
			defeated[side] = !g.hasCoreBuilding(side)
		}
	}
	if defeated[0] && defeated[1] {
		g.endGame("draw")
	} else if defeated[0] {
		g.endGame(1)
	} else if defeated[1] {
		g.endGame(0)
	} else if g.VictoryMode == "control" {
		w0 := g.Players[0].ControlScore >= 240
		w1 := g.Players[1].ControlScore >= 240
		if w0 && w1 {
			g.endGame("draw")
		} else if w0 {
			g.endGame(0)
		} else if w1 {
			g.endGame(1)
		}
	}
}

func (g *Game) nonFreightUnits(side int) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.Owner == side && u.Type != "freightPlane" && u.Type != "containerShip" {
			out = append(out, u)
		}
	}
	return out
}

func (g *Game) hasCoreBuilding(side int) bool {
	for _, b := range g.OwnedBuildings(side, "") {
		switch b.Type {
		case "hq", "refinery", "barracks", "factory", "armory", "airfield", "dock":
			return true
		}
	}
	return false
}

func (g *Game) removeSideEntities(side int) {
	liveU := g.Units[:0]
	for _, u := range g.Units {
		if u.Owner != side {
			liveU = append(liveU, u)
		}
	}
	g.Units = liveU
	liveB := g.Buildings[:0]
	for _, b := range g.Buildings {
		if b.Owner != side {
			liveB = append(liveB, b)
		}
	}
	g.Buildings = liveB
	for _, s := range g.Oil {
		if s.Owner == side {
			s.Owner = -1
		}
	}
	for _, s := range g.Beacons {
		if s.Owner == side {
			s.Owner = -1
		}
	}
	for i := range g.ResourceSites {
		if g.ResourceSites[i].Owner == side {
			g.ResourceSites[i].Owner = -1
		}
	}
}

// endGame 对齐 game.js L1377-1383。
func (g *Game) endGame(winner any) {
	if g.Winner != nil {
		return
	}
	g.Winner = winner
	g.Running = false
	if g.VictoryMode == "quick" {
		if side, ok := winner.(int); ok {
			live := g.Units[:0]
			for _, u := range g.Units {
				if u.Owner == side {
					live = append(live, u)
				}
			}
			g.Units = live
		}
	}
	if g.Events.End != nil {
		g.Events.End(winner)
	}
}
