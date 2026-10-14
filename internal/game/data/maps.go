package data

import "math"

// Rect 对齐源项目矩形字段（x1/y1/x2/y2）。
type Rect struct {
	X1, Y1, X2, Y2 float64
}

// Point 简单坐标。
type Point struct {
	X, Y float64
}

// Site 对齐 oil/beacons 条目（id/x/y）。
type Site struct {
	ID string
	X  float64
	Y  float64
}

// Bridge 对齐 bridges 条目（y1/y2 横跨水域的桥面段）。
type Bridge struct {
	Y1, Y2 float64
}

// MapSpec 对齐 MAPS 条目。
type MapSpec struct {
	Name         string
	Sector       string
	Future       bool
	World        World
	MaxPlayers   int
	EconomyMode  string
	Spawns       []Spawn
	Ore          []OreEntry
	Oil          []Site
	Beacons      []Site
	Barriers     []Rect
	NoBuild      []Rect
	Bridges      []Bridge
	Water        *Rect
	DockZones    []Rect
	Cover        []Rect
}

// Spawn 对齐 six-player-map.js SIX_SPAWNS 条目。
type Spawn struct {
	Side  int
	X, Y  float64
	Angle float64
}

// SIX_WORLD 对齐 six-player-map.js:2。
var SIX_WORLD = World{Width: 5760, Height: 5760, Cell: 40, Fog: 80}

var sixCenter = Point{X: 2880, Y: 2880}

// SixSpawns 对齐 six-player-map.js:4：六席按 π/3 逆时针分布于半径 1920 圆周。
var SixSpawns = func() []Spawn {
	spawns := make([]Spawn, 6)
	for side := 0; side < 6; side++ {
		angle := -math.Pi/2 + float64(side)*math.Pi/3
		spawns[side] = Spawn{
			Side:  side,
			X:     sixCenter.X + math.Cos(angle)*1920,
			Y:     sixCenter.Y + math.Sin(angle)*1920,
			Angle: angle + math.Pi,
		}
	}
	return spawns
}()

// SixPoint 对齐 six-player-map.js:5：局部模板坐标按席位朝向旋转。
func SixPoint(side int, dx, dy float64) Point {
	s := SixSpawns[side]
	return Point{
		X: s.X + math.Cos(s.Angle)*dx - math.Sin(s.Angle)*dy,
		Y: s.Y + math.Sin(s.Angle)*dx + math.Cos(s.Angle)*dy,
	}
}

