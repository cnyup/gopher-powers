package game

// stubs.go：update 主循环引用的 mixin 桩——各分批（1f-2..4/1d）逐个替换为完整实现。
// 每个桩保持源签名语义，先让主循环可编译可测，避免一次性落地 1000 行无法定位回归。

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// --- modern-combat（Wave 1d 完整实现，先空转）---

func (g *Game) updateElectronicWarfare(dt float64) { g.updateElectronicWarfareFull(dt) }
func (g *Game) updateProjectiles(dt float64)       { g.updateProjectilesFull(dt) }

// --- 生产（Wave 1f-4）---

func (g *Game) updateBuildQueue(p *Player, dt float64) {
	// 对齐 game.js L526-548。
	if !g.HasBuilding(p.Side, "hq") {
		if p.BuildQueue != nil {
			p.Credits += p.BuildQueue.Paid * .5
		}
		p.BuildQueue = nil
		p.ReadyBuilding = ""
		if p.Side == 0 {
			g.PendingBuilding = ""
		}
		return
	}
	q := p.BuildQueue
	if q == nil {
		return
	}
	d := gpdata.BUILDINGS[q.Type]
	need := d.Cost - q.Paid
	if max := d.Cost / d.Time * dt; need > max {
		need = max
	}
	if p.Credits+0.01 < need {
		return
	}
	p.Credits -= need
	q.Paid += need
	q.Progress += dt
	if q.Progress >= d.Time || q.Paid >= d.Cost-0.1 {
		p.BuildQueue = nil
		if g.Multiplayer {
			p.ReadyBuilding = q.Type
		}
		if g.IsAI(p.Side) {
			g.placeAIBuilding(q.Type, p.Side)
		} else if p.Side == 0 {
			g.PendingBuilding = q.Type
			if g.Events.Notice != nil {
				g.Events.Notice(lunarName(g, q.Type) + "建造完成，请在基地附近部署")
			}
			if g.Events.Voice != nil {
				g.Events.Voice("buildReady")
			}
		}
	}
}

