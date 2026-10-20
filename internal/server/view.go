// Package server 对齐 server/*.mjs。
// view.go 对齐 battle-view.mjs：席位重排 + 敌方字段剥离 + 建筑记忆 + fog 打包。
package server

import (
	"encoding/json"

	"github.com/cnyup/gopher-powers/internal/game"
	"github.com/cnyup/gopher-powers/internal/protocol"
)

// viewMemory 建筑记忆（room 内每席一份）。key 为建筑 id（int）。
type viewMemory map[int]json.RawMessage

func copyJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// battleView 对齐 battleView(game, side, memory, {spectator})。
// 返回 map 以保持字段序/动态结构与 JS 快照同构（协议逐字段兼容锚点）。
func battleView(g *game.Game, side int, memory viewMemory, spectator bool) map[string]any {
	order := seatOrder(g, side)
	mapping := map[int]int{} // 实际 side → 快照 side
	for i, s := range order {
		mapping[s] = i
	}
	owner := func(v int) any {
		if v < 0 {
			return nil
		}
		if m, ok := mapping[v]; ok {
			return m
		}
		return v
	}
	visible := func(e game.TargetEntityView) bool { return g.CanSeeEntityView(side, e) }

	// 可见实体集（己方全部 + 敌方未登舰且可见）。
	var entities []*game.Unit
	ids := map[int]bool{}
	for _, u := range g.Units {
		if u.Owner == side || (u.EmbarkedIn == 0 && visible(game.UnitViewOf(u))) {
			entities = append(entities, u)
			ids[u.ID] = true
		}
	}
	entityView := func(e any) map[string]any {
		raw := copyJSON(e)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		if m == nil {
			return nil
		}
		if o, ok := m["owner"].(float64); ok {
			m["owner"] = owner(int(o))
		}
		snapSide := 0
		if os, ok := m["owner"].(int); ok {
			snapSide = os
		}
		if snapSide != 0 || spectator {
			// 敌方/观战剥离指令面。
			m["order"] = nil
			m["path"] = []any{}
			m["resumeOrder"] = nil
			m["targetId"] = nil
			m["passengers"] = []any{}
			m["embarkedIn"] = nil
			m["queue"] = []any{}
			m["active"] = nil
			m["rallyPoint"] = nil
			delete(m, "freight")
			delete(m, "serviceTargetId")
			delete(m, "stock")
			delete(m, "ammo")
		}
		return m
	}
	_ = ids

	// 建筑记忆更新：敌方可见建筑入记忆。
	for _, b := range g.Buildings {
		if b.Owner != side && visible(game.BuildingViewOf(b)) {
			memory[b.ID] = copyJSON(entityView(b))
		}
	}
	// 己方 sabotager 归属为 0 时清除敌建筑 sabotage 视图。
	for id, raw := range memory {
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		if sab, ok := m["sabotage"].(map[string]any); ok {
			if o, ok2 := sab["owner"].(float64); ok2 && int(o) == 0 {
				if e := g.GetBuilding(id); e == nil || e.Sabotage == nil || e.Sabotage.Owner != side {
					delete(m, "sabotage")
					memory[id] = copyJSON(m)
				}
			}
		}
	}
	// 记忆中已不可见且原地无建筑的条目删除（原位置可见时才删——消失确认）。
	for id := range memory {
		if id < 0 {
			continue // site 记忆（负数键）不参与建筑的消失确认。
		}
		b := g.GetBuilding(id)
		pos := buildingPosOf(memory[id])
		if b == nil && pos != nil && g.IsVisibleFor(side, pos.x, pos.y) {
			delete(memory, id)
		}
	}

	sites := func(list any) []any {
		// ore/oil/beacons/resourceSites 各自 json 同构处理。
		b, _ := json.Marshal(list)
		var arr []map[string]any
		_ = json.Unmarshal(b, &arr)
		out := make([]any, 0, len(arr))
		for _, s := range arr {
			id, _ := s["id"].(string)
			x, _ := s["x"].(float64)
			y, _ := s["y"].(float64)
			ownerV, hasOwner := s["owner"]
			ownerNum, hasOwnerNum := ownerV.(float64)
			here := g.IsVisibleFor(side, x, y)
			mine := hasOwnerNum && int(ownerNum) == side
			if here || mine || memorySite(memory, id) == nil {
				v := s
				// 迷雾遮蔽且非己方：隐藏真实归属与剩余量（对齐 battle-view.mjs:38）。
				if !here && hasOwnerNum && int(ownerNum) != side {
					if hasOwner {
						v["owner"] = nil
					}
					if _, hasAmount := s["amount"]; hasAmount {
						v["amount"] = s["max"]
					}
				}
				// 归属重排：读 v 当前值而非原始 ownerV——否则上一段的剥离会被覆盖回真实归属。
				if hasOwner {
					if cur, ok := v["owner"]; ok && cur != nil {
						if f, ok2 := cur.(float64); ok2 {
							v["owner"] = owner(int(f))
						}
					}
				}
				setSiteMemory(memory, id, v)
			}
			out = append(out, copyAny(siteMemory(memory, id)))
		}
		return out
	}

	players := make([]any, 0, len(order))
	for i, s := range order {
		p := g.Players[s]
		if i == 0 && !spectator {
			raw := copyJSON(p)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			m["side"] = 0
			m["actualSide"] = s
			m["controlLocked"] = g.IsControlLocked(s)
			players = append(players, m)
		} else {
			players = append(players, map[string]any{
				"side": i, "actualSide": s, "faction": p.Faction,
				"eliminated": p.Eliminated, "controlType": p.ControlType,
				"credits": 0, "powerIn": 0, "powerOut": 0,
				"controlScore": p.ControlScore, "buildQueue": nil, "readyBuilding": nil,
				"abilityCharge": 0, "abilityCooldown": 0,
				"cyberPending": cyberPendingView(p),
			})
		}
	}

	var unitsOut []any
	for _, u := range entities {
		unitsOut = append(unitsOut, entityView(u))
	}
	var buildingsOut []any
	for _, b := range g.OwnedBuildingsList(side) {
		buildingsOut = append(buildingsOut, entityView(b))
	}
	for id, raw := range memory {
		if id < 0 {
			continue // site 记忆用负数键，不属于建筑输出（对齐源 typeof id === 'number'）。
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			_ = id
			buildingsOut = append(buildingsOut, copyAny(m))
		}
	}

	fog := g.Fogs[side]
	var projs []any
	for _, p := range g.Projectiles {
		if g.IsVisibleFor(side, p.X, p.Y) && g.IsVisibleFor(side, p.ToX, p.ToY) {
			raw := copyJSON(p)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if !g.IsVisibleFor(side, p.StartX, p.StartY) {
				m["startX"] = p.X
				m["startY"] = p.Y
			}
			m["owner"] = owner(p.Owner)
			m["targetId"] = nil
			m["sourceId"] = nil
			projs = append(projs, m)
		}
	}
	var effects []any
	for _, e := range g.Effects {
		toX, toY := 0.0, 0.0
		hasTo := e.ToX != nil && e.ToY != nil
		if hasTo {
			toX, toY = *e.ToX, *e.ToY
		}
		if g.IsVisibleFor(side, e.X, e.Y) && (!hasTo || g.IsVisibleFor(side, toX, toY)) {
			raw := copyJSON(e)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			m["owner"] = owner(e.Owner)
			effects = append(effects, m)
		}
	}

	winner := any(g.Winner)
	if w, ok := g.Winner.(int); ok {
		winner = owner(w)
	}

	return map[string]any{
		"spectator":   spectator,
		"seatOrder":   order,
		"time":        g.Time,
		"running":     g.Running,
		"paused":      g.Paused,
		"winner":      winner,
		"players":     players,
		"units":       unitsOut,
		"buildings":   buildingsOut,
		"ore":         sites(g.Ore),
		"oil":         sites(g.Oil),
		"beacons":     sites(g.Beacons),
		"resourceSites": sites(g.ResourceSites),
		"fog":         protocol.PackFog(fog.Cols, fog.Rows, fog.Visible, fog.Explored),
		"projectiles": projs,
		"effects":     effects,
	}
}

func copyAny(m map[string]any) map[string]any {
	raw := copyJSON(m)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func seatOrder(g *game.Game, side int) []int {
	order := []int{side}
	for _, p := range g.Players {
		if p.Side != side {
			order = append(order, p.Side)
		}
	}
	return order
}

func cyberPendingView(p *game.Player) any {
	if p.CyberPending == nil {
		return nil
	}
	return map[string]any{"executeAt": p.CyberPending.ExecuteAt}
}

// site 记忆存独立键空间（负数编码避开建筑 id）。
func siteKey(id string) int {
	h := 0
	for _, c := range id {
		h = h*131 + int(c)
	}
	return -h - 1
}

func memorySite(memory viewMemory, siteID string) json.RawMessage {
	return memory[siteKey(siteID)]
}

func setSiteMemory(memory viewMemory, siteID string, v map[string]any) {
	memory[siteKey(siteID)] = copyJSON(v)
}

func siteMemory(memory viewMemory, siteID string) map[string]any {
	raw := memory[siteKey(siteID)]
	if raw == nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}
type xy struct{ x, y float64 }

func buildingPosOf(raw json.RawMessage) *xy {
	var m struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return &xy{m.X, m.Y}
}
