package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/report"
	"github.com/cnyup/gopher-powers/internal/game/rules"
)

type reportEvent = report.Event

// combat.go 对齐 game.js L767-1000：canAttack/unitRange/unitDamage/fire/applyHit/damage/closestEnemy。

// targetEntity 战斗统一目标视图（Unit/Building 都满足）。
type targetEntity interface {
	getID() int
	getOwner() int
	getKind() string
	getType() string
	getX() float64
	getY() float64
	getHP() float64
}

func (u *Unit) getID() int      { return u.ID }
func (u *Unit) getOwner() int   { return u.Owner }
func (u *Unit) getKind() string { return u.Kind }
func (u *Unit) getType() string { return u.Type }
func (u *Unit) getX() float64   { return u.X }
func (u *Unit) getY() float64   { return u.Y }
func (u *Unit) getHP() float64  { return u.HP }

func (b *Building) getID() int      { return b.ID }
func (b *Building) getOwner() int   { return b.Owner }
func (b *Building) getKind() string { return b.Kind }
func (b *Building) getType() string { return b.Type }
func (b *Building) getX() float64   { return b.X }
func (b *Building) getY() float64   { return b.Y }
func (b *Building) getHP() float64  { return b.HP }

// aircraftGrounded 委托 full 版。
func (g *Game) aircraftGrounded(e targetEntity) bool {
	if e.getKind() != "unit" {
		return false
	}
	return g.aircraftGroundedFull(e.(*Unit))
}

// CanAttack 对齐 game.js L767-795。
func (g *Game) CanAttack(source targetEntity, target targetEntity) bool {
	if tu, ok := target.(*Unit); ok && tu.EmbarkedIn != 0 {
		return false
	}
	if target.getKind() == "unit" && gpdata.HasTag(target.getType(), "submerged") {
		_, suIsUnit := source.(*Unit)
		if !suIsUnit || (!gpdata.HasTag(source.getType(), "anti-sub") && source.getType() != "submarine") {
			return false
		}
	}
	if source.getKind() == "building" {
		return target.getKind() == "building" || g.aircraftGrounded(target) || !gpdata.HasTag(target.getType(), "jet")
	}
	typ := source.getType()
	if gpdata.UNITS[typ].Damage == 0 {
		return false
	}
	// bridgeBlocksDirectShot（对齐 bridge-fire）。
	if g.bridgeBlocksShot(source, target) {
		return false
	}
	if typ == "carrier" && len(g.carrierAircraft(source.(*Unit))) > 0 {
		return false
	}
	// 目标 tag 集（停泊飞机剥离 air 标签）。
	var tags []string
	if target.getKind() == "unit" {
		for _, t := range gpdata.UNITS[target.getType()].Tags {
			if g.aircraftGrounded(target) && (t == "air" || t == "jet" || t == "drone") {
				continue
			}
			tags = append(tags, t)
		}
	}
	has := func(t string) bool {
		for _, x := range tags {
			if x == t {
				return true
			}
		}
		return false
	}
	switch typ {
	case "aa":
		return has("air")
	case "laser":
		return has("drone")
	case "submarine":
		return has("ship")
	case "rocket", "loiterer", "carrier", "bomber":
		if has("air") {
			return false
		}
	case "aegis":
		return has("air")
	case "fighter", "navalFighter":
		return true
	case "railgun":
		if has("air") {
			return false
		}
	case "strike", "navalStrike":
		return true
	}
	if gpdata.HasTag(typ, "ship") {
		if has("jet") && !gpdata.HasTag(typ, "anti-air") {
			return false
		}
		water := g.Map.Water
		if water == nil {
			return false
		}
		cx := clamp(target.getX(), water.X1+20, water.X2-20)
		cy := clamp(target.getY(), water.Y1+20, water.Y2-20)
		reach := gpdata.UNITS[typ].Range
		if target.getKind() == "building" {
			b := target.(*Building)
			reach += b.Size * .36
		}
		return Distance(cx, cy, target.getX(), target.getY()) <= reach
	}
	if !has("jet") {
		return true
	}
	return typ == "elite" && g.Players[source.getOwner()].Faction == "china"
}

// bridgeBlocksShot 委托 data.BridgeBlocksDirectShot。
func (g *Game) bridgeBlocksShot(source, target targetEntity) bool {
	if source.getKind() != "unit" || target.getKind() != "unit" {
		return false
	}
	return gpdata.BridgeBlocksDirectShot(g.Map, gpdata.ShotEntity{Kind: source.getKind(), Type: source.getType(), X: source.getX(), Y: source.getY()}, gpdata.ShotEntity{Kind: target.getKind(), Type: target.getType(), X: target.getX(), Y: target.getY()})
}

