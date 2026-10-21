package data

import (
	"errors"
	"slices"
)

// BattlefieldMap 对齐 battlefield-expansion.js:4-14：双人图按 1.5 缩放为纵深版。
// 返回新副本，不改动源表；六方图与 scale=1 原样返回。
func BattlefieldMap(mapID string, scale float64) (*MapSpec, error) {
	source, ok := MAPS[mapID]
	if !ok {
		return nil, errors.New("未知地图")
	}
	if scale == 1 || source.MaxPlayers == 6 {
		return source, nil
	}
	if scale != 1.5 {
		return nil, errors.New("不支持的战场尺寸")
	}
	rect := func(r Rect) Rect { return Rect{r.X1 * scale, r.Y1 * scale, r.X2 * scale, r.Y2 * scale} }
	sites := func(vs []Site) []Site {
		out := make([]Site, len(vs))
		for i, v := range vs {
			out[i] = Site{ID: v.ID, X: v.X * scale, Y: v.Y * scale}
		}
		return out
	}
	world := source.World
	if world.Width == 0 {
		world = WORLD
	}
	m := &MapSpec{
		Name: source.Name + " · 纵深",
		World: World{Width: world.Width * scale, Height: world.Height * scale, Cell: world.Cell, Fog: world.Fog},
	}
	m.Ore = make([]OreEntry, len(source.Ore))
	for i, e := range source.Ore {
		m.Ore[i] = OreEntry{e.X * scale, e.Y * scale, e.Amount, e.Kind}
	}
	m.Oil = sites(source.Oil)
	m.Beacons = sites(source.Beacons)
	m.Barriers = make([]Rect, len(source.Barriers))
	for i, r := range source.Barriers {
		m.Barriers[i] = rect(r)
	}
	m.NoBuild = make([]Rect, len(source.NoBuild))
	for i, r := range source.NoBuild {
		m.NoBuild[i] = rect(r)
	}
	m.Bridges = make([]Bridge, len(source.Bridges))
	for i, b := range source.Bridges {
		m.Bridges[i] = Bridge{Y1: b.Y1 * scale, Y2: b.Y2 * scale}
	}
	if source.Water != nil {
		w := rect(*source.Water)
		m.Water = &w
	}
	m.DockZones = make([]Rect, len(source.DockZones))
	for i, r := range source.DockZones {
		m.DockZones[i] = rect(r)
	}
	m.Cover = make([]Rect, len(source.Cover))
	for i, r := range source.Cover {
		m.Cover[i] = rect(r)
	}
	return m, nil
}

// ResourceType 对齐 battlefield-expansion.js RESOURCE_TYPES。
type ResourceType struct {
	Name        string
	Amount      float64
	Description string
}

var RESOURCE_TYPES = map[string]ResourceType{
	"cache":   {Name: "应急物资箱", Amount: 300, Description: "陆地单位停驻回收；共 300 资金，先侦察再争夺。"},
	"salvage": {Name: "设备回收场", Amount: 900, Description: "工程师、采矿车或补给车以每秒 45 资金回收，共 900；不在常规战场挖矿。"},
	"depot":   {Name: "中立能源仓", Amount: 1800, Description: "工程师接管后提供 40 电力与前哨视野，按每秒 8 资金转移库存；可依托扩建基地，敌方能重新夺取。"},
}

// ResourceSite 对齐 sixResources/resourceLayout 产出条目。
// json tag 必须显式声明：这些字段经 battleView 直接 JSON 序列化给前端，
// 缺 tag 会输出 Go 字段名（ID/X/Owner…），前端读 site.id 得到 undefined。
type ResourceSite struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"` // 恒为 "resource"
	Type     string  `json:"type"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Owner    int     `json:"owner"` // -1 = null
	Amount   float64 `json:"amount"`
	Max      float64 `json:"max"`
	Progress float64 `json:"progress"`
}

// sixResources 对齐 six-player-map.js:13。
func sixResources() []ResourceSite {
	out := make([]ResourceSite, 0, 18)
	for side := 0; side < 6; side++ {
		for _, e := range []struct {
			Type   string
			X, Y   float64
			Amount float64
		}{
			{"cache", 520, 190, 450}, {"salvage", 600, -180, 1200}, {"depot", 1050, 0, 2400},
		} {
			p := SixPoint(side, e.X, e.Y)
			out = append(out, ResourceSite{
				ID: "six-resource-" + itoa(side) + "-" + e.Type, Kind: "resource", Type: e.Type,
				X: p.X, Y: p.Y, Owner: -1, Amount: e.Amount, Max: e.Amount,
			})
		}
	}
	return out
}

// ResourceLayout 对齐 battlefield-expansion.js:17-25。
func ResourceLayout(m *MapSpec) []ResourceSite {
	if m.MaxPlayers == 6 {
		return sixResources()
	}
	world := m.World
	if world.Width == 0 {
		world = WORLD
	}
	w, h := world.Width, world.Height
	landX := w * .40
	if m.Water != nil {
		landX = m.Water.X1 * .68
	}
	spots := []struct {
		Type string
		X, Y float64
	}{
		{"cache", landX, h * .27},
		{"salvage", landX * .92, h * .73},
		{"depot", landX + 55, h * .36},
	}
	out := make([]ResourceSite, 0, 6)
	for _, sp := range spots {
		for side := 0; side < 2; side++ {
			x, y := sp.X, sp.Y
			if side == 1 {
				x, y = w-x, h-y
			}
			out = append(out, ResourceSite{
				ID: "resource-" + sp.Type + "-" + itoa(side), Kind: "resource", Type: sp.Type,
				X: x, Y: y, Owner: -1, Amount: RESOURCE_TYPES[sp.Type].Amount, Max: RESOURCE_TYPES[sp.Type].Amount,
			})
		}
	}
	return out
}

// HasTag 判断单位类型是否带指定 tag（对齐 JS tags.includes）。
func HasTag(typ, tag string) bool {
	return slices.Contains(UNITS[typ].Tags, tag)
}