func (g *Game) updateBuilding(b *Building, dt float64) {
	// 对齐 game.js L549-608（维修/炮塔/生产）。
	p := g.Players[b.Owner]
	if b.Repairing {
		pricePerHp := gpdata.BUILDINGS[b.Type].Cost * .5 / b.MaxHP
		amount := 42 * dt
		if b.MaxHP-b.HP < amount {
			amount = b.MaxHP - b.HP
		}
		if p.Credits/pricePerHp < amount {
			amount = p.Credits / pricePerHp
		}
		if amount > 0 {
			b.HP += amount
			p.Credits -= amount * pricePerHp
			gmetric(g, b.Owner, "repairHP", amount)
		}
		if b.HP >= b.MaxHP-.01 {
			b.HP = b.MaxHP
			b.Repairing = false
		}
	}
	if b.Type == "turret" {
		if !g.HasPower(b.Owner) || g.Time < b.StunUntil {
			return
		}
		b.FireTimer -= dt
		if b.FireTimer <= 0 {
			target := g.ClosestEnemy(b, 250, b.Owner)
			if target != nil {
				b.Angle = atan2(target.getY()-b.Y, target.getX()-b.X)
				g.Fire(b, target, 26, "turret")
				b.FireTimer = 0.9
			}
		}
	}
	if !isProducer(b.Type) {
		return
	}
	if b.Active == nil && len(b.Queue) > 0 {
		b.Active = &Active{Type: b.Queue[0], Progress: 0, Paid: 0}
		b.Queue = b.Queue[1:]
	}
	if b.Active == nil {
		return
	}
	if g.Map.Future && b.Type == "barracks" && !g.HasPower(b.Owner) {
		return
	}
	typ := b.Active.Type
	duration := g.productionDuration(b, typ)
	cost := g.unitCost(b.Owner, typ)
	need := cost - b.Active.Paid
	if max := cost / duration * dt; need > max {
		need = max
	}
	if p.Credits+0.01 < need {
		return
	}
	p.Credits -= need
	b.Active.Paid += need
	b.Active.Progress += dt
	if b.Active.Progress >= duration || b.Active.Paid >= cost-0.1 {
		exit := g.productionExit(b, typ)
		if g.Time < b.SpawnRetryAt {
			return
		}
		var spawn *Point
		if gpdata.HasTag(typ, "ship") {
			spawn = g.findNavalSpawn(b, typ)
		} else {
			spawn = g.findSpawn(exit.X, exit.Y, typ)
		}
		if spawn == nil {
			b.SpawnRetryAt = g.Time + .3
			return
		}
		unit := g.AddUnit(b.Owner, typ, spawn.X, spawn.Y, false)
		gmetric(g, b.Owner, "unitsProduced", 1)
		// 出厂滑行 + 出口推进（对齐源 updateBuilding 尾段；exit 复用上方声明）。
		var goal Point
		if gpdata.HasTag(typ, "ship") {
			ng := g.navalGoal(spawn.X+exit.Side*exit.Distance, spawn.Y, unitRadiusOf(typ))
			goal = ng
		} else {
			fwdX := exit.Side
			fwdY := 0.0
			if exit.ForwardX != 0 || exit.ForwardY != 0 {
				fwdX, fwdY = exit.ForwardX, exit.ForwardY
			}
			if sp := g.findSpawn(spawn.X+fwdX*exit.Distance, spawn.Y+fwdY*exit.Distance, typ); sp != nil {
				goal = *sp
			} else {
				goal = Point{spawn.X, spawn.Y}
			}
		}
		unit.Deployment = &Deployment{BuildingID: b.ID, Start: g.Time, Until: g.Time + 2.6, FromX: spawn.X, FromY: spawn.Y, X: goal.X, Y: goal.Y}
		unit.Angle = exit.Angle
		unit.TurretAngle = exit.Angle
		b.ExitUntil = g.Time + 2.6
		// AI 初始指令（对齐源：engineer 占点 / scout 探桥 / jet 前压 / ship 出港 / 其他推进）。
		if g.IsAI(b.Owner) {
			switch {
			case typ == "engineer":
				// 就近可占点。
				var site *Site
				bestD := 1e18
				for _, ss := range append(append([]*Site{}, g.Oil...), g.Beacons...) {
					if ss.Owner != b.Owner && g.HasExploredFor(b.Owner, ss.X, ss.Y) {
						if dd := Distance(ss.X, ss.Y, unit.X, unit.Y); dd < bestD {
							bestD, site = dd, ss
						}
					}
				}
				if site != nil {
					unit.Order = &Order{Type: "capture", X: site.X, Y: site.Y}
				}
			case typ == "scout" || typ == "ghost":
				unit.Order = &Order{Type: "move", X: g.World.Width/2 + 70 + g.Random(-100, 100), Y: g.HomeY + g.Random(-280, 280)}
			case gpdata.HasTag(typ, "jet"):
				unit.Order = &Order{Type: "attackMove", X: 470 + g.Random(-110, 110), Y: g.HomeY + g.Random(-240, 240)}
			case gpdata.HasTag(typ, "ship"):
				unit.Order = &Order{Type: "attackMove", X: g.Map.Water.X1 + unitRadiusOf(typ) + 40, Y: g.HomeY + g.Random(-290, 290)}
			case typ != "harvester":
				unit.Order = &Order{Type: "move", X: g.World.Width - 630 + g.Random(-70, 70), Y: g.HomeY + g.Random(-120, 120)}
			}
		}
		if b.RallyPoint != nil && typ != "harvester" {
			goal2 := g.ResolveMoveGoal(unit, b.RallyPoint.X, b.RallyPoint.Y)
			unit.Order = &Order{Type: "move", X: goal2.X, Y: goal2.Y}
		}
		b.Active = nil
	}
}

// --- 移动/战斗（Wave 1f-2 完整实现，桩最小化）---

func (g *Game) moveUnit(u *Unit, goal Point, dt float64, stopDistance float64) {
	u.MotionGoal = &Point{goal.X, goal.Y}
	if g.CollectingMoves {
		g.PendingMoves = append(g.PendingMoves, pendingMove{unit: u, goal: Point{goal.X, goal.Y}, dt: dt, stopDistance: stopDistance})
		return
	}
	g.moveUnitNow(u, goal, dt, stopDistance)
}

