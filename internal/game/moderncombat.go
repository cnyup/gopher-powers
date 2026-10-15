package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/rules"
)

// moderncombat.go 对齐 src/modern-combat.js（496 行 mixin）+ projectile-flight.js turnToward。

// turnToward 对齐 projectile-flight.js:15-18。
func turnToward(current, target, maxTurn float64) float64 {
	delta := math.Atan2(math.Sin(target-current), math.Cos(target-current))
	return current + clamp(delta, -maxTurn, maxTurn)
}

// ServiceReady 对齐 modern-combat.js:15-17。
func (g *Game) ServiceReady(u *Unit) bool {
	return g.HasPower(u.Owner) && g.Time-u.LastDamageAt > 3 && g.Time-u.LastMovedAt >= .5
}

// aircraftGroundedFull 对齐 modern-combat.js:19-23（机场停泊判定）。
func (g *Game) aircraftGroundedFull(u *Unit) bool {
	if !gpdata.HasTag(u.Type, "air") || u.EmbarkedIn != 0 || u.Deployment != nil || u.DeckApproach != nil {
		return false
	}
	if u.Order != nil {
		switch u.Order.Type {
		case "move", "attackMove", "attack", "patrol", "board":
			return false
		}
	}
	if g.Time-u.LastMovedAt < .5 {
		return false
	}
	for _, b := range g.OwnedBuildings(u.Owner, "airfield") {
		if Distance(b.X, b.Y, u.X, u.Y) <= b.Size*.55+24 {
			return true
		}
	}
	return false
}

// serviceGroundUnit 对齐 modern-combat.js:25-37。
func (g *Game) serviceGroundUnit(u *Unit, dt float64, home *Building) bool {
	if !g.ServiceReady(u) || g.IsRobot(u.Type) || gpdata.HasTag(u.Type, "air") || gpdata.HasTag(u.Type, "ship") || gpdata.HasTag(u.Type, "logistics") {
		return false
	}
	infantry := gpdata.HasTag(u.Type, "infantry")
	if home == nil {
		for _, b := range g.OwnedBuildings(u.Owner, "") {
			if infantry && b.Type != "barracks" {
				continue
			}
			if !infantry && b.Type != "factory" && b.Type != "armory" {
				continue
			}
			if Distance(b.X, b.Y, u.X, u.Y) <= b.Size*.55+40 {
				home = b
				break
			}
		}
	}
	if home == nil || home.HP <= 0 || home.Owner != u.Owner || Distance(home.X, home.Y, u.X, u.Y) > home.Size*.55+40 {
		return false
	}
	p := g.Players[u.Owner]
	rate := 22.0
	if infantry {
		rate = 12
	}
	repaired := rate * dt
	if u.MaxHP-u.HP < repaired {
		repaired = u.MaxHP - u.HP
	}
	if p.Credits/.3 < repaired {
		repaired = p.Credits / .3
	}
	u.HP += repaired
	p.Credits = math.Max(0, p.Credits-repaired*.3)
	gmetric(g, u.Owner, "repairHP", repaired)
	if repaired > 0 && g.Time >= u.ServiceFXAt {
		u.ServiceFXAt = g.Time + .7
		toX, toY := u.X, u.Y
		g.Effects = append(g.Effects, &Effect{Type: "shot", Style: "repair", X: home.X, Y: home.Y, ToX: &toX, ToY: &toY, TargetType: u.Type, Owner: u.Owner, Age: 0, Duration: .6})
	}
	return true
}

// refillStationAmmo 对齐 modern-combat.js:39-47。
func (g *Game) refillStationAmmo(u *Unit, dt float64) {
	d := gpdata.UNITS[u.Type]
	if d.Ammo == 0 || g.IsRobot(u.Type) || (u.Ammo != nil && *u.Ammo >= d.Ammo) || !g.ServiceReady(u) {
		return
	}
	u.RearmProgress += dt
	cost := 10.0
	if d.AmmoCost != 0 {
		cost = d.AmmoCost
	}
	if u.RearmProgress >= d.RearmTime/d.Ammo && g.Players[u.Owner].Credits >= cost {
		g.Players[u.Owner].Credits -= cost
		if u.Ammo == nil {
			a := 0.0
			u.Ammo = &a
		}
		*u.Ammo++
		u.RearmProgress = 0
	}
}

