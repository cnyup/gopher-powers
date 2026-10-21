package game

import gpdata "github.com/cnyup/gopher-powers/internal/game/data"

// placeexports.go：前端放置预检所需的只读规则面。
//
// 源项目 render.js:1177 每帧调 game.canPlace(0,type,pointer.x,pointer.y) 决定
// ghost 红/绿——放置合法性判断本来就在客户端本地做。我们的前端在浏览器里跑，
// 模拟在 Go 进程内，所以把 canPlace 里前端拿不到的那部分输入下发：
// 世界尺寸、地图禁建区、船坞区、建筑尺寸表。
// 其余判据输入（矿/油/信标/资源站/现有建筑）快照里已有，前端直接复用，不重复下发。

// PlaceRuleRect 轴对齐矩形（世界坐标）。
type PlaceRuleRect struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

// PlaceRules 放置预检规则（对齐 CanPlace 中前端无法自行推导的输入）。
type PlaceRules struct {
	WorldWidth  float64            `json:"worldWidth"`
	WorldHeight float64            `json:"worldHeight"`
	NoBuild     []PlaceRuleRect    `json:"noBuild"`
	DockZones   []PlaceRuleRect    `json:"dockZones"`
	Sizes       map[string]float64 `json:"sizes"`
}

// PlaceRulesFor 生成放置预检规则。
// 视角方恒为快照 side 0，CanPlace 的 dock 分区分支（side==0 → 左半图）在前端等价于 side 0，
// 故不额外下发侧别。
func (g *Game) PlaceRulesFor(side int) PlaceRules {
	rect := func(r gpdata.Rect) PlaceRuleRect {
		return PlaceRuleRect{X1: r.X1, Y1: r.Y1, X2: r.X2, Y2: r.Y2}
	}
	out := PlaceRules{
		WorldWidth:  g.World.Width,
		WorldHeight: g.World.Height,
		Sizes:       make(map[string]float64, len(gpdata.BUILDINGS)),
	}
	for k, v := range gpdata.BUILDINGS {
		out.Sizes[k] = v.Size
	}
	for _, r := range g.Map.NoBuild {
		out.NoBuild = append(out.NoBuild, rect(r))
	}
	for _, r := range g.Map.DockZones {
		out.DockZones = append(out.DockZones, rect(r))
	}
	return out
}