// carrierAircraft 对齐 modern-combat carrierAircraft（1d 完整化，先按翼载机判定）。
func (g *Game) carrierAircraft(c *Unit) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.HP > 0 && u.HomeCarrierID == c.ID {
			out = append(out, u)
		}
	}
	return out
}

// UnitRange 对齐 game.js L834-839。
func (g *Game) UnitRange(u *Unit, target targetEntity) float64 {
	r := gpdata.UNITS[u.Type].Range
	if u.Type == "elite" {
		switch g.Players[u.Owner].Faction {
		case "middleeast":
			r *= 1.15
		case "russia":
			if target.getKind() == "building" {
				r *= 1.28
			}
		}
	}
	return r
}

// UnitDamage 对齐 game.js L841-878（克制表全量）。
func (g *Game) UnitDamage(u *Unit, target targetEntity) float64 {
	amount := gpdata.UNITS[u.Type].Damage
	var targetTags []string
	if target.getKind() == "unit" {
		for _, t := range gpdata.UNITS[target.getType()].Tags {
			if g.aircraftGrounded(target) && (t == "air" || t == "jet" || t == "drone") {
				continue
			}
			targetTags = append(targetTags, t)
		}
	}
	has := func(t string) bool {
		for _, x := range targetTags {
			if x == t {
				return true
			}
		}
		return false
	}
	isBuilding := target.getKind() == "building"
	switch u.Type {
	case "drone":
		if has("armor") {
			amount *= 1.8
		}
		if has("jet") {
			amount *= 0.7
		}
	case "aa":
		if has("air") {
			amount *= 2.25
		}
	case "fighter", "aegis", "navalFighter":
		if has("air") {
			if has("jet") {
				amount *= 1.15
			} else {
				amount *= 1.4
			}
		}
	case "strike", "navalStrike":
		if has("air") {
			amount *= .35
		}
		if isBuilding {
			amount *= 1.7
		}
		if has("armor") {
			amount *= 1.2
		}
		if has("ship") {
			amount *= 1.25
		}
		if has("infantry") {
			amount *= 0.65
		}
	case "patrol":
		if isBuilding {
			amount *= .45
		}
		if has("ship") {
			amount *= 1.18
		}
	case "frigate":
		if isBuilding {
			amount *= .65
		}
		if has("jet") {
			amount *= 1.55
		}
	case "tank":
		if has("infantry") || has("anti-air") {
			amount *= 1.45
		}
	case "rifle":
		if has("armor") {
			amount *= 0.52
		}
	case "elite":
		switch g.Players[u.Owner].Faction {
		case "china":
			if has("air") {
				amount *= 2.2
			}
		case "russia":
			if isBuilding {
				amount *= 2.2
			}
		case "nato":
			if has("infantry") {
				amount *= 1.6
			}
			if has("drone") {
				amount *= 1.45
			}
		case "asia":
			if has("armor") {
				amount *= 1.7
			}
		case "middleeast":
			if has("drone") || has("vehicle") {
				amount *= 1.2
			}
		}
	}
	if g.Players[u.Owner].Faction == "russia" && isBuilding {
		amount *= 1.15
	}
	if g.Players[u.Owner].Faction == "russia" && u.Type == "aa" && has("air") {
		amount *= 0.88
	}
	if g.Players[u.Owner].Faction == "middleeast" && u.Type == "tank" {
		amount *= 0.9
	}
	// tacticalDamage（掩体/朝向因子）。
	return amount * g.tacticalFactor(u, target)
}

// tacticalFactor 对齐 tactical-rules tacticalDamage 调用。
func (g *Game) tacticalFactor(u *Unit, target targetEntity) float64 {
	ct := rules.CombatTarget{Kind: target.getKind(), Type: target.getType(), X: target.getX(), Y: target.getY()}
	angle := 0.0
	if tu, ok := target.(*Unit); ok {
		angle = tu.Angle
	}
	return rules.TacticalDamage(g.Map, u.Type, u.X, u.Y, ct, angle)
}