// serviceIdleUnit 对齐 modern-combat.js:49-55。
func (g *Game) serviceIdleUnit(u *Unit, dt float64) {
	if u.Order != nil || !g.ServiceReady(u) {
		return
	}
	var atStation bool
	switch {
	case gpdata.HasTag(u.Type, "air"):
		atStation = g.aircraftGroundedFull(u)
	case gpdata.HasTag(u.Type, "ship"):
		for _, b := range g.OwnedBuildings(u.Owner, "dock") {
			goal := g.navalGoal(b.X, b.Y, unitRadiusOf(u.Type)+10)
			if Distance(u.X, u.Y, goal.X, goal.Y) <= 115 {
				atStation = true
				break
			}
		}
	default:
		atStation = g.serviceGroundUnit(u, dt, nil)
	}
	if atStation {
		g.refillStationAmmo(u, dt)
	}
}

// serviceAircraft 对齐 modern-combat.js:57-68。
func (g *Game) serviceAircraft(u *Unit, dt float64) {
	if u.HP >= u.MaxHP || g.Time-u.LastDamageAt <= 3 || g.Time-u.LastMovedAt < .5 || !g.HasPower(u.Owner) {
		return
	}
	var home *Building
	for _, b := range g.OwnedBuildings(u.Owner, "airfield") {
		if Distance(b.X, b.Y, u.X, u.Y) <= b.Size*.55+30 {
			home = b
			break
		}
	}
	if home == nil {
		return
	}
	p := g.Players[u.Owner]
	repaired := 22.0 * dt
	if u.MaxHP-u.HP < repaired {
		repaired = u.MaxHP - u.HP
	}
	if p.Credits/.3 < repaired {
		repaired = p.Credits / .3
	}
	u.HP += repaired
	p.Credits = math.Max(0, p.Credits-repaired*.3)
	gmetric(g, u.Owner, "repairHP", repaired)
	if repaired > 0 && g.Time >= u.ServiceFXAt {
		u.ServiceFXAt = g.Time + .7
		toX, toY := u.X, u.Y
		g.Effects = append(g.Effects, &Effect{Type: "shot", Style: "repair", X: home.X, Y: home.Y, ToX: &toX, ToY: &toY, TargetType: u.Type, Owner: u.Owner, Age: 0, Duration: .6})
	}
}

// RequestResupplyFull 对齐 modern-combat.js:69-74（真实现替换 ai.go 桩）。
func (g *Game) RequestResupplyFull(u *Unit, home *Building) bool {
	if u == nil || u.HP <= 0 || u.EmbarkedIn != 0 || u.Type == "harvester" || u.Type == "supply" || gpdata.HasTag(u.Type, "logistics") {
		return false
	}
	if u.Order == nil || u.Order.Type != "rearm" {
		u.ResumeOrder = u.Order
		u.Order = &Order{Type: "rearm"}
		u.Path = nil
		u.PathTimer = 0
	}
	if home != nil && home.Type == "dock" && home.HP > 0 && home.Owner == u.Owner && gpdata.HasTag(u.Type, "ship") {
		u.Order.HomeID = home.ID
	}
	return true
}

// submarineSurfacedFull 对齐 modern-combat.js:76-81。
func (g *Game) submarineSurfacedFull(u *Unit) bool {
	if u.Type != "submarine" || u.EmbarkedIn != 0 || u.HP <= 0 {
		return false
	}
	if u.Order != nil && u.Order.Type != "rearm" {
		return false
	}
	for _, home := range g.OwnedBuildings(u.Owner, "dock") {
		var bx, by float64
		if u.Order != nil && u.Order.Type == "rearm" {
			p := g.shipBerth(u, home)
			bx, by = p.X, p.Y
		} else {
			p := g.navalGoal(home.X, home.Y, unitRadiusOf(u.Type)+10)
			bx, by = p.X, p.Y
		}
		if u.Order == nil || u.Order.HomeID == 0 || u.Order.HomeID == home.ID {
			if Distance(u.X, u.Y, bx, by) <= 95 {
				return true
			}
		}
	}
	return false
}

