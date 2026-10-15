// Package pathfind 逐位对齐 npm pathfinding AStarFinder（源用法 game.js:1086,1131,1134）
// + npm heap 二叉堆序。关键语义：堆持有节点引用（f 突变实时可见），
// 故此实现堆内只存格索引、比较时实时读 f——快照式 f 会漂移 tie-break。
package pathfind

import "math"

type Grid struct {
	Cols, Rows int
	walkable   []bool
}

func NewGrid(cols, rows int) *Grid {
	g := &Grid{Cols: cols, Rows: rows, walkable: make([]bool, cols*rows)}
	for i := range g.walkable {
		g.walkable[i] = true
	}
	return g
}

func (g *Grid) IsWalkableAt(x, y int) bool {
	if x < 0 || y < 0 || x >= g.Cols || y >= g.Rows {
		return false
	}
	return g.walkable[y*g.Cols+x]
}

func (g *Grid) SetWalkableAt(x, y int, ok bool) {
	if x < 0 || y < 0 || x >= g.Cols || y >= g.Rows {
		return
	}
	g.walkable[y*g.Cols+x] = ok
}

type Node struct{ X, Y int }

const sqrt2 = math.Sqrt2

type nodeData struct {
	g, f, h float64
	hSet    bool
	opened  bool
	closed  bool
	parent  int
}

// pathHeap 复刻 heap/lib/heap.js（cmp = a.f - b.f < 0）。元素是格索引。
type pathHeap struct {
	arr  []int // 格索引
	f    func(idx int) float64
	pos  map[int]int
}

func (h *pathHeap) less(i, j int) bool { return h.f(h.arr[i]) < h.f(h.arr[j]) }

// push 对齐 heappush：append 后 _siftdown(array, 0, len-1)。
func (h *pathHeap) push(idx int) {
	h.arr = append(h.arr, idx)
	h.pos[idx] = len(h.arr) - 1
	h.siftDown(0, len(h.arr)-1)
}

// siftDown 对齐 _siftdown(array, startpos, pos)：向根方向冒。
func (h *pathHeap) siftDown(startpos, pos int) {
	newitem := h.arr[pos]
	for pos > startpos {
		parentpos := (pos - 1) >> 1
		if h.f(newitem) < h.f(h.arr[parentpos]) {
			h.arr[pos] = h.arr[parentpos]
			h.pos[h.arr[pos]] = pos
			pos = parentpos
			continue
		}
		break
	}
	h.arr[pos] = newitem
	h.pos[newitem] = pos
}

// siftUp 对齐 _siftup(array, pos)：沿小子下移，再 _siftdown 回冒。
func (h *pathHeap) siftUp(pos int) {
	startpos := pos
	endpos := len(h.arr)
	newitem := h.arr[pos]
	childpos := 2*pos + 1
	for childpos < endpos {
		right := childpos + 1
		if right < endpos && !h.less(childpos, right) {
			childpos = right
		}
		h.arr[pos] = h.arr[childpos]
		h.pos[h.arr[pos]] = pos
		pos = childpos
		childpos = 2*pos + 1
	}
	h.arr[pos] = newitem
	h.pos[newitem] = pos
	h.siftDown(startpos, pos)
}

// pop 对齐 heappop：尾元素替根 + _siftup(0)。
func (h *pathHeap) pop() int {
	last := len(h.arr) - 1
	lastelt := h.arr[last]
	h.arr = h.arr[:last]
	delete(h.pos, lastelt)
	if len(h.arr) == 0 {
		return lastelt
	}
	returnitem := h.arr[0]
	h.arr[0] = lastelt
	h.pos[lastelt] = 0
	h.siftUp(0)
	return returnitem
}

// updateItem 对齐 heap.updateItem：先 _siftdown(0,pos) 再 _siftup(pos)（双向）。
func (h *pathHeap) updateItem(idx int) {
	pos, ok := h.pos[idx]
	if !ok {
		return
	}
	h.siftDown(0, pos)
	h.siftUp(pos)
}

// pathTrace 可选观测钩子（对拍诊断用）。
var pathTrace func(ev string, x, y int, f, g float64)

// SetPathTrace 注入/移除 A* 观测钩子。
func SetPathTrace(f func(ev string, x, y int, f, g float64)) { pathTrace = f }