// Fire 对齐 game.js L881-915（特效 + 命中收集 + 溅射）。
func (g *Game) Fire(source targetEntity, target targetEntity, amount float64, style string) {
	// 弹道型在 1d launchProjectile 完整化；此处直射路径。
	g.Effects = append(g.Effects, &Effect{
		Type: "shot", SourceID: source.getID(), X: source.getX(), Y: source.getY(),
		ToX: &[]float64{target.getX()}[0], ToY: &[]float64{target.getY()}[0],
		SourceType: source.getType(), TargetType: target.getType(),
		Age: 0, Duration: .18, Owner: source.getOwner(), Style: style,
	})
	hitDur := .42
	g.Effects = append(g.Effects, &Effect{
		Type: "hit", X: target.getX(), Y: target.getY(), TargetType: target.getType(),
		Age: 0, Duration: hitDur, Owner: source.getOwner(),
	})
	stun := 0.0
	if source.getType() == "elite" && g.Players[source.getOwner()].Faction == "middleeast" {
		if gpdata.HasTag(target.getType(), "drone") || gpdata.HasTag(target.getType(), "vehicle") {
			stun = 1.5
		}
	}
	g.pushHit(pendingHit{target: target, amount: amount, side: source.getOwner(), stun: stun})
	if g.Events.Shot != nil {
		g.Events.Shot(style, source.getOwner())
	}
}

func (g *Game) pushHit(h pendingHit) {
	if g.CollectingHits {
		g.PendingHits = append(g.PendingHits, h)
	} else {
		g.applyHit(h)
	}
}

// applyHit 对齐 game.js L917-920。
func (g *Game) applyHit(h pendingHit) {
	if h.target.getHP() <= 0 {
		return
	}
	g.Damage(h.target, h.amount, h.side)
	if h.stun > 0 && h.target.getHP() > 0 {
		if tu, ok := h.target.(*Unit); ok {
			if tu.StunUntil < g.Time+h.stun {
				tu.StunUntil = g.Time + h.stun
			}
		}
	}
}

// Damage 对齐 game.js L922-950。
func (g *Game) Damage(target targetEntity, amount float64, attackerSide int) {
	if target.getHP() <= 0 || math.IsInf(amount, 0) || math.IsNaN(amount) || amount <= 0 {
		return
	}
	if g.Players[target.getOwner()].ShieldUntil > g.Time {
		amount *= 0.48
	}
	if target.getKind() == "building" && g.Players[target.getOwner()].Faction == "china" && target.getType() == "turret" {
		amount *= 0.85
	}
	// recordBattleDamage 对齐。
	actual := amount
	if target.getHP() < actual {
		actual = target.getHP()
	}
	gmetric(g, target.getOwner(), "damageTaken", actual)
	if attackerSide >= 0 && attackerSide != target.getOwner() {
		gmetric(g, attackerSide, "damageDealt", actual)
	}
	// 扣血。
	switch v := target.(type) {
	case *Unit:
		v.HP -= amount
		v.LastDamageAt = g.Time
	case *Building:
		v.HP -= amount
		v.LastDamageAt = g.Time
	}
	// 击杀判定在 recordBattleDamage 内（amount >= hp）；此处补事件与特效。
	if target.getHP() <= 0 {
		// recordBattleDamage 尾段。
		building := target.getKind() == "building"
		if building {
			gmetric(g, target.getOwner(), "buildingsLost", 1)
		} else {
			gmetric(g, target.getOwner(), "losses", 1)
		}
		if attackerSide >= 0 && attackerSide != target.getOwner() {
			if building {
				gmetric(g, attackerSide, "buildingsDestroyed", 1)
			} else {
				gmetric(g, attackerSide, "kills", 1)
			}
		}
		if g.BattleStats != nil {
			g.BattleStats.Events = append(g.BattleStats.Events, battleEvent{Time: g.Time, Owner: target.getOwner(), Type: target.getType(), Kind: lostKind(building)})
			if len(g.BattleStats.Events) > 32 {
				g.BattleStats.Events = g.BattleStats.Events[len(g.BattleStats.Events)-32:]
			}
		}
		switch v := target.(type) {
		case *Unit:
			v.HP = 0
		case *Building:
			v.HP = 0
		}
		size := 40.0
		if b, ok := target.(*Building); ok {
			size = b.Size
		}
		g.Effects = append(g.Effects, &Effect{Type: "explosion", X: target.getX(), Y: target.getY(), TargetType: target.getType(), Age: 0, Duration: 1.2, Owner: attackerSide, Size: &size})
		g.RecalculatePower()
	}
}

func lostKind(building bool) string {
	if building {
		return "buildingLost"
	}
	return "unitLost"
}

