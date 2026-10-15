package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/spacing"
)

// dispatch.go 对齐 game.js L385-437：command/stopSelected/toggleAutoSupply/selectAt/selectBox/selectAllCombat。

// Command 对齐 game.js L385 command(x, y, attackMove, side)。
func (g *Game) Command(x, y float64, attackMove bool, side int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	patrol := g.OrderMode == "patrol"
	var selectedUnits []*Unit
	for _, id := range g.Selected {
		if e := g.GetUnit(id); e != nil && e.Owner == side && e.EmbarkedIn == 0 && !freightTypes[e.Type] {
			selectedUnits = append(selectedUnits, e)
		}
	}
	if len(selectedUnits) == 0 {
		for _, id := range g.Selected {
			g.SetRallyPoint(side, id, x, y)
		}
		return true
	}
	// 敌目标。
	var enemy targetEntity
	for i := len(g.Units) - 1; i >= 0; i-- {
		e := g.Units[i]
		if e.Owner != side && e.HP > 0 && g.CanSeeEntity(side, e) && Distance(e.X, e.Y, x, y) < 25 {
			enemy = e
			break
		}
	}
	if enemy == nil {
		for i := len(g.Buildings) - 1; i >= 0; i-- {
			b := g.Buildings[i]
			if b.Owner != side && b.HP > 0 && g.CanSeeEntity(side, b) && Distance(b.X, b.Y, x, y) < b.Size*0.55 {
				enemy = b
				break
			}
		}
	}
	// 站点（油井/信标/能源仓）——仅 engineer capture 用坐标。
	var sitePt *Point
	siteHit := func(sx, sy float64, owner int) bool {
		return Distance(sx, sy, x, y) < 40 && owner != side && g.HasExploredFor(side, sx, sy)
	}
	for _, s := range g.Oil {
		if siteHit(s.X, s.Y, s.Owner) {
			sitePt = &Point{s.X, s.Y}
			break
		}
	}
	if sitePt == nil {
		for _, s := range g.Beacons {
			if siteHit(s.X, s.Y, s.Owner) {
				sitePt = &Point{s.X, s.Y}
				break
			}
		}
	}
	if sitePt == nil {
		for i := range g.ResourceSites {
			s := &g.ResourceSites[i]
			if s.Type == "depot" && siteHit(s.X, s.Y, s.Owner) {
				sitePt = &Point{s.X, s.Y}
				break
			}
		}
	}
	// 物资箱。
	var cache *gpdata.ResourceSite
	for i := range g.ResourceSites {
		s := &g.ResourceSites[i]
		if s.Type != "depot" && s.Amount > 0 && Distance(s.X, s.Y, x, y) < 45 && g.HasExploredFor(side, s.X, s.Y) {
			cache = s
			break
		}
	}
	// 被潜入建筑（拆弹）。
	var bomb *Building
	for _, b := range g.OwnedBuildings(side, "") {
		if b.Sabotage != nil && Distance(b.X, b.Y, x, y) < b.Size*.55 {
			bomb = b
			break
		}
	}
	// 运输载具。
	var transport *Unit
	if !attackMove {
		for _, v := range g.ActiveUnits(side, "") {
			if gpdata.UNITS[v.Type].Capacity == 0 {
				continue
			}
			hit := false
			for _, id := range g.Selected {
				if id == v.ID {
					hit = true
					break
				}
			}
			if hit {
				continue
			}
			r := 45.0
			if v.Type == "carrier" {
				r = 95
			} else if v.Type == "landing" {
				r = 75
			}
			if Distance(v.X, v.Y, x, y) < r {
				transport = v
				break
			}
		}
	}
	var port *Building
	if !attackMove && !patrol {
		for _, b := range g.OwnedBuildings(side, "dock") {
			if Distance(b.X, b.Y, x, y) <= b.Size*.65 {
				port = b
				break
			}
		}
	}
	// 阵型偏移。
	ents := make([]spacing.Entity, len(selectedUnits))
	for i, u := range selectedUnits {
		ents[i] = unitView{u}
	}
	offsets := spacing.FormationOffsets(ents, spacing.Point{X: x, Y: y})
	for _, u := range selectedUnits {
		off := offsets[u.ID]
		if patrol {
			goal := g.ResolveMoveGoal(u, clamp(x+off.X, 20, g.World.Width-20), clamp(y+off.Y, 20, g.World.Height-20))
			u.Order = &Order{Type: "patrol", X: goal.X, Y: goal.Y, OriginX: u.X, OriginY: u.Y, DestinationX: goal.X, DestinationY: goal.Y}
		} else if port != nil && gpdata.HasTag(u.Type, "ship") {
			g.RequestResupplyFull(u, port)
		} else if transport != nil && g.CanBoardTransport(u, transport) {
			u.Order = &Order{Type: "board", TargetID: transport.ID}
		} else if !attackMove && bomb != nil && u.Type == "engineer" {
			u.Order = &Order{Type: "defuse", TargetID: bomb.ID, X: bomb.X, Y: bomb.Y, Progress: 0}
		} else if !attackMove && enemy != nil && enemy.getKind() == "building" && u.Type == "scout" && enemy.(*Building).Sabotage == nil {
			u.Order = &Order{Type: "infiltrate", TargetID: enemy.getID(), X: enemy.getX(), Y: enemy.getY()}
		} else if !attackMove && cache != nil && !gpdata.HasTag(u.Type, "air") && !gpdata.HasTag(u.Type, "ship") &&
			(cache.Type == "cache" || u.Type == "engineer" || u.Type == "harvester" || u.Type == "supply") {
			u.Order = &Order{Type: "collect", TargetID: cacheID(cache.ID), X: cache.X, Y: cache.Y, Progress: 0}
		} else if sitePt != nil && u.Type == "engineer" {
			u.Order = &Order{Type: "capture", X: sitePt.X, Y: sitePt.Y}
		} else if enemy != nil && g.CanAttack(u, enemy) {
			u.Order = &Order{Type: "attack", TargetID: enemy.getID(), X: enemy.getX(), Y: enemy.getY()}
		} else {
			goal := g.ResolveMoveGoal(u, clamp(x+off.X, 20, g.World.Width-20), clamp(y+off.Y, 20, g.World.Height-20))
			typ := "move"
			if attackMove {
				typ = "attackMove"
			}
			u.Order = &Order{Type: typ, X: goal.X, Y: goal.Y}
		}
		u.ResumeOrder = nil
		u.Path = nil
		u.PathTimer = 0
		u.DeployProgress = 0
		u.AvoidSide = 0
		u.AvoidUntil = 0
	}
	g.Effects = append(g.Effects, &Effect{Type: "order", X: x, Y: y, Age: 0, Duration: 0.6, Owner: side})
	g.OrderMode = ""
	return true
}