// shipBerth 对齐 modern-combat.js:83-88（泊位分配）。
func (g *Game) shipBerth(u *Unit, home *Building) Point {
	var peers []*Unit
	for _, v := range g.ActiveUnits(u.Owner, "") {
		if gpdata.HasTag(v.Type, "ship") && v.Order != nil && v.Order.Type == "rearm" &&
			(v.Order.HomeID == 0 || v.Order.HomeID == home.ID) {
			peers = append(peers, v)
		}
	}
	// 按 id 升序。
	for i := 1; i < len(peers); i++ {
		for j := i; j > 0 && peers[j].ID < peers[j-1].ID; j-- {
			peers[j], peers[j-1] = peers[j-1], peers[j]
		}
	}
	slot := 0
	for i, v := range peers {
		if v.ID == u.ID {
			slot = i
			break
		}
	}
	offset := 0.0
	if slot != 0 {
		off := float64((slot + 1) / 2) * 215
		if slot%2 == 1 {
			offset = off
		} else {
			offset = -off
		}
	}
	dir := 1.0
	if home.Owner != 0 {
		dir = -1
	}
	extra := 0.0
	if slot >= 7 {
		extra = 210
	}
	return g.navalGoal(home.X+dir*extra, home.Y+offset, unitRadiusOf(u.Type)+10)
}

// serviceShip 对齐 modern-combat.js:90-102。
func (g *Game) serviceShip(u *Unit, dt float64, home *Building) {
	if u.HP >= u.MaxHP || g.Time-u.LastDamageAt <= 3 || g.Time-u.LastMovedAt < .5 || !g.HasPower(u.Owner) {
		return
	}
	if home == nil {
		for _, b := range g.OwnedBuildings(u.Owner, "dock") {
			goal := g.navalGoal(b.X, b.Y, unitRadiusOf(u.Type)+10)
			if Distance(u.X, u.Y, goal.X, goal.Y) <= 115 {
				home = b
				break
			}
		}
	}
	if home == nil || home.Owner != u.Owner || home.Type != "dock" || home.HP <= 0 {
		return
	}
	berth := g.shipBerth(u, home)
	if Distance(u.X, u.Y, berth.X, berth.Y) > 95 {
		goal := g.navalGoal(home.X, home.Y, unitRadiusOf(u.Type)+10)
		if Distance(u.X, u.Y, goal.X, goal.Y) > 115 {
			return
		}
	}
	p := g.Players[u.Owner]
	repair := 30.0 * dt
	if u.MaxHP-u.HP < repair {
		repair = u.MaxHP - u.HP
	}
	if p.Credits/.35 < repair {
		repair = p.Credits / .35
	}
	u.HP += repair
	p.Credits = math.Max(0, p.Credits-repair*.35)
	gmetric(g, u.Owner, "repairHP", repair)
	if repair > 0 && g.Time >= u.ServiceFXAt {
		u.ServiceFXAt = g.Time + .7
		toX, toY := u.X, u.Y
		g.Effects = append(g.Effects, &Effect{Type: "shot", Style: "repair", X: home.X, Y: home.Y, ToX: &toX, ToY: &toY, TargetType: u.Type, Owner: u.Owner, Age: 0, Duration: .6})
	}
}