// ClosestEnemy 对齐 game.js L952-969。
func (g *Game) ClosestEnemy(source targetEntity, rangeLimit float64, owner int) targetEntity {
	var nearest targetEntity
	best := rangeLimit
	for _, target := range g.allTargets() {
		if target.getOwner() == owner || target.getHP() <= 0 || !g.CanAttack(source, target) {
			continue
		}
		if source.getKind() == "building" && !g.hasLineOfFire(source, target) {
			continue
		}
		d := Distance(source.getX(), source.getY(), target.getX(), target.getY())
		if target.getKind() == "building" {
			d -= target.(*Building).Size * 0.3
		}
		if d >= rangeLimit {
			continue
		}
		airborne := target.getKind() == "unit" && gpdata.HasTag(target.getType(), "air") && !g.aircraftGrounded(target)
		primaryAir := source.getType() == "fighter" || source.getType() == "navalFighter"
		primaryGround := source.getType() == "strike" || source.getType() == "navalStrike"
		priority := 1.0
		if primaryAir {
			if airborne {
				priority = .3
			}
		} else if primaryGround {
			if airborne {
				priority = 1
			} else {
				priority = .3
			}
		} else if (source.getType() == "aa" || source.getType() == "frigate") && target.getKind() == "unit" && gpdata.HasTag(target.getType(), "jet") {
			priority = .65
		}
		if d*priority < best && g.CanSeeEntity(owner, target) {
			best = d * priority
			nearest = target
		}
	}
	return nearest
}

func (g *Game) allTargets() []targetEntity {
	out := make([]targetEntity, 0, len(g.Units)+len(g.Buildings))
	for _, u := range g.Units {
		out = append(out, u)
	}
	for _, b := range g.Buildings {
		out = append(out, b)
	}
	return out
}

// hasLineOfFire 对齐 game.js L825-832。
func (g *Game) hasLineOfFire(source, target targetEntity) bool {
	switch source.getType() {
	case "rocket", "loiterer":
		return true
	}
	if source.getKind() == "unit" && gpdata.HasTag(source.getType(), "air") {
		return true
	}
	if target.getKind() == "unit" && gpdata.HasTag(target.getType(), "air") {
		return true
	}
	if source.getKind() == "unit" && gpdata.HasTag(source.getType(), "ship") {
		return true
	}
	if target.getKind() == "unit" && gpdata.HasTag(target.getType(), "ship") {
		return true
	}
	for _, r := range g.Map.Barriers {
		if crossesRect(source.getX(), source.getY(), target.getX(), target.getY(), r) {
			return false
		}
	}
	return true
}

// CanSeeEntity 对齐 game.js L1202-1211。
func (g *Game) CanSeeEntity(side int, target targetEntity) bool {
	if tu, ok := target.(*Unit); ok && tu.EmbarkedIn != 0 {
		return false
	}
	if target.getOwner() == side {
		return true
	}
	if tb, ok := target.(*Building); ok && tb.HP > 0 && tb.Sabotage != nil && tb.Sabotage.Owner == side {
		return true
	}
	if !g.IsVisibleFor(side, target.getX(), target.getY()) {
		return false
	}
	if tu, ok := target.(*Unit); ok && gpdata.HasTag(tu.Type, "submerged") {
		if g.submarineSurfaced(tu) || tu.ExposedUntil > g.Time {
			return true
		}
		for _, u := range g.ActiveUnits(side, "") {
			if sonar := gpdata.UNITS[u.Type].Sonar; sonar > 0 && Distance(u.X, u.Y, tu.X, tu.Y) <= sonar {
				return true
			}
		}
		return false
	}
	if target.getKind() != "unit" || !gpdata.HasTag(target.getType(), "stealth") {
		return true
	}
	for _, u := range g.ActiveUnits(side, "") {
		if Distance(u.X, u.Y, target.getX(), target.getY()) < g.detectionRange(side, u) {
			return true
		}
	}
	for _, s := range g.Beacons {
		if s.Owner == side && Distance(s.X, s.Y, target.getX(), target.getY()) < 330 {
			return true
		}
	}
	for _, b := range g.OwnedBuildings(side, "radar") {
		if Distance(b.X, b.Y, target.getX(), target.getY()) < g.detectionRange(side, b) {
			return true
		}
	}
	return false
}

// submarineSurfaced 委托 full 版。
func (g *Game) submarineSurfaced(u *Unit) bool { return g.submarineSurfacedFull(u) }

// detectionRange 对齐 game.js L1193-1199。
func (g *Game) detectionRange(side int, e targetEntity) float64 {
	if e.getKind() == "unit" && g.IsRobot(e.getType()) {
		if u := e.(*Unit); u.Battery != nil && *u.Battery <= 0 {
			return 0
		}
	}
	if e.getKind() == "building" {
		if e.getType() == "radar" && g.HasPower(side) {
			if g.Players[side].Faction == "china" {
				return 610
			}
			return 470
		}
		return 0
	}
	switch e.getType() {
	case "scout":
		return 260
	case "aa":
		if g.Players[side].Faction == "china" {
			return 230
		}
		return 175
	}
	return 82
}

// battleEvent 别名 report.Event（Game.BattleStats.Events 元素类型）。
type battleEvent = reportEvent
