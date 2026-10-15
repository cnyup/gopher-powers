package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// intelligence.go 对齐 src/intelligence.js（INTELLIGENCE_RULES / intelligenceCombat mixin）。

// IntelligenceRules 对齐 INTELLIGENCE_RULES。
var IntelligenceRules = struct {
	BombDelay    float64
	DefuseTime   float64
	CyberCharge  float64
	CyberWarning float64
	CyberDuration float64
	CyberCost    float64
}{45, 3, 120, 12, 4, 500}

// LaunchCyber 对齐 intelligence.js launchCyber。
func (g *Game) LaunchCyber(side int) bool {
	targetSide := g.nearestEnemySide(side)
	if targetSide < 0 {
		return false
	}
	p := g.Players[side]
	var source *Building
	for _, b := range g.OwnedBuildings(side, "super") {
		source = b
		break
	}
	if g.IsControlLocked(side) || source == nil || !g.HasPower(side) ||
		p.CyberCharge < IntelligenceRules.CyberCharge || p.Credits < IntelligenceRules.CyberCost || p.CyberPending != nil {
		return false
	}
	p.Credits -= IntelligenceRules.CyberCost
	p.CyberCharge = 0
	p.CyberPending = &CyberPending{SourceID: source.ID, TargetSide: targetSide, ExecuteAt: g.Time + IntelligenceRules.CyberWarning}
	return true
}

// UpdateIntelligence 对齐 intelligence.js updateIntelligence(dt)：炸弹引爆 + cyber 状态机。
func (g *Game) UpdateIntelligence(dt float64) {
	for _, b := range g.Buildings {
		if b.HP > 0 && b.Sabotage != nil && b.Sabotage.DetonateAt <= g.Time {
			owner := b.Sabotage.Owner
			b.Sabotage = nil
			g.Damage(b, b.MaxHP, owner)
		}
	}
	for _, p := range g.Players {
		hasSuper := g.HasBuilding(p.Side, "super")
		hasPow := g.HasPower(p.Side)
		active := hasSuper && hasPow
		if active && p.CyberPending == nil {
			p.CyberCharge = math.Min(IntelligenceRules.CyberCharge, p.CyberCharge+dt)
		} else if !active {
			p.CyberCharge = 0
		}
		if p.CyberPending != nil {
			src := g.getEntityAny(p.CyberPending.SourceID)
			bv, isB := src.(buildingView2)
			var b *Building
			if isB {
				b = bv.b
			}
			if !active || !isB || b.HP <= 0 || b.Type != "super" || b.Owner != p.Side {
				p.CyberPending = nil
			} else if p.CyberPending.ExecuteAt <= g.Time {
				// targetSide 已在 LaunchCyber 固定；六方最近敌在此重取。
				_ = 0
				if p.CyberPending.TargetSide >= 0 && p.CyberPending.TargetSide < len(g.Players) {
					target := g.Players[p.CyberPending.TargetSide]
					if !target.Eliminated {
						target.CyberSourceID = b.ID
						target.CyberLockedUntil = g.Time + IntelligenceRules.CyberDuration
					}
				}
				p.CyberPending = nil
			}
		}
		if p.CyberSourceID != 0 && !g.IsControlLocked(p.Side) {
			p.CyberSourceID = 0
			p.CyberLockedUntil = 0
		}
	}
	// 指令执行（infiltrate/defuse/collect 的单位侧行为）。
	for _, u := range g.Units {
		if u.HP > 0 {
			g.intelligenceOrder(u, dt)
		}
	}
}

// intelligenceOrder 对齐 intelligence.js orderHandling 段（44-77 行）。
func (g *Game) intelligenceOrder(u *Unit, dt float64) {
	o := u.Order
	if o == nil || (o.Type != "infiltrate" && o.Type != "defuse" && o.Type != "collect") {
		return
	}
	var target interface {
		getX() float64
		getY() float64
	}
	_ = target
	var bTgt *Building
	if o.TargetID != 0 {
		if e := g.getEntityAny(o.TargetID); e != nil {
			if bv, ok := e.(buildingView2); ok {
				bTgt = bv.b
			}
		}
	}
	var site *ResourceSiteT
	if bTgt == nil && o.SiteID != "" {
		for i := range g.ResourceSites {
			if g.ResourceSites[i].ID == o.SiteID {
				site = &g.ResourceSites[i]
				break
			}
		}
	}
	if bTgt == nil && site == nil {
		u.Order = nil
		u.Path = nil
		return
	}
	tx, ty := o.X, o.Y
	reach := 34.0
	if bTgt != nil {
		tx, ty = bTgt.X, bTgt.Y
		reach = bTgt.Size*.48 + 16
	} else {
		tx, ty = site.X, site.Y
	}
	if Distance(u.X, u.Y, tx, ty) > reach {
		o.Progress = 0
		g.moveUnit(u, Point{tx, ty}, dt, reach-4)
		return
	}
	if g.Time-u.LastDamageAt < 1 {
		return
	}
	switch o.Type {
	case "infiltrate":
		bTgt.Sabotage = &Sabotage{Owner: u.Owner, PlantedAt: g.Time, DetonateAt: g.Time + IntelligenceRules.BombDelay}
		u.HP = 0
		u.Order = nil
	case "defuse":
		o.Progress += dt
		if o.Progress >= IntelligenceRules.DefuseTime {
			bTgt.Sabotage = nil
			u.Order = nil
			u.Path = nil
		}
	case "collect":
		// 敌方回收者同时抵达时停止交付。
		for _, v := range g.EnemyUnits(u.Owner, "") {
			if v.Order != nil && v.Order.Type == "collect" && v.Order.TargetID == o.TargetID && v.Order.SiteID == o.SiteID &&
				Distance(v.X, v.Y, tx, ty) <= reach {
				return
			}
		}
		rate := 45.0
		if site.Type == "cache" {
			rate = 75
		}
		gain := math.Min(site.Amount, dt*rate)
		site.Amount -= gain
		g.Players[u.Owner].Credits += gain
		o.Progress += dt
		u.CollectionFX += gain
		if u.CollectionFX >= 75 || site.Amount <= 0 {
			amt := math.Round(u.CollectionFX)
			g.Effects = append(g.Effects, &Effect{Type: "income", X: u.X, Y: u.Y, Amount: &amt, Owner: u.Owner, Age: 0, Duration: 1.5})
			u.CollectionFX = 0
		}
		if site.Amount <= 0 {
			site.Amount = 0
			u.Order = nil
			u.Path = nil
		}
	}
}

