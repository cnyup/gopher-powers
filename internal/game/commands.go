package game

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// commands.go 对齐 game.js 的玩家指令面（build/place/train/repair/sell/command/ability）
// 与 intelligence.js isControlLocked。

// IsControlLocked 对齐 intelligence.js:10-13。
func (g *Game) IsControlLocked(side int) bool {
	p := g.Players[side]
	if p.CyberLockedUntil <= g.Time {
		return false
	}
	src := g.GetBuilding(p.CyberSourceID)
	return src != nil && src.Kind == "building" && src.Owner != side && src.Type == "super" && src.HP > 0 && g.HasPower(src.Owner)
}

// CanBuild 对齐 game.js L215-219。
func (g *Game) CanBuild(side int, typ string) bool {
	d, ok := gpdata.BUILDINGS[typ]
	if !ok {
		return false
	}
	if d.Naval && g.Map.Water == nil {
		return false
	}
	if !g.HasBuilding(side, "hq") {
		return false
	}
	if d.Requires != "" && !g.HasBuilding(side, d.Requires) {
		return false
	}
	return true
}

// StartBuild 对齐 game.js L220-229。
func (g *Game) StartBuild(side int, typ string) bool {
	if g.IsControlLocked(side) {
		return false
	}
	p := g.Players[side]
	if !g.CanBuild(side, typ) || p.BuildQueue != nil || p.ReadyBuilding != "" || (side == 0 && g.PendingBuilding != "") {
		return false
	}
	p.BuildQueue = &BuildQueue{Type: typ, Progress: 0, Paid: 0}
	if side == 0 {
		if g.Events.Notice != nil {
			g.Events.Notice(lunarName(g, typ) + "开始建造")
		}
		if g.Events.Voice != nil {
			g.Events.Voice("construction")
		}
	}
	return true
}

// CancelBuilding 对齐 game.js L230-245。
func (g *Game) CancelBuilding(side int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	p := g.Players[side]
	refund := 0.0
	if p.BuildQueue != nil {
		refund = p.BuildQueue.Paid
	} else if p.ReadyBuilding != "" {
		refund = gpdata.BUILDINGS[p.ReadyBuilding].Cost
	} else if side == 0 && g.PendingBuilding != "" {
		refund = gpdata.BUILDINGS[g.PendingBuilding].Cost
	}
	if p.BuildQueue == nil && p.ReadyBuilding == "" && !(side == 0 && g.PendingBuilding != "") {
		return false
	}
	p.Credits += refund
	p.BuildQueue = nil
	p.ReadyBuilding = ""
	if side == 0 {
		g.PendingBuilding = ""
		g.PlacingBuilding = false
		if g.Events.Notice != nil {
			g.Events.Notice("建造已取消，返还 " + itoa(int(refund)) + " 资金")
		}
	}
	return true
}

// CancelUnitProduction 对齐 game.js L246-258。
func (g *Game) CancelUnitProduction(side int, buildingID int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	b := g.GetBuilding(buildingID)
	if b == nil || b.Owner != side || !isProducer(b.Type) {
		return false
	}
	if b.Active == nil && len(b.Queue) == 0 {
		return false
	}
	refund := 0.0
	if b.Active != nil {
		refund = b.Active.Paid
		b.Active = nil
	} else {
		b.Queue = b.Queue[1:]
	}
	g.Players[side].Credits += refund
	if side == 0 && g.Events.Notice != nil {
		g.Events.Notice("生产已取消，返还 " + itoa(int(refund)) + " 资金")
	}
	return true
}

