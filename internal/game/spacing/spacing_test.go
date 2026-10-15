package spacing

import (
	"testing"
)

// 测试替身：轻量单位。
type dummy struct {
	id          int
	typ         string
	x, y        float64
	hp          float64
	stunUntil   float64
}

func (d *dummy) ID() int          { return d.id }
func (d *dummy) Type() string     { return d.typ }
func (d *dummy) X() float64       { return d.x }
func (d *dummy) Y() float64       { return d.y }
func (d *dummy) HP() float64      { return d.hp }
func (d *dummy) StunUntil() float64 { return d.stunUntil }
func (d *dummy) SetPos(x, y float64) { d.x, d.y = x, y }

// 对齐 unit-spacing.js:4-7 层级判定。
func TestUnitLayer(t *testing.T) {
	cases := map[string]string{
		"tank": "ground", "rifle": "ground", "harvester": "ground",
		"fighter": "jet", "bomber": "jet", "ewPlane": "jet",
		"drone": "drone", "ghost": "drone",
		"submarine": "submerged",
		"patrol": "naval", "carrier": "naval", "containerShip": "naval",
	}
	for typ, want := range cases {
		if got := UnitLayer(typ); got != want {
			t.Errorf("UnitLayer(%s) = %s, want %s", typ, got, want)
		}
	}
}

// 对齐 unit-spacing.js:9-14 半径表。
func TestUnitRadius(t *testing.T) {
	cases := map[string]float64{
		"rifle": 9, "engineer": 9,
		"fighter": 40, "airlift": 80, "bomber": 76, "freightPlane": 68,
		"drone": 20, "ghost": 20,
		"carrier": 92, "destroyer": 65, "submarine": 46, "frigate": 48, "patrol": 36, "landing": 74, "containerShip": 88,
		"harvester": 29, "tank": 30, "apc": 30,
	}
	for typ, want := range cases {
		if got := UnitRadius(typ); got != want {
			t.Errorf("UnitRadius(%s) = %v, want %v", typ, got, want)
		}
	}
}

// 对齐 UnitSpatialIndex 3×3 邻域。
func TestSpatialIndex(t *testing.T) {
	units := []Entity{
		&dummy{id: 1, typ: "tank", x: 100, y: 100, hp: 10},
		&dummy{id: 2, typ: "tank", x: 150, y: 120, hp: 10},   // 同格
		&dummy{id: 3, typ: "tank", x: 2300, y: 100, hp: 10},  // 隔 10 格，不应出现
		&dummy{id: 4, typ: "tank", x: 300, y: 100, hp: 0},    // 死亡跳过
		&dummy{id: 5, typ: "fighter", x: 120, y: 110, hp: 10}, // 不同层
	}
	idx := NewSpatialIndex(units)
	near := idx.Nearby(units[0])
	if len(near) != 1 || near[0].ID() != 2 {
		t.Fatalf("nearby = %v（期望仅 id=2）", ids(near))
	}
}

func ids(es []Entity) []int {
	out := make([]int, len(es))
	for i, e := range es {
		out[i] = e.ID()
	}
	return out
}

// 对齐 separateUnits：重叠两单位被推开且不超过 push 上限。
func TestSeparateUnits(t *testing.T) {
	a := &dummy{id: 1, typ: "tank", x: 100, y: 100, hp: 10}
	b := &dummy{id: 2, typ: "tank", x: 115, y: 100, hp: 10} // 距离 15 < 30+30+2
	SeparateUnits([]Mobile{a, b}, 0.05, func(Mobile, Point) bool { return true }, 0)
	dx := b.x - a.x
	dy := b.y - a.y
	if !(dx*dx+dy*dy >= 0.0001) {
		t.Fatal("分离后仍在同一点")
	}
	if a.x == 100 && a.y == 100 && b.x == 115 && b.y == 100 {
		t.Fatal("没有任何单位被移动")
	}
}

// canMove 拒绝时回退推另一侧。
func TestSeparateUnitsBlocked(t *testing.T) {
	a := &dummy{id: 1, typ: "tank", x: 100, y: 100, hp: 10}
	b := &dummy{id: 2, typ: "tank", x: 115, y: 100, hp: 10}
	SeparateUnits([]Mobile{a, b}, 0.05, func(m Mobile, p Point) bool { return m.ID() != 1 }, 0)
	if a.x != 100 || a.y != 100 {
		t.Fatal("被阻挡单位不应移动")
	}
	if b.x == 115 && b.y == 100 {
		t.Fatal("另一侧应被推两次")
	}
}

// 阵型：对齐源 unit-spacing.test.js 断言——均值≈0 + 任意两偏移距离 ≥ 半径和+10。
func TestFormationOffsets(t *testing.T) {
	for _, n := range []int{1, 2, 5, 9, 16} {
		units := make([]Entity, n)
		for i := 0; i < n; i++ {
			typ := "tank"
			if i%2 == 1 {
				typ = "rifle"
			}
			units[i] = &dummy{id: i, typ: typ, x: 400, y: 600, hp: 10}
		}
		offs := FormationOffsets(units, Point{X: 800, Y: 600})
		if len(offs) != n {
			t.Fatalf("n=%d offsets = %d", n, len(offs))
		}
		var sx, sy float64
		for _, o := range offs {
			sx += o.X
			sy += o.Y
		}
		if sx > 1e-8 || sx < -1e-8 || sy > 1e-8 || sy < -1e-8 {
			t.Fatalf("n=%d 均值 (%v,%v) 应≈0", n, sx, sy)
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := offs[i].X - offs[j].X
				dy := offs[i].Y - offs[j].Y
				need := UnitRadius(units[i].Type()) + UnitRadius(units[j].Type()) + 10
				if dx*dx+dy*dy < need*need {
					t.Fatalf("n=%d 偏移 %d-%d 距离 %v < %v", n, i, j, dist(dx, dy), need)
				}
			}
		}
	}
}
