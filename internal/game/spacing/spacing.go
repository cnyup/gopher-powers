// Package spacing 对齐源项目 src/unit-spacing.js。
package spacing

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// Entity 是参与间距计算的最小单位视图（Game 的 Unit 会实现它）。
type Entity interface {
	ID() int
	Type() string
	X() float64
	Y() float64
	HP() float64
	StunUntil() float64
}

// Mobile 实体可被位移（separateUnits 修改坐标）。
type Mobile interface {
	Entity
	SetPos(x, y float64)
}

// UnitLayer 对齐 unit-spacing.js:4-7：jet > drone > submerged > ship > ground。
func UnitLayer(typ string) string {
	switch {
	case gpdata.HasTag(typ, "jet"):
		return "jet"
	case gpdata.HasTag(typ, "drone"):
		return "drone"
	case gpdata.HasTag(typ, "submerged"):
		return "submerged"
	case gpdata.HasTag(typ, "ship"):
		return "naval"
	default:
		return "ground"
	}
}

// UnitRadius 对齐 unit-spacing.js:9-14。
func UnitRadius(typ string) float64 {
	if gpdata.HasTag(typ, "infantry") {
		return 9
	}
	if gpdata.HasTag(typ, "jet") {
		switch typ {
		case "airlift":
			return 80
		case "bomber":
			return 76
		case "freightPlane":
			return 68
		}
		return 40
	}
	if gpdata.HasTag(typ, "drone") {
		return 20
	}
	if gpdata.HasTag(typ, "ship") {
		switch typ {
		case "carrier":
			return 92
		case "destroyer":
			return 65
		case "submarine":
			return 46
		case "frigate":
			return 48
		case "patrol":
			return 36
		case "landing":
			return 74
		case "containerShip":
			return 88
		}
		return 48
	}
	if typ == "harvester" {
		return 29
	}
	return 30
}

const cellSize = 224

// SpatialIndex 对齐 UnitSpatialIndex：层内 224 网格分桶。
type SpatialIndex struct {
	cells map[string][]Entity
}

func key(layer string, x, y int) string {
	return layer + ":" + itoa(x) + ":" + itoa(y)
}

// NewSpatialIndex 对齐 unit-spacing.js:17-27（跳过 hp<=0 与 embarked）。
func NewSpatialIndex(units []Entity) *SpatialIndex {
	idx := &SpatialIndex{cells: map[string][]Entity{}}
	for _, u := range units {
		if u.HP() <= 0 {
			continue
		}
		k := key(UnitLayer(u.Type()), floor(u.X()/cellSize), floor(u.Y()/cellSize))
		idx.cells[k] = append(idx.cells[k], u)
	}
	return idx
}

// Nearby 对齐 unit-spacing.js:29-37：3×3 邻域、排除自身与死亡单位。
func (idx *SpatialIndex) Nearby(u Entity) []Entity {
	var result []Entity
	x, y, layer := floor(u.X()/cellSize), floor(u.Y()/cellSize), UnitLayer(u.Type())
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for _, other := range idx.cells[key(layer, x+dx, y+dy)] {
				if other != u && other.HP() > 0 {
					result = append(result, other)
				}
			}
		}
	}
	return result
}