// CanPlace 对齐 game.js L259-272。
func (g *Game) CanPlace(side int, typ string, x, y float64) bool {
	size := gpdata.BUILDINGS[typ].Size
	if x < size || y < size || x > g.World.Width-size || y > g.World.Height-size {
		return false
	}
	if typ == "dock" {
		hit := false
		for _, r := range g.Map.DockZones {
			if x >= r.X1 && x <= r.X2 && y >= r.Y1 && y <= r.Y2 {
				if (side == 0 && x < g.World.Width/2) || (side != 0 && x > g.World.Width/2) {
					hit = true
				}
			}
		}
		if !hit {
			return false
		}
	} else {
		for _, r := range g.Map.NoBuild {
			if intersectsRect(x, y, size*0.6, r) {
				return false
			}
		}
	}
	for _, o := range g.Ore {
		if o.Amount > 0 && Distance(o.X, o.Y, x, y) < 95 {
			return false
		}
	}
	for _, o := range g.Oil {
		if Distance(o.X, o.Y, x, y) < 95 {
			return false
		}
	}
	for _, s := range g.Beacons {
		if Distance(s.X, s.Y, x, y) < 100 {
			return false
		}
	}
	for _, s := range g.ResourceSites {
		if Distance(s.X, s.Y, x, y) < 80 {
			return false
		}
	}
	for _, b := range g.Buildings {
		if b.HP > 0 && Distance(b.X, b.Y, x, y) < (b.Size+size)*0.63+20 {
			return false
		}
	}
	if typ == "dock" {
		return g.HasBuilding(side, "factory")
	}
	for _, b := range g.OwnedBuildings(side, "") {
		if Distance(b.X, b.Y, x, y) < 310 {
			return true
		}
	}
	for _, s := range g.ResourceSites {
		if s.Type == "depot" && s.Owner == side && Distance(s.X, s.Y, x, y) < 260 {
			return true
		}
	}
	return false
}

// PlaceBuilding 对齐 game.js L273-288。
func (g *Game) PlaceBuilding(side int, typ string, x, y float64) bool {
	if g.IsControlLocked(side) {
		return false
	}
	if g.Multiplayer && g.Players[side].ReadyBuilding != typ {
		return false
	}
	if !g.CanPlace(side, typ, x, y) {
		return false
	}
	g.AddBuilding(side, typ, x, y)
	g.Players[side].ReadyBuilding = ""
	if side == 0 {
		g.PendingBuilding = ""
		g.PlacingBuilding = false
		if g.Events.Notice != nil {
			g.Events.Notice(lunarName(g, typ) + "部署完成")
		}
		if g.Events.Voice != nil {
			g.Events.Voice("deployed")
		}
	}
	g.Effects = append(g.Effects, &Effect{Type: "build", X: x, Y: y, Age: 0, Duration: 0.8, Owner: side})
	return true
}

// ToggleRepair 对齐 game.js L289-298。
func (g *Game) ToggleRepair(side int, id int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	b := g.GetBuilding(id)
	if b == nil || b.Owner != side || b.HP <= 0 {
		return false
	}
	if b.HP >= b.MaxHP && !b.Repairing {
		return false
	}
	b.Repairing = !b.Repairing && b.HP < b.MaxHP
	if side == 0 && g.Events.Notice != nil {
		if b.Repairing {
			g.Events.Notice("正在维修" + lunarName(g, b.Type))
		} else {
			g.Events.Notice("已停止维修")
		}
	}
	return true
}

// QueueUnit 对齐 game.js L340-352。
func (g *Game) QueueUnit(side int, typ string) bool {
	if g.IsControlLocked(side) || g.Players[side].Eliminated {
		return false
	}
	if g.PlayerCount == 6 && g.unitCapacityUsed(side) >= 40 {
		return false
	}
	d, ok := gpdata.UNITS[typ]
	if !ok || freightTypes[typ] || (typ == "harvester" && g.EconomyMode != "mining") ||
		(d.Naval && g.Map.Water == nil) || !gpdata.SupportsMap(d.Map, g.MapID) ||
		(d.Faction != "" && g.Players[side].Faction != d.Faction) ||
		(d.Requires != "" && !g.HasBuilding(side, d.Requires)) {
		return false
	}
	var producer *Building
	for _, b := range g.OwnedBuildings(side, d.Producer) {
		if producer == nil || len(b.Queue) < len(producer.Queue) {
			producer = b
		}
	}
	if producer == nil || len(producer.Queue) >= 5 {
		return false
	}
	producer.Queue = append(producer.Queue, typ)
	if side == 0 {
		if g.Events.Notice != nil {
			g.Events.Notice(equipName(g, typ) + "已加入队列")
		}
		if g.Events.Voice != nil {
			g.Events.Voice("queued")
		}
	}
	return true
}

func (g *Game) unitCapacityUsed(side int) int {
	n := len(g.OwnedUnits(side, ""))
	for _, b := range g.OwnedBuildings(side, "") {
		n += len(b.Queue)
		if b.Active != nil {
			n++
		}
	}
	return n
}

