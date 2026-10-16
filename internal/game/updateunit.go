package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// updateunit.go：updateUnit 完整主干（对齐 game.js L643-766 指令驱动循环），
// 替换 stubs.go 的骨架版。

func (g *Game) updateUnitFull(u *Unit, dt float64) {
	// 月表机器人能量（简化：电池递减 + 低电量返场）。
	if g.IsRobot(u.Type) {
		if g.Time < u.StunUntil {
			return
		}
		if u.Battery != nil {
			moving := g.Time-u.LastMovedAt < .15
			drain := 0.08
			if moving {
				drain = 0.22
			}
			*u.Battery = math.Max(0, *u.Battery-dt*drain)
			if *u.Battery <= 25 && (u.Order == nil || u.Order.Type != "rearm") {
				g.RequestResupplyFull(u, nil)
			}
			if u.Order != nil && u.Order.Type == "rearm" {
				return
			}
			if *u.Battery <= 0 {
				return
			}
		}
	}
	if u.Deployment != nil {
		if g.Time < u.StunUntil {
			return
		}
		if Distance(u.X, u.Y, u.Deployment.X, u.Deployment.Y) > 9 && g.Time < u.Deployment.Until {
			g.moveUnit(u, Point{u.Deployment.X, u.Deployment.Y}, dt, 8)
			return
		}
		u.Deployment = nil
		u.Path = nil
		u.PathTimer = 0
	}
	if u.TemporaryUntil != 0 && g.Time >= u.TemporaryUntil {
		u.HP = 0
		return
	}
	if freightTypes[u.Type] {
		g.updateFreightFull(u, dt)
		return
	}
	// 特殊指令（潜入/拆弹/回收）——对齐 game.js:652 updateSpecialOrder：
	// 在 updateUnit 内、玩家循环（buildQueue 扣费）之后执行，采集收入的入账
	// 时点因此晚一拍，直接影响 buildQueue 的 credits 判定（对拍 tick1335 铁证）。
	if g.Time >= u.StunUntil && g.updateSpecialOrder(u, dt) {
		return
	}
	if g.Time < u.StunUntil {
		return
	}
	jam := 1.0
	if u.JammedUntil > g.Time {
		jam = .4
	}
	u.FireTimer -= dt * jam
	d := gpdata.UNITS[u.Type]
	// 空勤（停机坪维修）与舰艇勤务。
	if gpdata.HasTag(u.Type, "air") && (u.Order == nil || u.Order.Type != "rearm") {
		g.serviceAircraft(u, dt)
	}
	if gpdata.HasTag(u.Type, "ship") && (u.Order == nil || u.Order.Type != "rearm") {
		g.serviceShip(u, dt, nil)
	}
	g.serviceIdleUnit(u, dt)
	if u.Order != nil && u.Order.Type == "board" {
		g.BoardTransport(u, dt)
		return
	}
	if u.Type == "supply" {
		g.updateSupply(u, dt)
		return
	}
	// 弹尽返场。
	if d.Ammo > 0 && !g.IsRobot(u.Type) && u.Ammo != nil && (*u.Ammo <= 0 || (u.Type == "carrier" && u.Wing != nil && *u.Wing <= 0)) && (u.Order == nil || u.Order.Type != "rearm") {
		u.ResumeOrder = u.Order
		u.Order = &Order{Type: "rearm"}
		u.Path = nil
	}
	if u.Order != nil && u.Order.Type == "rearm" {
		g.updateGroundRearm(u, dt)
		return
	}
	// 工程师 capture。
	if u.Type == "engineer" && u.Order != nil && u.Order.Type == "capture" {
		// 站点按距离重匹配（id 是字符串）。
		var site interface {
			pos() (float64, float64)
			own() int
		}
		_ = site
		// 油/信标：找最近可占（在指令坐标附近）。
		if g.Time >= u.StunUntil {
			cx, cy := u.Order.X, u.Order.Y
			trySites := append([]*Site{}, g.Oil...)
			trySites = append(trySites, g.Beacons...)
			var best *Site
			bestD := 38.0
			for _, s := range trySites {
				if s.Owner == u.Owner {
					continue
				}
				if dd := Distance(u.X, u.Y, s.X, s.Y); dd < bestD && Distance(cx, cy, s.X, s.Y) < 40 {
					bestD, best = dd, s
				}
			}
			if best != nil {
				best.Owner = u.Owner
				u.HP = 0
				g.RecalculatePower()
				g.FogTimer = 0
				g.Effects = append(g.Effects, &Effect{Type: "capture", X: best.X, Y: best.Y, Age: 0, Duration: 1.2, Owner: u.Owner})
			} else if bestD >= 38 {
				// 移向站点。
				if Distance(u.X, u.Y, cx, cy) > 26 {
					g.moveUnit(u, Point{cx, cy}, dt, 26)
					return
				}
			}
		}
		if u.Order != nil && u.Order.Type == "capture" && u.HP > 0 {
			// 未捕获成功保持待机（防止死循环推进）。
			u.Order = nil
		}
		return
	}
	// 无伤害单位：纯移动。
	if d.Damage == 0 {
		if g.patrolStep(u, dt) {
			return
		}
		if u.Order != nil && (u.Order.Type == "move" || u.Order.Type == "attackMove" || u.Order.Type == "attack") {
			if Distance(u.X, u.Y, u.Order.X, u.Order.Y) > 10 {
				g.moveUnit(u, Point{u.Order.X, u.Order.Y}, dt, 8)
			} else if u.Order.Type != "attack" {
				u.Order = nil
			} else {
				u.Order = nil
			}
		}
		return
	}
	// 战斗主干：目标获取 → 射程判定 → 开火 → 追击/拉开。
	var target targetEntity
	if u.Order != nil && u.Order.Type == "attack" {
		if e := g.GetUnit(u.Order.TargetID); e != nil {
			target = e
		} else if b := g.GetBuilding(u.Order.TargetID); b != nil {
			target = b
		}
	}
	if target != nil && (!g.CanSeeEntity(u.Owner, target) || !g.CanAttack(u, target)) {
		u.Order = &Order{Type: "attackMove", X: u.Order.X, Y: u.Order.Y}
		target = nil
	}
	if target != nil && target.getHP() <= 0 {
		if u.Order != nil && u.Order.Type == "attack" {
			u.Order = nil
		}
		target = nil
	}
	// 对齐 game.js:706：acquisition = order.type !== 'move' ? (artillery? max(sight,range): sight) : range*0.85。
	// 注意 order 为 nil 时 JS 的 `!== 'move'` 为真 → 走 sight 分支（警戒索敌），
	// 只有显式 move 指令才用 range*0.85（行军中被打了才还手）。
	acquisition := d.Sight
	if u.Order != nil && u.Order.Type == "move" {
		acquisition = d.Range * 0.85
	} else if gpdata.HasTag(u.Type, "artillery") {
		acquisition = math.Max(d.Sight, d.Range)
	}
	if target == nil {
		target = g.ClosestEnemy(u, acquisition, u.Owner)
	}
	if target != nil {
		r := g.UnitRange(u, target)
		reach := r
		if tb, ok := target.(*Building); ok {
			reach += tb.Size * 0.36
		}
		if u.Type == "bomber" {
			reach = 28
		}
		dist := Distance(u.X, u.Y, target.getX(), target.getY())
		clear := g.hasLineOfFire(u, target)
		moving := (u.Order != nil && u.Order.Type == "move") || g.Time-u.LastMovedAt < .12
		if d.DeployTime > 0 {
			if moving || dist > reach {
				u.DeployProgress = 0
			} else {
				u.DeployProgress = math.Min(d.DeployTime, u.DeployProgress+dt)
			}
		}
		flights := 0
		if u.Type == "carrier" {
			for _, p := range g.Projectiles {
				if !p.Finished && p.SourceID == u.ID && p.Kind == "wing" {
					flights++
				}
			}
		}
		minR := d.MinRange
		canFire := dist >= minR && dist <= reach && clear && u.FireTimer <= 0 && u.EWSuppressedUntil <= g.Time && !u.Overheated &&
			(d.DeployTime == 0 || u.DeployProgress >= d.DeployTime) && (u.Type != "carrier" || flights < int(*u.Wing))
		if canFire {
			u.TurretAngle = math.Atan2(target.getY()-u.Y, target.getX()-u.X)
			g.Fire(u, target, g.UnitDamage(u, target), u.Type)
			if d.Ammo > 0 && !g.IsRobot(u.Type) && u.Ammo != nil {
				*u.Ammo--
			}
			if u.Type == "laser" {
				g.laserHeat(u)
			}
			u.FireTimer = d.Cooldown
			if u.Type == "bomber" {
				g.RequestResupplyFull(u, nil)
				return
			}
		}
		if u.Order != nil && u.Order.Type == "move" {
			g.moveUnit(u, Point{u.Order.X, u.Order.Y}, dt, 8)
		} else if d.MinRange > 0 && dist < d.MinRange {
			angle := math.Atan2(u.Y-target.getY(), u.X-target.getX())
			goal := g.ResolveMoveGoal(u, clamp(u.X+math.Cos(angle)*100, 20, g.World.Width-20), clamp(u.Y+math.Sin(angle)*100, 20, g.World.Height-20))
			g.moveUnit(u, goal, dt, 8)
			u.DeployProgress = 0
		} else if !clear || dist > reach*.96 {
			stop := 8.0
			if clear {
				stop = reach * .85
			}
			g.moveUnit(u, Point{target.getX(), target.getY()}, dt, stop)
		}
		if u.Type != "bomber" && clear && dist <= reach*.96 && gpdata.HasTag(u.Type, "jet") && d.Damage > 0 {
			angle := math.Atan2(u.Y-target.getY(), u.X-target.getX()) + dt*.9
			g.moveUnit(u, g.ResolveMoveGoal(u, target.getX()+math.Cos(angle)*reach*.7, target.getY()+math.Sin(angle)*reach*.7), dt, 3)
		}
	} else if g.patrolStep(u, dt) {
		return
	} else if u.Order != nil && (u.Order.Type == "move" || u.Order.Type == "attackMove") {
		if Distance(u.X, u.Y, u.Order.X, u.Order.Y) > 10 {
			g.moveUnit(u, Point{u.Order.X, u.Order.Y}, dt, 8)
		} else {
			u.Order = nil
		}
	}
}