// updateElectronicWarfareFull 对齐 modern-combat.js:104-138。
func (g *Game) updateElectronicWarfareFull(dt float64) {
	storm := rules.WeatherState(g.Map, g.Time).Phase == "storm"
	for _, u := range g.Units {
		if storm && u.HP > 0 && u.EmbarkedIn == 0 && gpdata.HasTag(u.Type, "air") && !g.hasSignalCover(u.Owner, u.X, u.Y) {
			u.JammedUntil = g.Time + .3
		}
		u.Heat = math.Max(0, u.Heat-dt*18)
		if u.Overheated && u.Heat < 28 {
			u.Overheated = false
		}
		if u.EmbarkedIn != 0 {
			if t := g.GetUnit(u.EmbarkedIn); t != nil && t.HP > 0 {
				u.X, u.Y = t.X, t.Y
			}
		}
		sonar := gpdata.UNITS[u.Type].Sonar
		if u.HP > 0 && u.EmbarkedIn == 0 && sonar > 0 && g.Time >= u.SonarFXAt {
			u.SonarFXAt = g.Time + 4
			size := sonar
			g.Effects = append(g.Effects, &Effect{Type: "sonar", SourceID: u.ID, X: u.X, Y: u.Y, Size: &size, Owner: u.Owner, Age: 0, Duration: 1.5})
		}
	}
	// 电子压制机：每 10 秒活跃 2.5 秒。
	for _, plane := range g.Units {
		if plane.HP <= 0 || plane.EmbarkedIn != 0 || plane.Type != "ewPlane" || g.Time < plane.StunUntil ||
			(plane.Order != nil && plane.Order.Type == "rearm") || g.aircraftGroundedFull(plane) {
			continue
		}
		if math.Mod(g.Time, 10) >= 2.5 {
			continue
		}
		for _, target := range g.EnemyUnits(plane.Owner, "") {
			if (target.Type == "aa" || target.Type == "frigate" || target.Type == "destroyer") &&
				Distance(plane.X, plane.Y, target.X, target.Y) <= gpdata.UNITS["ewPlane"].Range && g.CanSeeEntity(plane.Owner, target) {
				target.EWSuppressedUntil = g.Time + .15
			}
		}
		if g.Time >= plane.PulseAt {
			plane.PulseAt = g.Time + 1
			size := gpdata.UNITS["ewPlane"].Range
			g.Effects = append(g.Effects, &Effect{Type: "jam", X: plane.X, Y: plane.Y, Size: &size, Owner: plane.Owner, Age: 0, Duration: .8})
		}
	}
	// 电子干扰车。
	for _, jammer := range g.Units {
		if jammer.HP <= 0 || jammer.EmbarkedIn != 0 || jammer.Type != "jammer" || g.Time < jammer.StunUntil {
			continue
		}
		r := gpdata.UNITS["jammer"].Range
		if g.Players[jammer.Owner].Faction == "middleeast" {
			r *= 1.15
		}
		for _, u := range g.EnemyUnits(jammer.Owner, "") {
			if gpdata.HasTag(u.Type, "drone") && Distance(jammer.X, jammer.Y, u.X, u.Y) <= r {
				u.JammedUntil = g.Time + .3
			}
		}
		if g.Time >= jammer.PulseAt {
			jammer.PulseAt = g.Time + 2
			g.Effects = append(g.Effects, &Effect{Type: "jam", X: jammer.X, Y: jammer.Y, Size: &r, Owner: jammer.Owner, Age: 0, Duration: 1.2})
		}
	}
}

// hasSignalCover 委托 rules.HasSignalCover。
func (g *Game) hasSignalCover(owner int, x, y float64) bool {
	var relays []rules.RelayView
	for _, u := range g.ActiveUnits(owner, "relay") {
		relays = append(relays, rules.RelayView{X: u.X, Y: u.Y, StunUntil: u.StunUntil})
	}
	beacons := make([]rules.BeaconView, len(g.Beacons))
	for i, b := range g.Beacons {
		beacons[i] = rules.BeaconView{X: b.X, Y: b.Y, Owner: b.Owner}
	}
	return rules.HasSignalCover(g.Time, relays, beacons, owner, x, y)
}

// laserHeat 对齐 modern-combat.js:140-143。
func (g *Game) laserHeat(u *Unit) {
	u.Heat = math.Min(100, u.Heat+34)
	if u.Heat >= 95 {
		u.Overheated = true
	}
}