// FindPath 对齐 AStarFinder.findPath（OnlyWhenNoObstacles 对角）。
// 邻接序对齐 Grid.getNeighbors：↑→↓← 后 ↖↗↘↙。
func FindPath(sx, sy, ex, ey int, grid *Grid) []Node {
	w := grid.Cols
	if !grid.IsWalkableAt(sx, sy) || !grid.IsWalkableAt(ex, ey) {
		return []Node{}
	}
	nodes := make([]nodeData, grid.Cols*grid.Rows)
	for i := range nodes {
		nodes[i].parent = -1
	}
	// 堆读实时 f（节点引用语义）。
	heap := &pathHeap{f: func(idx int) float64 { return nodes[idx].f }, pos: map[int]int{}}
	startIdx := sy*w + sx
	nodes[startIdx].g = 0
	nodes[startIdx].f = 0 // 库：起点 f=0（非 g+h）
	heap.push(startIdx)
	nodes[startIdx].opened = true

	for len(heap.arr) > 0 {
		cur := heap.pop()
		if pathTrace != nil {
			pathTrace("POP", cur%w, cur/w, nodes[cur].f, nodes[cur].g)
		}
		nodes[cur].closed = true
		cx, cy := cur%w, cur/w
		if cx == ex && cy == ey {
			var rev []Node
			for n := cur; n != -1; n = nodes[n].parent {
				rev = append(rev, Node{n % w, n / w})
			}
			for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
				rev[i], rev[j] = rev[j], rev[i]
			}
			return rev
		}
		s0 := grid.IsWalkableAt(cx, cy-1) // ↑
		s1 := grid.IsWalkableAt(cx+1, cy) // →
		s2 := grid.IsWalkableAt(cx, cy+1) // ↓
		s3 := grid.IsWalkableAt(cx-1, cy) // ←
		type nb struct {
			x, y int
			diag bool
		}
		var nbs []nb
		if s0 {
			nbs = append(nbs, nb{cx, cy - 1, false})
		}
		if s1 {
			nbs = append(nbs, nb{cx + 1, cy, false})
		}
		if s2 {
			nbs = append(nbs, nb{cx, cy + 1, false})
		}
		if s3 {
			nbs = append(nbs, nb{cx - 1, cy, false})
		}
		if s0 && s3 && grid.IsWalkableAt(cx-1, cy-1) { // ↖
			nbs = append(nbs, nb{cx - 1, cy - 1, true})
		}
		if s0 && s1 && grid.IsWalkableAt(cx+1, cy-1) { // ↗
			nbs = append(nbs, nb{cx + 1, cy - 1, true})
		}
		if s1 && s2 && grid.IsWalkableAt(cx+1, cy+1) { // ↘
			nbs = append(nbs, nb{cx + 1, cy + 1, true})
		}
		if s2 && s3 && grid.IsWalkableAt(cx-1, cy+1) { // ↙
			nbs = append(nbs, nb{cx - 1, cy + 1, true})
		}
		for _, n := range nbs {
			ni := n.y*w + n.x
			if nodes[ni].closed {
				continue
			}
			step := 1.0
			if n.diag {
				step = sqrt2
			}
			ng := nodes[cur].g + step
			if !nodes[ni].opened || ng < nodes[ni].g {
				nodes[ni].g = ng
				if !nodes[ni].hSet {
					nodes[ni].h = octile(n.x, n.y, ex, ey)
					nodes[ni].hSet = true
				}
				nodes[ni].f = ng + nodes[ni].h
				nodes[ni].parent = cur
			if !nodes[ni].opened {
				heap.push(ni)
				if pathTrace != nil {
					pathTrace("PUSH", n.x, n.y, nodes[ni].f, nodes[ni].g)
				}
				nodes[ni].opened = true
				} else {
					heap.updateItem(ni)
				}
			}
		}
	}
	return []Node{}
}

// octile 对齐 pathfinding 库 Heuristic.octile 的精确求值序：
// F = √2−1; (dx < dy) ? F*dx + dy : F*dy + dx。
// 注意 1：不能代数化简为 (dx-dy)+√2*dy——浮点舍入不同（1ulp）。
// 注意 2：乘加必须拆成两条语句——arm64 上 Go 会把 f*dy+dx 融合成 FMA
// （单次舍入），与 JS 的两次舍入结果不同（第三次踩坑，机制记入注释）。
//go:noinline
func octMulAdd(f, m float64) float64 { return f * m }

func octile(x, y, tx, ty int) float64 {
	dx := math.Abs(float64(x - tx))
	dy := math.Abs(float64(y - ty))
	s := math.Sqrt2 // 拆两条语句：单表达式 math.Sqrt2-1 会被编译器常量折叠
	f := s - 1      // 到任意精度（0.…503），与 JS 运行时 IEEE754（0.…515）位级不同。
	if dx < dy {
		m := octMulAdd(f, dx)
		return m + dy
	}
	m := octMulAdd(f, dy)
	return m + dx
}

// CompressPath 对齐 PF.Util.compressPath：仅保留方向变化点（含首尾）。
func CompressPath(path []Node) []Node {
	if len(path) <= 2 {
		return path
	}
	out := []Node{path[0]}
	for i := 1; i < len(path)-1; i++ {
		dx1 := path[i].X - path[i-1].X
		dy1 := path[i].Y - path[i-1].Y
		dx2 := path[i+1].X - path[i].X
		dy2 := path[i+1].Y - path[i].Y
		if dx1 != dx2 || dy1 != dy2 {
			out = append(out, path[i])
		}
	}
	out = append(out, path[len(path)-1])
	return out
}