// patrolStep 对齐 game.js L327。
func (g *Game) patrolStep(u *Unit, dt float64) bool {
	if u.Order == nil || u.Order.Type != "patrol" {
		return false
	}
	order := u.Order
	if Distance(u.X, u.Y, order.X, order.Y) <= 16 {
		returning := math.Hypot(order.X-order.DestinationX, order.Y-order.DestinationY) < 1
		if returning {
			order.X, order.Y = order.OriginX, order.OriginY
		} else {
			order.X, order.Y = order.DestinationX, order.DestinationY
		}
		u.Path = nil
		u.PathTimer = 0
	}
	g.moveUnit(u, Point{order.X, order.Y}, dt, 8)
	return true
}

// updateGroundRearm 对齐 modern-combat.js:242（地面/舰艇返场整备）。
func (g *Game) updateGroundRearm(u *Unit, dt float64) {
	if g.IsRobot(u.Type) {
		return
	}
	naval := gpdata.HasTag(u.Type, "ship")
	if !naval {
		for _, s := range g.ActiveUnits(u.Owner, "supply") {
			if s.Stock != nil && *s.Stock >= 8 && Distance(s.X, s.Y, u.X, u.Y) < gpdata.UNITS["supply"].Range &&
				(s.Order == nil || (s.Order.Type != "move" && s.Order.Type != "attackMove")) {
				g.moveUnit(u, Point{s.X, s.Y}, dt, 75)
				return
			}
		}
	}
	var home *Building
	var homes []*Building
	for _, b := range g.OwnedBuildings(u.Owner, "") {
		if naval && b.Type == "dock" {
			homes = append(homes, b)
		} else if !naval && gpdata.HasTag(u.Type, "infantry") && b.Type == "barracks" {
			homes = append(homes, b)
		} else if !naval && !gpdata.HasTag(u.Type, "infantry") && (b.Type == "factory" || b.Type == "armory") {
			homes = append(homes, b)
		}
	}
	if u.Order != nil && u.Order.HomeID != 0 {
		for _, b := range homes {
			if b.ID == u.Order.HomeID {
				home = b
				break
			}
		}
	}
	if home == nil && len(homes) > 0 {
		home = homes[0]
		bestD := Distance(home.X, home.Y, u.X, u.Y)
		for _, b := range homes[1:] {
			if dd := Distance(b.X, b.Y, u.X, u.Y); dd < bestD {
				home, bestD = b, dd
			}
		}
	}
	if home == nil {
		return
	}
	var berth Point
	if naval {
		berth = g.shipBerth(u, home)
	} else {
		berth = Point{home.X, home.Y}
	}
	threshold := home.Size*.55 + 40
	if naval {
		threshold = 75
	}
	if Distance(u.X, u.Y, berth.X, berth.Y) > threshold {
		u.RearmProgress = 0
		stop := threshold - 16
		if naval {
			stop = 48
		}
		g.moveUnit(u, berth, dt, stop)
		return
	}
	if !g.HasPower(u.Owner) {
		return
	}
	if naval {
		g.serviceShip(u, dt, home)
	} else {
		g.serviceGroundUnit(u, dt, home)
	}
	if !g.ServiceReady(u) {
		return
	}
	g.refillStationAmmo(u, dt)
	d := gpdata.UNITS[u.Type]
	if u.Type == "carrier" && len(g.carrierAircraft(u)) == 0 && (u.Wing == nil || *u.Wing < d.Wing) && g.Players[u.Owner].Credits >= 150 {
		u.WingRearm += dt
		if u.WingRearm >= 8 {
			if u.Wing != nil {
				*u.Wing++
			}
			u.WingRearm = 0
			g.Players[u.Owner].Credits -= 150
		}
	}
	ammoFull := d.Ammo == 0 || (u.Ammo != nil && *u.Ammo == d.Ammo)
	if ammoFull && u.HP >= u.MaxHP-.01 && (u.Type != "carrier" || len(g.carrierAircraft(u)) > 0 || (u.Wing != nil && *u.Wing == d.Wing)) {
		g.RestoreCombatOrder(u)
	}
}

