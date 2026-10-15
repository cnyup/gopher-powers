package pathfind

import (
	"math"
	"reflect"
	"testing"
)

// 黄金向量：由真实 pathfinding npm 库（源项目 node_modules）导出。
// tie-break 声明：Go 实现与库在同 f 节点的弹出顺序可能不同（等价路径不同拐点），
// 断言锁定「路径代价 + 首尾 + 有效性」，不锁逐格序列——单位移动由
// moveWithAvoidance 主导，拐点差异不影响快照语义（已记录于重写文档）。

func pathCost(p []Node) float64 {
	c := 0.0
	for i := 1; i < len(p); i++ {
		dx, dy := p[i].X-p[i-1].X, p[i].Y-p[i-1].Y
		switch {
		case dx != 0 && dy != 0:
			c += sqrt2
		default:
			c += math.Abs(float64(dx)) + math.Abs(float64(dy))
		}
	}
	return c
}

// S1 空旷对角：压缩后仅首尾。
func TestGoldenStraightDiagonal(t *testing.T) {
	g := NewGrid(10, 10)
	want := nodes(0, 0, 9, 9)
	if got := find(t, 0, 0, 9, 9, g); !reflect.DeepEqual(got, want) {
		t.Fatalf("S1 = %v, want %v", got, want)
	}
}

// S2 竖墙留缺口 y=5：库路径代价 = 2√2+2+√2+2+4√2 = 7√2+8。
func TestGoldenWallGap(t *testing.T) {
	g := NewGrid(10, 10)
	for y := 0; y < 10; y++ {
		if y != 5 {
			g.SetWalkableAt(4, y, false)
		}
	}
	got := find(t, 0, 0, 9, 9, g)
	if len(got) < 3 || got[0] != (Node{0, 0}) || got[len(got)-1] != (Node{9, 9}) {
		t.Fatalf("S2 端点错误: %v", got)
	}
	// 必须穿过缺口 (4,5)。
	hitGap := false
	for _, n := range got {
		if n.X == 4 && n.Y == 5 {
			hitGap = true
		}
	}
	if !hitGap {
		t.Fatalf("S2 未穿缺口: %v", got)
	}
	if c := pathCost(got); math.Abs(c-(7*sqrt2+8)) > 1e-9 {
		t.Fatalf("S2 代价 = %v, want %v", c, 7*sqrt2+8)
	}
}

// S3 起点被墙角围死：库返回 []（空数组）。
func TestGoldenCorneredStart(t *testing.T) {
	g := NewGrid(5, 5)
	g.SetWalkableAt(1, 0, false)
	g.SetWalkableAt(0, 1, false)
	if got := FindPath(0, 0, 4, 4, g); len(got) != 0 {
		t.Fatalf("S3 = %v, want empty", got)
	}
}

// S4 全墙不可达：空。
func TestGoldenUnreachable(t *testing.T) {
	g := NewGrid(6, 6)
	for y := 0; y < 6; y++ {
		g.SetWalkableAt(3, y, false)
	}
	if got := FindPath(0, 0, 5, 5, g); len(got) != 0 {
		t.Fatalf("S4 = %v, want empty", got)
	}
}

// S5 起点即终点：单点。
func TestGoldenSamePoint(t *testing.T) {
	g := NewGrid(4, 4)
	want := nodes(2, 2)
	if got := FindPath(2, 2, 2, 2, g); !reflect.DeepEqual(got, want) {
		t.Fatalf("S5 = %v, want %v", got, want)
	}
}

// S6 复合障碍（竖墙+横墙）：代价与库一致。
func TestGoldenComplexObstacles(t *testing.T) {
	g := NewGrid(12, 8)
	for y := 1; y < 7; y++ {
		g.SetWalkableAt(5, y, false)
	}
	for x := 5; x < 10; x++ {
		g.SetWalkableAt(x, 6, false)
	}
	got := CompressPath(FindPath(0, 7, 11, 0, g))
	if got[0] != (Node{0, 7}) || got[len(got)-1] != (Node{11, 0}) {
		t.Fatalf("S6 端点错误: %v", got)
	}
	// 库路径 [[0,7],[2,5],[2,3],[4,1],[4,0],[11,0]]：段 2√2 + 2 + 2√2 + 1 + 7。
	wantCost := 4*sqrt2 + 10
	if c := pathCost(got); math.Abs(c-wantCost) > 1e-9 {
		t.Fatalf("S6 代价 = %v, want %v", c, wantCost)
	}
	// 全程不穿墙。
	for _, n := range got {
		if (n.X == 5 && n.Y >= 1 && n.Y <= 6) || (n.Y == 6 && n.X >= 5 && n.X <= 9) {
			t.Fatalf("S6 穿墙: %v", got)
		}
	}
}

// dontCrossCorners 语义独立验证。
func TestDontCrossCorners(t *testing.T) {
	g := NewGrid(3, 3)
	g.SetWalkableAt(1, 0, false)
	p := FindPath(0, 0, 2, 2, g)
	for i := 1; i < len(p); i++ {
		dx, dy := p[i].X-p[i-1].X, p[i].Y-p[i-1].Y
		if dx != 0 && dy != 0 {
			if !g.IsWalkableAt(p[i-1].X+dx, p[i-1].Y) || !g.IsWalkableAt(p[i-1].X, p[i-1].Y+dy) {
				t.Fatal("出现穿角移动")
			}
		}
	}
}

// CompressPath 边界。
func TestCompressPath(t *testing.T) {
	if got := CompressPath(nil); got != nil {
		t.Fatal("nil 应返回 nil")
	}
	single := nodes(1, 1)
	if got := CompressPath(single); !reflect.DeepEqual(got, single) {
		t.Fatal("单点应原样返回")
	}
	line := nodes(0, 0, 1, 0, 2, 0)
	lineWant := nodes(0, 0, 2, 0)
	if got := CompressPath(line); !reflect.DeepEqual(got, lineWant) {
		t.Fatalf("直线压缩 = %v, want %v", got, lineWant)
	}
	zig := nodes(0, 0, 1, 1, 2, 2, 3, 2)
	want := nodes(0, 0, 2, 2, 3, 2)
	if got := CompressPath(zig); !reflect.DeepEqual(got, want) {
		t.Fatalf("折线压缩 = %v, want %v", got, want)
	}
}

// 堆序自检：FIFO 推入相同 f 序列，pop 应符合 heap 库语义（根替换 + 小子下移）。
func TestHeapOrderBasic(t *testing.T) {
	fvals := []float64{5, 3, 8, 3, 1}
	h := &pathHeap{f: func(i int) float64 { return fvals[i] }, pos: map[int]int{}}
	for i, _ := range fvals {
		h.push(i)
	}
	// pop 顺序应按 f 升序：1(idx4), 3(idx1 或 3 依堆形状), ...
	first := h.pop()
	if fvals[first] != 1 {
		t.Fatalf("首次 pop f=%v, want 1", fvals[first])
	}
	second := h.pop()
	if fvals[second] != 3 {
		t.Fatalf("第二次 pop f=%v, want 3", fvals[second])
	}
}

func find(t *testing.T, sx, sy, ex, ey int, g *Grid) []Node {
	t.Helper()
	return CompressPath(FindPath(sx, sy, ex, ey, g))
}

func nodes(xs ...int) []Node {
	var out []Node
	for i := 0; i+1 < len(xs); i += 2 {
		out = append(out, Node{xs[i], xs[i+1]})
	}
	return out
}