// nearestEnemySide 对齐 game.js nearestEnemySide（六方取基地距离最近敌席；双人=1-side）。
func (g *Game) nearestEnemySide(side int) int {
	if g.PlayerCount == 2 {
		if side == 0 {
			return 1
		}
		return 0
	}
	best, bestD := -1, 1e18
	home := g.Map.Spawns[side]
	for _, p := range g.Players {
		if p.Side == side || p.Eliminated {
			continue
		}
		sp := g.Map.Spawns[p.Side]
		if d := Distance(home.X, home.Y, sp.X, sp.Y); d < bestD {
			bestD, best = d, p.Side
		}
	}
	return best
}

// updateIntelligenceAI 对齐 intelligence.js updateIntelligenceAI(side)。
func (g *Game) updateIntelligenceAI(side int) {
	if g.IsControlLocked(side) {
		return
	}
	// 拆弹调度。
	var bombs []*Building
	for _, b := range g.OwnedBuildings(side, "") {
		if b.Sabotage != nil {
			bombs = append(bombs, b)
		}
	}
	if len(bombs) > 0 {
		for _, e := range g.ActiveUnits(side, "engineer") {
			if e.Order != nil && e.Order.Type == "defuse" {
				continue
			}
			best := bombs[0]
			bestD := Distance(best.X, best.Y, e.X, e.Y)
			for _, b := range bombs[1:] {
				if d := Distance(b.X, b.Y, e.X, e.Y); d < bestD {
					best, bestD = b, d
				}
			}
			e.Order = &Order{Type: "defuse", TargetID: best.ID, X: best.X, Y: best.Y, Progress: 0}
			e.Path = nil
		}
		hasEngineer := len(g.OwnedUnits(side, "engineer")) > 0
		producing := false
		for _, b := range g.OwnedBuildings(side, "barracks") {
			if (b.Active != nil && b.Active.Type == "engineer") || contains(b.Queue, "engineer") {
				producing = true
			}
		}
		if !hasEngineer && !producing {
			g.queueAIUnit("engineer", side)
		}
	}
	// 间谍潜入调度。
	for _, spy := range g.ActiveUnits(side, "scout") {
		if spy.Order != nil && (spy.Order.Type == "infiltrate" || spy.Order.Type == "rearm") {
			continue
		}
		var target *Building
		bestD := 1e18
		for _, b := range g.EnemyBuildings(side, "") {
			if b.Sabotage != nil || !g.CanSeeEntity(side, b) {
				continue
			}
			if d := Distance(b.X, b.Y, spy.X, spy.Y); d < gpdata.UNITS["scout"].Sight && d < bestD {
				target, bestD = b, d
			}
		}
		if target != nil {
			spy.Order = &Order{Type: "infiltrate", TargetID: target.ID, X: target.X, Y: target.Y}
			spy.Path = nil
		}
	}
	// 空闲地面单位采集调度。
	for _, u := range g.ActiveUnits(side, "") {
		if u.Order != nil || gpdata.HasTag(u.Type, "air") || gpdata.HasTag(u.Type, "ship") || gpdata.HasTag(u.Type, "logistics") {
			continue
		}
		var site *ResourceSiteT
		bestD := 380.0
		for i := range g.ResourceSites {
			s := &g.ResourceSites[i]
			if s.Type == "depot" || s.Amount <= 0 || !g.HasExploredFor(side, s.X, s.Y) {
				continue
			}
			if s.Type != "cache" && u.Type != "engineer" && u.Type != "harvester" && u.Type != "supply" {
				continue
			}
			if d := Distance(s.X, s.Y, u.X, u.Y); d < bestD {
				site, bestD = s, d
			}
		}
		if site != nil {
			u.Order = &Order{Type: "collect", SiteID: site.ID, X: site.X, Y: site.Y, Progress: 0}
		}
	}
	if g.Players[side].CyberCharge >= IntelligenceRules.CyberCharge {
		g.LaunchCyber(side)
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