// cacheID/resource 站 id 是字符串；Go Order.TargetID 为 int——资源站用负索引编码（collect 分支经距离重匹配）。
func cacheID(id string) int { return 0 }

// StopSelected 对齐 game.js L424 stopSelected。
func (g *Game) StopSelected(side int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	hasUnit := false
	for _, id := range g.Selected {
		if u := g.GetUnit(id); u != nil {
			hasUnit = true
			break
		}
	}
	if hasUnit && g.Events.Voice != nil {
		g.Events.Voice("stopOrder")
	}
	for _, id := range g.Selected {
		u := g.GetUnit(id)
		if u == nil || u.Owner != side || freightTypes[u.Type] {
			continue
		}
		u.Order = nil
		u.ResumeOrder = nil
		u.Path = nil
		if u.Type == "supply" {
			f := false
			u.AutoSupply = &f
		}
	}
	g.OrderMode = ""
	return true
}

// ToggleAutoSupply 对齐 game.js L431。
func (g *Game) ToggleAutoSupply(u *Unit) bool {
	if u == nil || u.Type != "supply" || g.IsControlLocked(u.Owner) {
		return false
	}
	v := u.AutoSupply == nil || *u.AutoSupply == false
	u.AutoSupply = &v
	u.ServiceTargetID = 0
	return true
}

// ResolveMoveGoal 对齐 game.js L811-824。
func (g *Game) ResolveMoveGoal(u *Unit, x, y float64) Point {
	if gpdata.HasTag(u.Type, "ship") {
		return g.navalGoal(x, y, unitRadiusOf(u.Type))
	}
	radius := 14.0
	if gpdata.HasTag(u.Type, "infantry") {
		radius = 6
	}
	if gpdata.HasTag(u.Type, "air") || !g.isGroundBlocked(x, y, radius) {
		return Point{x, y}
	}
	// 网格扫描最近可达格。
	var best *Point
	bestScore := math.Inf(1)
	cols := int(g.World.Width / g.World.Cell)
	rows := int(g.World.Height / g.World.Cell)
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			cand := Point{float64(cx)*g.World.Cell + g.World.Cell/2, float64(cy)*g.World.Cell + g.World.Cell/2}
			if g.isGroundBlocked(cand.X, cand.Y, radius) {
				continue
			}
			score := Distance(cand.X, cand.Y, x, y) + Distance(cand.X, cand.Y, u.X, u.Y)*0.15
			if score < bestScore {
				cp := cand
				best = &cp
				bestScore = score
			}
		}
	}
	if best != nil {
		return *best
	}
	return Point{u.X, u.Y}
}

// SetRallyPoint 对齐 game.js L620-631。
func (g *Game) SetRallyPoint(side, id int, x, y float64) bool {
	if g.IsControlLocked(side) {
		return false
	}
	b := g.GetBuilding(id)
	if b == nil || b.Owner != side || b.HP <= 0 || !isProducer(b.Type) {
		return false
	}
	point := Point{clamp(x, 30, g.World.Width-30), clamp(y, 30, g.World.Height-30)}
	if b.Type == "dock" {
		point = g.navalGoal(point.X, point.Y, 95)
	} else if b.Type != "airfield" && g.isGroundBlocked(point.X, point.Y, 14) {
		return false
	}
	b.RallyPoint = &point
	return true
}