// launchProjectile 对齐 modern-combat.js:145-168。
func (g *Game) launchProjectile(source targetEntity, target targetEntity, amount float64, style string) bool {
	if style == "carrier" {
		flights := 0
		for _, p := range g.Projectiles {
			if !p.Finished && p.SourceID == source.getID() && p.Kind == "wing" {
				flights++
			}
		}
		if su, ok := source.(*Unit); ok && (flights >= int(*su.Wing) || *su.Wing <= 0) {
			return false
		}
	}
	kind := "missile"
	switch style {
	case "bomber":
		kind = "bomb"
	case "carrier":
		kind = "wing"
	case "rocket":
		kind = "rocket"
	case "submarine":
		kind = "torpedo"
	case "loiterer":
		kind = "loitering"
	}
	speed := 185.0
	switch kind {
	case "wing":
		speed = 185
	case "torpedo":
		speed = 155
	case "rocket":
		speed = 300
	case "missile":
		speed = 360
	}
	hp := 38.0
	if kind == "wing" {
		hp = 90
	}
	p := &Projectile{
		ID: g.NextProjectileID, Kind: kind, Owner: source.getOwner(), SourceID: source.getID(),
		TargetID: target.getID(), X: source.getX(), Y: source.getY(),
		StartX: source.getX(), StartY: source.getY(),
		ToX: target.getX(), ToY: target.getY(),
		Angle:     math.Atan2(target.getY()-source.getY(), target.getX()-source.getX()),
		Amount:    amount, HP: hp, Speed: speed,
	}
	g.NextProjectileID++
	if kind == "torpedo" {
		su := source.(*Unit)
		offset := 6.2 * 8
		p.X = clamp(su.X+math.Cos(su.Angle)*offset, 0, g.World.Width)
		p.Y = clamp(su.Y+math.Sin(su.Angle)*offset, 0, g.World.Height)
		p.StartX, p.StartY = p.X, p.Y
		p.Angle = su.Angle
	}
	g.Projectiles = append(g.Projectiles, p)
	if su, ok := source.(*Unit); ok {
		sourceHeight := 20.0
		if gpdata.HasTag(su.Type, "jet") && !g.aircraftGroundedFull(su) {
			sourceHeight = 95
		} else if su.Type == "destroyer" {
			sourceHeight = 35
		}
		p.SourceHeight = sourceHeight
	}
	if tu, ok := target.(*Unit); ok {
		th := 12.0
		if gpdata.HasTag(tu.Type, "jet") && !g.aircraftGroundedFull(tu) {
			th = 95
		} else if gpdata.HasTag(tu.Type, "drone") {
			th = 25
		}
		p.TargetHeight = th
	} else {
		p.TargetHeight = 12
	}
	airborne := func(e targetEntity) bool {
		if e.getKind() != "unit" {
			return false
		}
		return gpdata.HasTag(e.getType(), "air") && !g.aircraftGroundedFull(e.(*Unit))
	}
	switch {
	case airborne(source) && airborne(target):
		p.FlightRole = "air-to-air"
	case airborne(source):
		p.FlightRole = "air-to-ground"
	default:
		p.FlightRole = "surface"
	}
	if kind == "bomb" {
		p.ToX, p.ToY = p.X, p.Y
		p.FallDuration = 1.1
	}
	if su, ok := source.(*Unit); ok && style == "submarine" {
		su.ExposedUntil = g.Time + 4
	}
	if g.Events.Shot != nil {
		g.Events.Shot(style, source.getOwner())
	}
	return true
}

