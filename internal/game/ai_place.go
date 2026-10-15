package game

import (
	"github.com/cnyup/gopher-powers/internal/game/equip"
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// equipProfileOf 桥接 equip 包（避免 import 环）。
func equipProfileOf(g *Game, typ string) *equip.Profile {
	return equip.ProfileOf(g.Players[0].Faction, typ, g.Map.Future)
}

// placeAIBuilding 对齐 game.js L1347-1362。
func (g *Game) placeAIBuilding(typ string, side int) {
	if typ == "dock" {
		zone := g.Map.DockZones[side]
		for _, dy := range []float64{0, -80, 80} {
			if g.PlaceBuilding(side, typ, (zone.X1+zone.X2)/2, g.HomeY+dy) {
				return
			}
		}
		g.Players[side].Credits += gpdata.BUILDINGS[typ].Cost * 0.6
		return
	}
	index := g.Players[side].AIPlanIndex - 1
	if index < 0 {
		index = 0
	}
	if index >= len(aiSlots) {
		index = len(aiSlots) - 1
	}
	preferred := aiSlots[index]
	var ox, oy float64
	if g.PlayerCount == 6 {
		p := gpdata.SixPoint(side, preferred[0]-205, preferred[1]-720)
		ox, oy = p.X, p.Y
	} else {
		ox = g.World.Width - preferred[0]
		oy = g.World.Height - preferred[1] - g.HomeY + 720
	}
	if g.PlaceBuilding(side, typ, ox, oy) {
		return
	}
	for radius := 0; radius <= 7; radius++ {
		for a := 0; a < 16; a++ {
			theta := float64(a) / 16 * 2 * 3.141592653589793
			x := ox + cos(theta)*float64(radius)*50
			y := oy + sin(theta)*float64(radius)*50
			if g.PlaceBuilding(side, typ, x, y) {
				return
			}
		}
	}
	g.Players[side].Credits += gpdata.BUILDINGS[typ].Cost * 0.6
}

// launchAIWave 对齐 game.js L1364-1372。
func (g *Game) launchAIWave(side int) {
	if g.IsControlLocked(side) {
		return
	}
	var combat []*Unit
	for _, u := range g.ActiveUnits(side, "") {
		if (gpdata.UNITS[u.Type].Damage > 0 || u.Type == "apc") && !gpdata.HasTag(u.Type, "ship") && !gpdata.HasTag(u.Type, "jet") {
			combat = append(combat, u)
		}
	}
	if len(combat) < 3 {
		return
	}
	var wave []*Unit
	for _, u := range combat {
		if u.Order != nil && (u.Order.Type == "attackMove" || u.Order.Type == "rearm" || u.Order.Type == "board") {
			continue
		}
		wave = append(wave, u)
		if len(wave) >= g.Difficulty.WaveSize {
			break
		}
	}
	if len(wave) == 0 {
		return
	}
	var objective *Point
	if g.VictoryMode == "control" {
		var best *Site
		bestD := 1e18
		ref := wave[0]
		for _, s := range g.Beacons {
			if s.Owner == side || !g.HasExploredFor(side, s.X, s.Y) {
				continue
			}
			if d := Distance(s.X, s.Y, ref.X, ref.Y); d < bestD {
				bestD, best = d, s
			}
		}
		if best != nil {
			objective = &Point{best.X, best.Y}
		}
	}
	dest := objective
	if dest == nil {
		d := g.enemyObjective(side)
		dest = &d
	}
	for _, u := range wave {
		u.Order = &Order{Type: "attackMove", X: dest.X + g.Random(-70, 70), Y: dest.Y + g.Random(-90, 90)}
		u.Path = nil
	}
	if g.Events.Notice != nil {
		if g.IsVisibleFor(0, wave[0].X, wave[0].Y) {
			if objective != nil {
				g.Events.Notice("敌军正在争夺信标")
			} else {
				g.Events.Notice("敌军正在向基地推进")
			}
		}
	}
}