// sixPlayerMap 对齐 six-player-map.js:6-12。
func sixPlayerMap() MapSpec {
	ore := make([]OreEntry, 0, 30)
	for i := 0; i < 6; i++ {
		for _, e := range []OreEntry{
			{430, -320, 4000, "gold"}, {700, -340, 4000, "gold"}, {550, 400, 4000, "gold"}, {760, 0, 2800, "gem"},
		} {
			p := SixPoint(i, e.X, e.Y)
			ore = append(ore, OreEntry{p.X, p.Y, e.Amount, e.Kind})
		}
	}
	oil := make([]Site, 0, 6)
	beacons := make([]Site, 0, 6)
	for i := 0; i < 6; i++ {
		a := -math.Pi/2 + float64(i)*math.Pi/3
		ore = append(ore, OreEntry{sixCenter.X + math.Cos(a)*520, sixCenter.Y + math.Sin(a)*520, 5000, "gem"})
		oil = append(oil, Site{ID: "six-oil-" + itoa(i), X: sixCenter.X + math.Cos(a+math.Pi/6)*1140, Y: sixCenter.Y + math.Sin(a+math.Pi/6)*1140})
		beacons = append(beacons, Site{ID: "six-beacon-" + itoa(i), X: sixCenter.X + math.Cos(a+math.Pi/6)*660, Y: sixCenter.Y + math.Sin(a+math.Pi/6)*660})
	}
	return MapSpec{
		Name: "六域会战", Sector: "候选战区08 · 六方自由对战", World: SIX_WORLD, MaxPlayers: 6,
		EconomyMode: "mining", Spawns: SixSpawns, Ore: ore, Oil: oil, Beacons: beacons,
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

// MAPS 对齐 data.js MAPS（sixfront 用六方生成器，其余静态表）。
var MAPS = map[string]*MapSpec{
	"sixfront": func() *MapSpec { m := sixPlayerMap(); return &m }(),
	"archipelago": {
		Name: "跨海登陆战", Sector: "战区 07 · 无桥海战", World: World{Width: 3520, Height: 2240, Cell: 40, Fog: 80},
		Oil:  []Site{{ID: "oil-a", X: 890, Y: 240}, {ID: "oil-b", X: 2630, Y: 2000}},
		Beacons: []Site{{ID: "beacon-west", X: 870, Y: 700}, {ID: "beacon-east", X: 2650, Y: 1540}},
		Barriers:  []Rect{{1040, 0, 2480, 2240}},
		NoBuild:   []Rect{{1000, 0, 2520, 2240}},
		Water:     &Rect{1040, 0, 2480, 2240},
		DockZones: []Rect{{900, 970, 1000, 1270}, {2520, 970, 2620, 1270}},
	},
	"meridian": {
		Name: "子午环阵", Sector: "战区 06 · 原创科幻", Future: true, World: World{Width: 3200, Height: 2080, Cell: 40, Fog: 80},
		Ore: []OreEntry{
			{330, 830, 2700, "gold"}, {330, 1250, 2700, "gold"}, {870, 400, 2100, "gem"}, {870, 1680, 2100, "gem"},
			{1150, 1040, 3100, "gold"}, {2050, 1040, 3100, "gold"}, {2330, 400, 2100, "gem"}, {2330, 1680, 2100, "gem"},
			{2870, 830, 2700, "gold"}, {2870, 1250, 2700, "gold"},
		},
		Oil:     []Site{{ID: "oil-a", X: 940, Y: 650}, {ID: "oil-b", X: 2260, Y: 1430}},
		Beacons: []Site{{ID: "beacon-north", X: 1600, Y: 400}, {ID: "beacon-center", X: 1600, Y: 1040}, {ID: "beacon-south", X: 1600, Y: 1680}},
		Barriers: []Rect{{1480, 620, 1720, 900}, {1480, 1180, 1720, 1460}},
		NoBuild:  []Rect{{1420, 580, 1780, 940}, {1420, 1140, 1780, 1500}},
		Cover:    []Rect{{930, 760, 1110, 920}, {2090, 1160, 2270, 1320}, {1180, 1420, 1380, 1550}, {1820, 530, 2020, 660}},
	},
	"frontier": {
		Name: "纵深战区", Sector: "战区 05 · 纵深陆战", World: World{Width: 4480, Height: 2880, Cell: 40, Fog: 80},
		Ore: []OreEntry{
			{330, 1230, 2600, "gold"}, {330, 1650, 2600, "gold"}, {860, 660, 1800, "gem"}, {860, 2220, 1800, "gem"},
			{1420, 1160, 3200, "gold"}, {1420, 1720, 3200, "gold"}, {3060, 1160, 3200, "gold"}, {3060, 1720, 3200, "gold"},
			{3620, 660, 1800, "gem"}, {3620, 2220, 1800, "gem"}, {4150, 1230, 2600, "gold"}, {4150, 1650, 2600, "gold"},
		},
		Oil:     []Site{{ID: "oil-a", X: 1320, Y: 420}, {ID: "oil-b", X: 3160, Y: 2460}, {ID: "oil-c", X: 1320, Y: 2460}, {ID: "oil-d", X: 3160, Y: 420}},
		Beacons: []Site{{ID: "beacon-north", X: 2240, Y: 820}, {ID: "beacon-south", X: 2240, Y: 2060}},
	},
	"ocean": {
		Name: "远洋战区", Sector: "战区 04 · 大型海战", World: World{Width: 3520, Height: 2240, Cell: 40, Fog: 80},
		Ore: []OreEntry{
			{330, 910, 3000, "gold"}, {330, 1330, 3000, "gold"}, {760, 400, 2200, "gem"}, {760, 1840, 2200, "gem"},
			{780, 1120, 3200, "gold"}, {2740, 1120, 3200, "gold"}, {2760, 400, 2200, "gem"}, {2760, 1840, 2200, "gem"},
			{3190, 910, 3000, "gold"}, {3190, 1330, 3000, "gold"},
		},
		Oil:     []Site{{ID: "oil-a", X: 890, Y: 240}, {ID: "oil-b", X: 2630, Y: 2000}},
		Beacons: []Site{{ID: "beacon-north", X: 1760, Y: 480}, {ID: "beacon-south", X: 1760, Y: 1760}},
		Barriers: []Rect{{1040, 0, 2480, 360}, {1040, 600, 2480, 1640}, {1040, 1880, 2480, 2240}},
		NoBuild:  []Rect{{1000, 0, 2520, 2240}},
		Bridges:  []Bridge{{360, 600}, {1640, 1880}},
		Water:    &Rect{1040, 0, 2480, 2240},
		DockZones: []Rect{{900, 970, 1000, 1270}, {2520, 970, 2620, 1270}},
	},
	"valley": {
		Name: "灰谷地带", Sector: "战区 01", Ore: ORE_LAYOUT,
		Oil:     []Site{{ID: "oil-a", X: 1110, Y: 210}, {ID: "oil-b", X: 1130, Y: 1230}},
		Beacons: []Site{{ID: "beacon-center", X: 1120, Y: 720}},
	},
	"canyon": {
		Name: "双桥裂谷", Sector: "战区 02",
		Ore: []OreEntry{
			{330, 510, 2100, "gold"}, {330, 930, 2100, "gold"},
			{760, 320, 1500, "gem"}, {760, 1120, 1500, "gem"},
			{820, 700, 2500, "gold"}, {1420, 740, 2500, "gold"},
			{1480, 320, 1500, "gem"}, {1480, 1120, 1500, "gem"},
			{1910, 510, 2100, "gold"}, {1910, 930, 2100, "gold"},
		},
		Oil:     []Site{{ID: "oil-a", X: 870, Y: 225}, {ID: "oil-b", X: 1370, Y: 1215}},
		Beacons: []Site{{ID: "beacon-north", X: 1120, Y: 410}, {ID: "beacon-south", X: 1120, Y: 1030}},
		Barriers: []Rect{{1040, 0, 1200, 320}, {1040, 500, 1200, 940}, {1040, 1120, 1200, 1440}},
		NoBuild:  []Rect{{1000, 0, 1240, 1440}},
		Bridges:  []Bridge{{320, 500}, {940, 1120}},
	},
	"strait": {
		Name: "海峡前线", Sector: "战区 03",
		Ore: []OreEntry{
			{330, 510, 2100, "gold"}, {330, 930, 2100, "gold"},
			{760, 320, 1500, "gem"}, {760, 1120, 1500, "gem"},
			{740, 700, 2500, "gold"}, {1500, 740, 2500, "gold"},
			{1480, 320, 1500, "gem"}, {1480, 1120, 1500, "gem"},
			{1910, 510, 2100, "gold"}, {1910, 930, 2100, "gold"},
		},
		Oil:     []Site{{ID: "oil-a", X: 860, Y: 220}, {ID: "oil-b", X: 1380, Y: 1220}},
		Beacons: []Site{{ID: "beacon-north", X: 1120, Y: 420}, {ID: "beacon-south", X: 1120, Y: 1020}},
		Barriers: []Rect{{960, 0, 1280, 340}, {960, 500, 1280, 940}, {960, 1100, 1280, 1440}},
		NoBuild:  []Rect{{920, 0, 1320, 1440}},
		Bridges:  []Bridge{{340, 500}, {940, 1100}},
		Water:    &Rect{960, 0, 1280, 1440},
		DockZones: []Rect{{800, 590, 900, 850}, {1340, 590, 1440, 850}},
	},
}
