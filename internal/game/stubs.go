package game

// stubs.go：update 主循环引用的 mixin 桩——各分批（1f-2..4/1d）逐个替换为完整实现。
// 每个桩保持源签名语义，先让主循环可编译可测，避免一次性落地 1000 行无法定位回归。

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// --- intelligence（Wave 1f-3 完整实现，先空转）---

func (g *Game) UpdateIntelligence(dt float64) {}

// --- modern-combat（Wave 1d 完整实现，先空转）---

func (g *Game) updateElectronicWarfare(dt float64) {}
func (g *Game) updateProjectiles(dt float64)       {}

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
			target := g.closestEnemy(buildingAsTarget(b), 250, b.Owner)
			if target != nil {
				b.Angle = atan2(targetY(target)-b.Y, targetX(target)-b.X)
				g.fire(buildingAsTarget(b), target, 26, "turret")
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
		b.Active = nil
		_ = unit // 出厂滑行/AI 初始指令在 1f-4 完整化。
	}
}

// --- 移动/战斗（Wave 1f-2 完整实现，桩最小化）---

func (g *Game) updateUnit(u *Unit, dt float64) {
	// 对齐 game.js L643 起的巨型方法；分批实现。
	// 1f-2 先处理：deployment 滑行、临时单位到期、freight、基础 fireTimer。
	if u.Deployment != nil {
		if g.Time >= u.Deployment.Until || Distance(u.X, u.Y, u.Deployment.X, u.Deployment.Y) <= 9 {
			u.Deployment = nil
			u.Path = nil
			u.PathTimer = 0
		} else if g.Time >= u.StunUntil {
			g.moveUnit(u, Point{u.Deployment.X, u.Deployment.Y}, dt, 8)
			return
		}
	}
	if u.TemporaryUntil != 0 && g.Time >= u.TemporaryUntil {
		u.HP = 0
		return
	}
	if u.Type == "freightPlane" || u.Type == "containerShip" {
		g.updateFreight(u, dt)
		return
	}
	if g.Time < u.StunUntil {
		return
	}
	jamFactor := 1.0
	if u.JammedUntil > g.Time {
		jamFactor = .4
	}
	u.FireTimer -= dt * jamFactor
	// 战斗/指令分支在 1f-2/1f-3 完整化。
	_ = jamFactor
}

func (g *Game) moveUnit(u *Unit, goal Point, dt float64, stopDistance float64) {
	u.MotionGoal = &Point{goal.X, goal.Y}
	if g.CollectingMoves {
		g.PendingMoves = append(g.PendingMoves, pendingMove{unit: u, goal: Point{goal.X, goal.Y}, dt: dt, stopDistance: stopDistance})
		return
	}
	g.moveUnitNow(u, goal, dt, stopDistance)
}

func (g *Game) moveUnitNow(u *Unit, goal Point, dt float64, stopDistance float64) {
	// 1f-2 完整实现（air/ship/地面寻路三分支）。
	if Distance(u.X, u.Y, goal.X, goal.Y) <= stopDistance {
		return
	}
	if gpdata.HasTag(u.Type, "air") {
		g.moveWithAvoidance(u, goal, dt, stopDistance)
		return
	}
	if gpdata.HasTag(u.Type, "ship") {
		dest := g.navalGoal(goal.X, goal.Y, unitRadiusOf(u.Type))
		g.moveWithAvoidance(u, dest, dt, stopDistance)
		return
	}
	// 地面：寻路（1f-2 接 pathfind）。
	g.moveWithAvoidance(u, goal, dt, stopDistance)
}

func (g *Game) moveWithAvoidance(u *Unit, point Point, dt float64, stopDistance float64) bool {
	// 对齐 game.js L1051-1081 十一方向避让；1f-2 完整评分，先直线。
	d := Distance(u.X, u.Y, point.X, point.Y)
	speed := gpdata.UNITS[u.Type].Speed
	amount := speed * dt
	if u.JammedUntil > g.Time {
		amount *= .6
	}
	if max := d - stopDistance; amount > max {
		amount = max
	}
	if amount <= .001 {
		return true
	}
	angle := atan2(point.Y-u.Y, point.X-u.X)
	next := Point{u.X + cos(angle)*amount, u.Y + sin(angle)*amount}
	if !g.canOccupyUnit(u, next) {
		return false
	}
	u.X, u.Y = next.X, next.Y
	u.Angle = angle
	u.MovePulse += dt * 7
	u.LastMovedAt = g.Time
	u.DeployProgress = 0
	return true
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

func (g *Game) updateFreight(u *Unit, dt float64) {
	// 完整版在 logistics 深化；最小闭环：入境→出境。
	f := u.Freight
	if f == nil {
		u.HP = 0
		return
	}
	if g.Time < u.StunUntil {
		return
	}
	if f.Phase == "outbound" {
		exit := f.Entry
		if u.Type == "freightPlane" {
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
		g.moveUnit(u, exit, dt, 18)
	}
}

func (g *Game) applyHit(h pendingHit) {
	// 1f-3 完整化。
}

func (g *Game) fire(source any, target any, amount float64, style string) {
	// 1f-3 完整化。
}

func (g *Game) closestEnemy(source any, rangeLimit float64, owner int) any {
	// 1f-3 完整化。
	return nil
}

// --- AI（Wave 1f-4）---

func (g *Game) updateAI(side int)    {}
func (g *Game) launchAIWave(side int) {}

func (g *Game) placeAIBuilding(typ string, side int) {}

// --- 工具 ---

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