func (g *Game) canOccupyUnit(u *Unit, p Point) bool {
	// 对齐 game.js L1040-1049。
	if p.X < 18 || p.X > g.World.Width-18 || p.Y < 18 || p.Y > g.World.Height-18 {
		return false
	}
	if gpdata.HasTag(u.Type, "air") {
		return true
	}
	layer := unitLayerOf(u.Type)
	if layer == "naval" || layer == "submerged" {
		return !g.isNavalBlocked(p.X, p.Y, maxF(18, unitRadiusOf(u.Type)))
	}
	radius := 14.0
	if gpdata.HasTag(u.Type, "infantry") {
		radius = 6
	}
	if g.isGroundBlocked(p.X, p.Y, radius) {
		return false
	}
	for _, b := range g.Buildings {
		if b.HP <= 0 {
			continue
		}
		minimum := b.Size*.45 + radius
		next := Distance(b.X, b.Y, p.X, p.Y)
		if next >= minimum {
			continue
		}
		previous := Distance(b.X, b.Y, u.X, u.Y)
		if previous >= minimum || next <= previous+.001 {
			return false
		}
	}
	return true
}

func (g *Game) isGroundBlocked(x, y, radius float64) bool {
	for _, r := range g.Map.Barriers {
		if intersectsRect(x, y, radius, r) {
			return true
		}
	}
	return false
}

func (g *Game) isNavalBlocked(x, y, radius float64) bool {
	w := g.Map.Water
	return w == nil || x-radius < w.X1 || x+radius > w.X2 || y-radius < w.Y1 || y+radius > w.Y2
}

func (g *Game) navalGoal(x, y, radius float64) Point {
	w := g.Map.Water
	if w == nil {
		return Point{x, y}
	}
	return Point{clamp(x, w.X1+radius, w.X2-radius), clamp(y, w.Y1+radius, w.Y2-radius)}
}

func unitLayerOf(typ string) string {
	switch {
	case hasTag(typ, "jet"):
		return "jet"
	case hasTag(typ, "drone"):
		return "drone"
	case hasTag(typ, "submerged"):
		return "submerged"
	case hasTag(typ, "ship"):
		return "naval"
	}
	return "ground"
}

func hasTag(typ, tag string) bool { return gpdata.HasTag(typ, tag) }

func unitRadiusOf(typ string) float64 {
	// 对齐 unit-spacing unitRadius（spacing 包非导出版本的本地镜像，避免环依赖）。
	if hasTag(typ, "infantry") {
		return 9
	}
	if hasTag(typ, "jet") {
		switch typ {
		case "airlift":
			return 80
		case "bomber":
			return 76
		case "freightPlane":
			return 68
		}
		return 40
	}
	if hasTag(typ, "drone") {
		return 20
	}
	if hasTag(typ, "ship") {
		switch typ {
		case "carrier":
			return 92
		case "destroyer":
			return 65
		case "submarine":
			return 46
		case "frigate":
			return 48
		case "patrol":
			return 36
		case "landing":
			return 74
		case "containerShip":
			return 88
		}
		return 48
	}
	if typ == "harvester" {
		return 29
	}
	return 30
}

func isProducer(typ string) bool {
	switch typ {
	case "barracks", "factory", "armory", "airfield", "dock":
		return true
	}
	return false
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func gmetric(g *Game, side int, key string, amount float64) {
	if g.Running && g.BattleStats != nil && side < len(g.BattleStats.Sides) && amount > 0 {
		g.BattleStats.Sides[side][key] += amount
	}
}

func lunarName(g *Game, typ string) string {
	d := gpdata.BUILDINGS[typ]
	if g.Map.Future && typ == "barracks" {
		return "机器人装配站"
	}
	return d.Name
}

func targetX(e any) float64 {
	switch v := e.(type) {
	case *Unit:
		return v.X
	case *Building:
		return v.X
	}
	return 0
}

func targetY(e any) float64 {
	switch v := e.(type) {
	case *Unit:
		return v.Y
	case *Building:
		return v.Y
	}
	return 0
}

func buildingAsTarget(b *Building) *Building { return b }

func (g *Game) updateFreight(u *Unit, dt float64) { g.updateFreightFull(u, dt) }

func (g *Game) updateUnit(u *Unit, dt float64) { g.updateUnitFull(u, dt) }