func floor(v float64) int {
	if v < 0 {
		return int(v) - 1
	}
	return int(v)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

// Point 目标坐标。
type Point struct{ X, Y float64 }

// FormationOffsets 对齐 unit-spacing.js:39-60：分层网格阵型 + 距离贪心分配。
// 返回 id → 偏移映射；units 需带 X/Y/ID/Type。
func FormationOffsets(units []Entity, target Point) map[int]Point {
	result := map[int]Point{}
	layers := map[string][]Entity{}
	var order []string
	for _, u := range units {
		layer := UnitLayer(u.Type())
		if _, ok := layers[layer]; !ok {
			order = append(order, layer)
		}
		layers[layer] = append(layers[layer], u)
	}
	for _, layer := range order {
		group := layers[layer]
		var cx, cy float64
		for _, u := range group {
			cx += u.X() / float64(len(group))
			cy += u.Y() / float64(len(group))
		}
		angle := mathAtan2(target.Y-cy, target.X-cx)
		forward := Point{X: mathCos(angle), Y: mathSin(angle)}
		right := Point{X: -forward.Y, Y: forward.X}
		n := len(group)
		columns := ceilSqrt(n)
		rows := (n + columns - 1) / columns
		maxR := 0.0
		for _, u := range group {
			if r := UnitRadius(u.Type()); r > maxR {
				maxR = r
			}
		}
		space := maxR*2 + 12
		type slot struct{ x, y float64 }
		var slots []slot
		for row := 0; row < rows; row++ {
			count := columns
			if left := n - row*columns; left < count {
				count = left
			}
			for col := 0; col < count; col++ {
				across := (float64(col) - float64(count-1)/2) * space
				depth := (float64(row) - float64(rows-1)/2) * space
				slots = append(slots, slot{across*right.X + depth*forward.X, across*right.Y + depth*forward.Y})
			}
		}
		var mx, my float64
		for _, s := range slots {
			mx += s.x / float64(len(slots))
			my += s.y / float64(len(slots))
		}
		remaining := make([]Entity, len(group))
		copy(remaining, group)
		sortByID(remaining)
		for _, s := range slots {
			offX, offY := s.x-mx, s.y-my
			// 稳定排序：距离升序，tie-break id 升序（对齐 JS sort 双比较器）。
			pick := 0
			for i := 1; i < len(remaining); i++ {
				di := distTo(remaining[i], target, offX, offY)
				dp := distTo(remaining[pick], target, offX, offY)
				if di < dp || (di == dp && remaining[i].ID() < remaining[pick].ID()) {
					pick = i
				}
			}
			chosen := remaining[pick]
			remaining = append(remaining[:pick], remaining[pick+1:]...)
			result[chosen.ID()] = Point{X: offX, Y: offY}
		}
	}
	return result
}

func distTo(u Entity, target Point, offX, offY float64) float64 {
	dx := u.X() - target.X - offX
	dy := u.Y() - target.Y - offY
	return dist(dx, dy)
}

func dist(dx, dy float64) float64 { return mathSqrt(dx*dx + dy*dy) }

func sortByID(us []Entity) {
	for i := 1; i < len(us); i++ {
		for j := i; j > 0 && us[j].ID() < us[j-1].ID(); j-- {
			us[j], us[j-1] = us[j-1], us[j]
		}
	}
}

func ceilSqrt(n int) int {
	r := isqrt(n)
	if r*r < n {
		return r + 1
	}
	return r
}

func isqrt(n int) int {
	r := 0
	for (r+1)*(r+1) <= n {
		r++
	}
	return r
}

// sortMobiles 按 id 升序（Mobile 版）。
func sortMobiles(us []Mobile) {
	for i := 1; i < len(us); i++ {
		for j := i; j > 0 && us[j].ID() < us[j-1].ID(); j-- {
			us[j], us[j-1] = us[j-1], us[j]
		}
	}
}

// SeparateUnits 对齐 unit-spacing.js:62-92：三轮迭代位移分离。
// canMove 报告 (unit, 新位置) 是否允许；embarked 由调用方过滤。
// 接口约定：units 必须全部为 Mobile。
func SeparateUnits(units []Mobile, dt float64, canMove func(Mobile, Point) bool, time float64) {
	var live []Mobile
	for _, u := range units {
		if u.HP() > 0 {
			live = append(live, u)
		}
	}
	sortMobiles(live)
	shift := func(u Mobile, dx, dy float64) bool {
		p := Point{X: u.X() + dx, Y: u.Y() + dy}
		if !canMove(u, p) {
			return false
		}
		u.SetPos(p.X, p.Y)
		return true
	}
	ents := make([]Entity, len(live))
	for i, m := range live {
		ents[i] = m
	}
	for pass := 0; pass < 3; pass++ {
		index := NewSpatialIndex(ents)
		for _, a := range live {
			for _, bEnt := range index.Nearby(a) {
				b, ok := bEnt.(Mobile)
				if !ok || a.ID() >= b.ID() {
					continue
				}
				dx, dy := b.X()-a.X(), b.Y()-a.Y()
				distance := dist(dx, dy)
				minimum := UnitRadius(a.Type()) + UnitRadius(b.Type()) + 2
				if distance >= minimum-.01 {
					continue
				}
				if distance < .001 {
					angle := float64((a.ID()*13+b.ID()*7)%32) * mathPi / 16
					dx, dy, distance = mathCos(angle), mathSin(angle), 1
				}
				push := (minimum - distance) / 2
				if alt := dt * 60 / 3; push > alt {
					push = alt
				}
				nx, ny := dx/distance, dy/distance
				aMobile := !(a.StunUntil() > time)
				bMobile := !(b.StunUntil() > time)
				movedA := aMobile && shift(a, -nx*push, -ny*push)
				movedB := bMobile && shift(b, nx*push, ny*push)
				if !movedA && movedB {
					shift(b, nx*push, ny*push)
				}
				if !movedB && movedA {
					shift(a, -nx*push, -ny*push)
				}
			}
		}
	}
}