// updateProjectilesFull 对齐 modern-combat.js:170-240。
func (g *Game) updateProjectilesFull(dt float64) {
	list := append([]*Projectile{}, g.Projectiles...)
	// 拦截阶段。
	for _, defender := range g.Units {
		if defender.HP <= 0 || defender.EmbarkedIn != 0 || g.Time < defender.StunUntil || defender.FireTimer > 0 {
			continue
		}
		switch defender.Type {
		case "laser", "aa", "frigate", "destroyer", "fighter", "aegis", "navalFighter":
		default:
			continue
		}
		d := gpdata.UNITS[defender.Type]
		if defender.EWSuppressedUntil > g.Time || defender.Overheated || (d.Ammo > 0 && (defender.Ammo == nil || *defender.Ammo <= 0)) || (defender.Order != nil && defender.Order.Type == "rearm") {
			continue
		}
		var target *Projectile
		bestD := 1e18
		for _, p := range list {
			if p.HP <= 0 || p.Owner == defender.Owner {
				continue
			}
			if p.Kind == "rocket" || p.Kind == "torpedo" || p.Kind == "bomb" {
				continue
			}
			if defender.Type == "laser" && p.Kind == "wing" {
				continue
			}
			if (defender.Type == "fighter" || defender.Type == "aegis" || defender.Type == "navalFighter") && p.Kind != "wing" {
				continue
			}
			dist := Distance(defender.X, defender.Y, p.X, p.Y)
			if dist <= d.Range && dist < bestD && g.IsVisibleFor(defender.Owner, p.X, p.Y) {
				bestD, target = dist, p
			}
		}
		if target == nil {
			continue
		}
		dmg := d.Damage * 1.8
		if defender.Type == "laser" {
			dmg = d.Damage
		}
		target.HP -= dmg
		defender.FireTimer = d.Cooldown
		defender.TurretAngle = math.Atan2(target.Y-defender.Y, target.X-defender.X)
		if d.Ammo > 0 && defender.Ammo != nil {
			*defender.Ammo--
		}
		if defender.Type == "laser" {
			g.laserHeat(defender)
		}
		g.Effects = append(g.Effects, &Effect{Type: "shot", X: defender.X, Y: defender.Y, ToX: &target.X, ToY: &target.Y, SourceType: defender.Type, Style: map[bool]string{true: "laser", false: "aa"}[defender.Type == "laser"], Owner: defender.Owner, Age: 0, Duration: .2})
		if g.Events.Shot != nil {
			g.Events.Shot(defender.Type, defender.Owner)
		}
	}
	// 飞行阶段。
	for _, p := range list {
		p.Age += dt
		if p.HP <= 0 {
			g.finishProjectile(p, true)
			continue
		}
		source := g.GetUnit(p.SourceID)
		jammed := false
		if p.Kind == "loitering" || p.Kind == "missile" || p.Kind == "wing" {
			if rules.WeatherState(g.Map, g.Time).Phase == "storm" && !g.hasSignalCover(p.Owner, p.X, p.Y) {
				jammed = true
			}
			if !jammed {
				for _, u := range g.EnemyUnits(p.Owner, "jammer") {
					r := gpdata.UNITS["jammer"].Range
					if g.Players[u.Owner].Faction == "middleeast" {
						r *= 1.15
					}
					if g.Time >= u.StunUntil && Distance(u.X, u.Y, p.X, p.Y) <= r {
						jammed = true
						break
					}
				}
			}
		}
		if jammed {
			p.Jam += dt
		} else if p.Jam > 0 {
			p.Jam = math.Max(0, p.Jam-dt*.25)
		}
		if p.Jam >= 1.1 && p.Kind != "wing" {
			g.finishProjectile(p, true)
			continue
		}
		var target targetEntity
		if tu := g.GetUnit(p.TargetID); tu != nil {
			target = tu
		} else if tb := g.GetBuilding(p.TargetID); tb != nil {
			target = tb
		}
		if p.Returning {
			if source == nil || source.HP <= 0 {
				g.finishProjectile(p, true)
				continue
			}
			p.ToX, p.ToY = source.X, source.Y
		} else if p.Kind != "rocket" && p.Kind != "bomb" && !jammed && target != nil && target.getHP() > 0 && g.CanSeeEntity(p.Owner, target) {
			p.ToX, p.ToY = target.getX(), target.getY()
		}
		remaining := Distance(p.X, p.Y, p.ToX, p.ToY)
		step := p.Speed * dt
		if jammed {
			step *= .5
		}
		heading := math.Atan2(p.ToY-p.Y, p.ToX-p.X)
		switch p.Kind {
		case "wing":
			p.Angle = turnToward(p.Angle, heading, 2.2*dt)
		case "missile":
			p.Angle = turnToward(p.Angle, heading, 5*dt)
		case "loitering":
			p.Angle = turnToward(p.Angle, heading, 5*dt)
		case "torpedo":
			p.Angle = turnToward(p.Angle, heading, 2*dt)
		default:
			p.Angle = heading
		}
		if p.Kind == "bomb" && p.FallDuration > 0 && p.Age < p.FallDuration {
			continue
		}
		if remaining > step+8 {
			p.X += math.Cos(p.Angle) * step
			p.Y += math.Sin(p.Angle) * step
		} else if p.Returning {
			g.finishProjectile(p, false)
		} else {
			g.projectileHit(p, target)
		}
		if p.Age > 22 {
			g.finishProjectile(p, p.Kind == "wing")
		}
	}
	live := g.Projectiles[:0]
	for _, p := range g.Projectiles {
		if !p.Finished {
			live = append(live, p)
		}
	}
	g.Projectiles = live
}

