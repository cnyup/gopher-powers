package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// serverexports.go：服务端指令面所需导出（对齐 commands.mjs 依赖的 game 公有面）。

// ValidMap2P 双人图校验（非六方图）。
func ValidMap2P(mapID string) bool {
	m, ok := gpdata.MAPS[mapID]
	return ok && m.MaxPlayers != 6
}

// ValidFaction 阵营校验。
func ValidFaction(f string) bool {
	_, ok := gpdata.FACTIONS[f]
	return ok
}

// ValidVictoryMode 胜利模式校验。
func ValidVictoryMode(v string) bool {
	_, ok := gpdata.VICTORY_MODES[v]
	return ok
}

// ValidBuilding 建筑表校验。
func ValidBuilding(t string) bool {
	_, ok := gpdata.BUILDINGS[t]
	return ok
}

// ValidUnit 单位表校验。
func ValidUnit(t string) bool {
	_, ok := gpdata.UNITS[t]
	return ok
}

// OwnedUnitsList 某方全部单位。
func (g *Game) OwnedUnitsList(side int) []*Unit {
	return g.OwnedUnits(side, "")
}

// SellBuilding 对齐 game.js:299 sellBuilding。
func (g *Game) SellBuilding(side, id int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	b := g.GetBuilding(id)
	if b == nil || b.Owner != side || b.Type == "hq" || b.HP <= 0 {
		return false
	}
	refund := math.Floor(gpdata.BUILDINGS[b.Type].Cost * .5)
	g.Players[side].Credits += refund
	b.HP = 0
	g.Effects = append(g.Effects, &Effect{Type: "build", X: b.X, Y: b.Y, Age: 0, Duration: .8, Owner: side})
	g.RecalculatePower()
	return true
}

// SellHarvester 对齐 game.js:316 sellHarvester。
func (g *Game) SellHarvester(side, id int) bool {
	if g.IsControlLocked(side) {
		return false
	}
	u := g.GetUnit(id)
	if u == nil || u.Type != "harvester" || u.Owner != side || u.HP <= 0 || u.EmbarkedIn != 0 {
		return false
	}
	refund := math.Floor(g.unitCost(side, u.Type) * .5 * u.HP / u.MaxHP)
	g.Players[side].Credits += refund
	u.HP = 0
	return true
}

// RequestResupplyDirect 对齐 modern-combat.js:69 requestResupply(u)（无 home）。
func (g *Game) RequestResupplyDirect(u *Unit) bool {
	return g.RequestResupplyFull(u, nil)
}

// CanPlacePublic 对齐 canPlace（前端放置预检用；服务端 place 仍独立校验）。
func (g *Game) CanPlacePublic(side int, typ string, x, y float64) bool {
	return g.CanPlace(side, typ, x, y)
}