// CastAbility 对齐 game.js L438-478。
func (g *Game) CastAbility(x, y float64, side int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	p := g.Players[side]
	if g.PlayerCount == 6 && p.Faction == "asia" && g.unitCapacityUsed(side) > 35 {
		return false
	}
	if !g.HasBuilding(side, "super") || !g.HasPower(side) || p.AbilityCharge < 100 || p.AbilityCooldown > 0 || p.Credits < 650 {
		return false
	}
	if p.Faction != "china" && !g.IsVisibleFor(side, x, y) {
		return false
	}
	p.Credits -= 650
	p.AbilityCharge = 0
	p.AbilityCooldown = 45
	if side == 0 && g.Events.Voice != nil {
		g.Events.Voice("ability")
	}
	faction := p.Faction
	g.Effects = append(g.Effects, &Effect{Type: "ability", Faction: faction, X: x, Y: y, Age: 0, Duration: 2.5, Owner: side})
	switch faction {
	case "china":
		p.ShieldUntil = g.Time + 12
		if side == 0 && g.Events.Notice != nil {
			g.Events.Notice("协同电子防护启动：友军获得 12 秒减伤")
		}
	case "middleeast":
		for _, u := range g.EnemyUnits(side, "") {
			if Distance(u.X, u.Y, x, y) < 290 && !gpdata.HasTag(u.Type, "infantry") {
				u.StunUntil = g.Time + 8
			}
		}
		for _, b := range g.EnemyBuildings(side, "turret") {
			if Distance(b.X, b.Y, x, y) < 290 {
				b.StunUntil = g.Time + 8
			}
		}
		if side == 0 && g.Events.Notice != nil {
			g.Events.Notice("区域脉冲命中，敌方电子装备已瘫痪")
		}
	case "asia":
		for i := 0; i < 5 && (g.PlayerCount != 6 || len(g.OwnedUnits(side, "")) < 40); i++ {
			ux := clamp(x+g.Random(-110, 110), 50, g.World.Width-50)
			uy := clamp(y+g.Random(-110, 110), 50, g.World.Height-50)
			u := g.AddUnit(side, "drone", ux, uy, true)
			goal := Point{X: g.World.Width - 205, Y: g.HomeY}
			if side != 0 {
				goal = Point{X: 205, Y: g.HomeY}
			}
			if g.PlayerCount == 6 {
				goal = g.enemyObjective(side)
			}
			u.Order = &Order{Type: "attackMove", X: goal.X, Y: goal.Y}
		}
		if side == 0 && g.Events.Notice != nil {
			g.Events.Notice("蜂群已抵达目标区域")
		}
	default: // russia / nato
		for _, target := range g.allTargets() {
			d := Distance(target.getX(), target.getY(), x, y)
			if d > 245 {
				continue
			}
			amount := 165.0
			if faction == "russia" {
				amount = 190
			}
			if faction == "russia" && target.getKind() == "building" {
				amount *= 1.22
			}
			if faction == "nato" && target.getKind() == "unit" && gpdata.HasTag(target.getType(), "armor") {
				amount *= 0.48
			}
			amount *= 1 - d/430
			if target.getType() == "hq" {
				b := target.(*Building)
				if amount > b.MaxHP*0.22 {
					amount = b.MaxHP * 0.22
				}
			}
			g.Damage(target, amount, side)
		}
		if side == 0 && g.Events.Notice != nil {
			g.Events.Notice(gpdata.FACTIONS[faction].Ability + "已命中目标区域")
		}
	}
	if side == 0 {
		g.PendingAbility = false
	} else if g.Events.Notice != nil {
		g.Events.Notice("敌方发动了" + gpdata.FACTIONS[faction].Ability)
	}
	return true
}

// enemyObjective 对齐 game.js L147。
func (g *Game) enemyObjective(side int) Point {
	// 可见敌 HQ 优先，否则出生点。
	for _, b := range g.EnemyBuildings(side, "hq") {
		if g.CanSeeEntity(side, b) {
			return Point{b.X, b.Y}
		}
	}
	if g.Map.Spawns != nil {
		for _, s := range g.Map.Spawns {
			if s.Side != side {
				return Point{s.X, s.Y}
			}
		}
	}
	if side != 0 {
		return Point{300, g.HomeY}
	}
	return Point{g.World.Width - 300, g.HomeY}
}

func equipName(g *Game, typ string) string {
	if p := equipProfileOf(g, typ); p != nil {
		return p.Name
	}
	return gpdata.UNITS[typ].Name
}
