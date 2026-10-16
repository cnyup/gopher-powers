// commands.go 对齐 server/commands.mjs executeCommand。
package server

import (
	"math"

	"github.com/cnyup/gopher-powers/internal/game"
)

// ExecuteCommand 指令白名单执行（协议冻结面）。
func ExecuteCommand(g *game.Game, side int, msg map[string]any) bool {
	typ, _ := msg["unitType"].(string)
	if side < 0 || side >= len(g.Players) || g.Players[side].Eliminated || !g.Running || g.Paused || g.IsControlLocked(side) {
		return false
	}
	own := func(id float64) any {
		i := int(id)
		if u := g.GetUnit(i); u != nil && u.Owner == side && u.HP > 0 {
			return u
		}
		if b := g.GetBuilding(i); b != nil && b.Owner == side && b.HP > 0 {
			return b
		}
		return nil
	}
	point := func() bool {
		x, okx := msg["x"].(float64)
		y, oky := msg["y"].(float64)
		return okx && oky && !math.IsInf(x, 0) && !math.IsNaN(x) && !math.IsInf(y, 0) && !math.IsNaN(y) &&
			x >= 0 && y >= 0 && x <= g.World.Width && y <= g.World.Height
	}
	var selected []int
	if arr, ok := msg["selected"].([]any); ok && len(arr) <= 120 {
		for _, v := range arr {
			if f, ok := v.(float64); ok && f == math.Trunc(f) && f >= -9007199254740991 && f <= 9007199254740991 {
				id := int(f)
				if e := own(f); e != nil {
					if u, isU := e.(*game.Unit); isU {
						_ = u
						selected = append(selected, id)
					} else if b, isB := e.(*game.Building); isB {
						_ = b
					}
				}
			}
		}
	}
	idF, _ := msg["id"].(float64)
	entity := own(idF)
	action, _ := msg["action"].(string)
	switch action {
	case "build":
		return game.ValidBuilding(typ) && g.StartBuild(side, typ)
	case "cancel-build":
		return g.CancelBuilding(side)
	case "place":
		x, _ := msg["x"].(float64)
		y, _ := msg["y"].(float64)
		return game.ValidBuilding(typ) && point() && g.PlaceBuilding(side, typ, x, y)
	case "train":
		count := len(g.OwnedUnitsList(side))
		for _, b := range g.OwnedBuildingsList(side) {
			count += len(b.Queue)
			if b.Active != nil {
				count++
			}
		}
		return game.ValidUnit(typ) && count < 120 && g.QueueUnit(side, typ)
	case "cancel-train":
		if b, ok := entity.(*game.Building); ok {
			return g.CancelUnitProduction(side, b.ID)
		}
		return false
	case "repair":
		if b, ok := entity.(*game.Building); ok {
			return g.ToggleRepair(side, b.ID)
		}
		return false
	case "sell":
		switch e := entity.(type) {
		case *game.Building:
			return g.SellBuilding(side, e.ID)
		case *game.Unit:
			return g.SellHarvester(side, e.ID)
		}
		return false
	case "rally":
		x, _ := msg["x"].(float64)
		y, _ := msg["y"].(float64)
		if b, ok := entity.(*game.Building); ok {
			return point() && g.SetRallyPoint(side, b.ID, x, y)
		}
		return false
	case "resupply":
		if u, ok := entity.(*game.Unit); ok {
			return g.RequestResupplyDirect(u)
		}
		return false
	case "unload":
		if u, ok := entity.(*game.Unit); ok {
			return g.UnloadTransport(u, false) > 0
		}
		return false
	case "auto-supply":
		if u, ok := entity.(*game.Unit); ok {
			return g.ToggleAutoSupply(u)
		}
		return false
	case "group-resupply", "group-unload":
		any := false
		for _, id := range selected {
			e := own(float64(id))
			if u, ok := e.(*game.Unit); ok {
				var ok2 bool
				if action == "group-resupply" {
					ok2 = g.RequestResupplyDirect(u)
				} else {
					ok2 = g.UnloadTransport(u, false) > 0
				}
				if ok2 {
					any = true
				}
			}
		}
		return any
	case "ability":
		x, _ := msg["x"].(float64)
		y, _ := msg["y"].(float64)
		return point() && g.CastAbility(x, y, side)
	case "satellite":
		return g.ActivateSatellite(side)
	case "cyber":
		return g.LaunchCyber(side)
	case "move", "stop":
		if len(selected) == 0 || (action == "move" && !point()) {
			return false
		}
		x, _ := msg["x"].(float64)
		y, _ := msg["y"].(float64)
		saved := g.Selected
		g.Selected = selected
		mode, _ := msg["mode"].(string)
		if mode == "patrol" {
			g.OrderMode = "patrol"
		} else {
			g.OrderMode = ""
		}
		var ok bool
		if action == "stop" {
			g.StopSelected(side)
			ok = true
		} else {
			ok = g.Command(x, y, msg["attackMove"] == true, side)
		}
		g.Selected = saved
		g.OrderMode = ""
		_ = ok
		return true
	default:
		return false
	}
}