// updateSupply 对齐 modern-combat.js:280（补给车全逻辑）。
func (g *Game) updateSupply(u *Unit, dt float64) {
	p := g.Players[u.Owner]
	var home *Building
	factories := g.OwnedBuildings(u.Owner, "factory")
	if len(factories) > 0 {
		home = factories[0]
		for _, b := range factories[1:] {
			if Distance(b.X, b.Y, u.X, u.Y) < Distance(home.X, home.Y, u.X, u.Y) {
				home = b
			}
		}
	}
	if u.Stock == nil || *u.Stock < 8 {
		if home == nil {
			return
		}
		if u.Order == nil || u.Order.Type != "restock" {
			u.ResumeOrder = u.Order
			u.Order = &Order{Type: "restock"}
		}
	}
	if u.Order != nil && u.Order.Type == "restock" {
		if home == nil {
			return
		}
		if Distance(u.X, u.Y, home.X, home.Y) > home.Size*.55+40 {
			u.RearmProgress = 0
			g.moveUnit(u, Point{home.X, home.Y}, dt, home.Size*.55+24)
			return
		}
		if !g.HasPower(u.Owner) || p.Credits < 100 {
			return
		}
		u.RearmProgress += dt
		if u.RearmProgress >= 8 {
			p.Credits -= 100
			stock := gpdata.UNITS["supply"].Stock
			u.Stock = &stock
			g.RestoreCombatOrder(u)
		}
		return
	}
	if u.Order != nil && (u.Order.Type == "move" || u.Order.Type == "attackMove") {
		if Distance(u.X, u.Y, u.Order.X, u.Order.Y) > 10 {
			g.moveUnit(u, Point{u.Order.X, u.Order.Y}, dt, 8)
			return
		}
		u.Order = nil
	}
	// 自动补给：搜需服务的友军并靠近（对齐源 updateSupply 目标搜索段）。
	if (u.AutoSupply == nil || *u.AutoSupply) && (p.Credits > 0 || g.HasPower(u.Owner)) {
		var target *Unit
		if u.ServiceTargetID != 0 {
			if t := g.GetUnit(u.ServiceTargetID); t != nil && t.Owner == u.Owner && t.HP > 0 && t.EmbarkedIn == 0 {
				target = t
			}
		}
		if target == nil {
			var best *Unit
			bestD := 1e18
			for _, v := range g.ActiveUnits(u.Owner, "") {
				if v == u || gpdata.HasTag(v.Type, "air") || gpdata.HasTag(v.Type, "ship") || v.Type == "harvester" {
					continue
				}
				needs := (v.HP < v.MaxHP-.01) || (gpdata.UNITS[v.Type].Ammo > 0 && v.Ammo != nil && *v.Ammo < gpdata.UNITS[v.Type].Ammo)
				if !needs || g.Time-v.LastDamageAt <= 3 || g.Time-v.LastFireAt <= 3 {
					continue
				}
				// 别的补给车已锁定则跳过。
				taken := false
				for _, s := range g.ActiveUnits(u.Owner, "supply") {
					if s != u && s.ServiceTargetID == v.ID && (s.AutoSupply == nil || *s.AutoSupply) {
						taken = true
						break
					}
				}
				if taken {
					continue
				}
				if dd := Distance(u.X, u.Y, v.X, v.Y); dd < bestD {
					best, bestD = v, dd
				}
			}
			target = best
		}
		if target != nil {
			u.ServiceTargetID = target.ID
			if Distance(u.X, u.Y, target.X, target.Y) >= gpdata.UNITS["supply"].Range*.85 && g.Time-target.LastDamageAt > 3 && g.Time-target.LastFireAt > 3 {
				g.moveUnit(u, Point{target.X, target.Y}, dt, gpdata.UNITS["supply"].Range*.75)
				return
			}
		} else {
			u.ServiceTargetID = 0
		}
	}
	if g.Time-u.LastMovedAt < .5 {
		return
	}
	var nearby []*Unit
	for _, v := range g.ActiveUnits(u.Owner, "") {
		if v == u || gpdata.HasTag(v.Type, "air") || gpdata.HasTag(v.Type, "ship") || v.Type == "harvester" {
			continue
		}
		if Distance(u.X, u.Y, v.X, v.Y) < gpdata.UNITS["supply"].Range {
			nearby = append(nearby, v)
		}
	}
	var damaged *Unit
	for _, v := range nearby {
		if v.HP < v.MaxHP && g.Time-v.LastDamageAt > 3 {
			if damaged == nil || v.HP/v.MaxHP < damaged.HP/damaged.MaxHP {
				damaged = v
			}
		}
	}
	if damaged != nil {
		repair := 24.0 * dt
		if damaged.MaxHP-damaged.HP < repair {
			repair = damaged.MaxHP - damaged.HP
		}
		if p.Credits/.3 < repair {
			repair = p.Credits / .3
		}
		if u.Stock != nil && *u.Stock/.2 < repair {
			repair = *u.Stock / .2
		}
		damaged.HP += repair
		p.Credits -= repair * .3
		if u.Stock != nil {
			*u.Stock -= repair * .2
		}
		gmetric(g, damaged.Owner, "repairHP", repair)
	}
	var empty *Unit
	for _, v := range nearby {
		if !g.IsRobot(v.Type) && gpdata.UNITS[v.Type].Ammo > 0 && v.Ammo != nil && *v.Ammo < gpdata.UNITS[v.Type].Ammo {
			empty = v
			break
		}
	}
	if empty != nil && u.ServiceTimer <= 0 && u.Stock != nil && *u.Stock >= 8 && p.Credits >= 10 {
		*u.Stock -= 8
		p.Credits -= 10
		*empty.Ammo++
		u.ServiceTimer = 1.5
	}
	u.ServiceTimer = math.Max(0, u.ServiceTimer-dt)
}