// projectileHit 对齐 updateProjectiles 命中分支。
func (g *Game) projectileHit(p *Projectile, target targetEntity) {
	if p.Kind == "rocket" || p.Kind == "bomb" {
		splash := 65.0
		if p.Kind == "bomb" {
			splash = gpdata.UNITS["bomber"].Splash
		}
		for _, e := range g.allTargets() {
			if e.getOwner() == p.Owner || e.getHP() <= 0 {
				continue
			}
			if eu, ok := e.(*Unit); ok && eu.EmbarkedIn != 0 {
				continue
			}
			if eu, ok := e.(*Unit); ok {
				if gpdata.HasTag(eu.Type, "submerged") {
					continue
				}
				if gpdata.HasTag(eu.Type, "air") && !g.aircraftGroundedFull(eu) {
					continue
				}
			}
			gap := Distance(e.getX(), e.getY(), p.ToX, p.ToY)
			if gap <= splash {
				g.pushHit(pendingHit{target: e, amount: p.Amount * (1 - gap/100), side: p.Owner, stun: 0})
			}
		}
	} else if target != nil && target.getHP() > 0 {
		if tu, ok := target.(*Unit); ok && tu.EmbarkedIn != 0 {
			return
		}
		subOK := true
		if tu, ok := target.(*Unit); ok && gpdata.HasTag(tu.Type, "submerged") {
			subOK = g.CanSeeEntity(p.Owner, target)
		}
		if subOK && Distance(target.getX(), target.getY(), p.ToX, p.ToY) < 45 {
			g.pushHit(pendingHit{target: target, amount: p.Amount, side: p.Owner, stun: 0})
		}
	}
	size := 24.0
	if p.Kind == "rocket" {
		size = 42
	}
	g.Effects = append(g.Effects, &Effect{Type: "explosion", X: p.ToX, Y: p.ToY, TargetType: p.TargetType, Size: &size, Owner: p.Owner, Age: 0, Duration: .7})
	if p.Kind == "wing" {
		if source := g.GetUnit(p.SourceID); source != nil && source.HP > 0 {
			p.Returning = true
			return
		}
	}
	g.finishProjectile(p, false)
}

// finishProjectile 对齐 modern-combat.js:241-252。
func (g *Game) finishProjectile(p *Projectile, intercepted bool) {
	if p.Finished {
		return
	}
	p.Finished = true
	if intercepted {
		g.Effects = append(g.Effects, &Effect{Type: "hit", X: p.X, Y: p.Y, TargetType: map[bool]string{true: "fighter", false: "drone"}[p.Kind == "wing"], Owner: p.Owner, Age: 0, Duration: .4})
		if p.Kind == "wing" {
			if source := g.GetUnit(p.SourceID); source != nil {
				*source.Wing = math.Max(0, *source.Wing-1)
			}
		}
	}
}
