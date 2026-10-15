package pathfind

import "testing"

// 黄金向量：tick473 对拍现场的网格与起终点（JS pathfinding 库实测输出）。
// raw:    [[42,18][42,19][42,20][42,21][41,21][40,21][40,20][40,19][40,18]]
// 压缩后: [[42,18][42,21][40,21][40,18]]
func TestGoldenTick473(t *testing.T) {
	grid := NewGrid(56, 36)
	blocked := [][2]int{{40, 15}, {41, 15}, {46, 15}, {40, 16}, {41, 16}, {41, 17}, {43, 17}, {44, 17}, {45, 17}, {41, 18}, {43, 18}, {44, 18}, {45, 18}, {41, 19}, {44, 19}, {41, 20}}
	for _, b := range blocked {
		grid.SetWalkableAt(b[0], b[1], false)
	}
	raw := FindPath(42, 18, 40, 18, grid)
	wantRaw := []Node{{42, 18}, {42, 19}, {42, 20}, {42, 21}, {41, 21}, {40, 21}, {40, 20}, {40, 19}, {40, 18}}
	if len(raw) != len(wantRaw) {
		t.Fatalf("raw len = %d, want %d:\n got %v\nwant %v", len(raw), len(wantRaw), raw, wantRaw)
	}
	for i, n := range raw {
		if n != wantRaw[i] {
			t.Fatalf("raw[%d] = %v, want %v\n got %v\nwant %v", i, n, wantRaw[i], raw, wantRaw)
		}
	}
}

func TestGoldenTick473Trace(t *testing.T) {
	grid := NewGrid(56, 36)
	blocked := [][2]int{{40, 15}, {41, 15}, {46, 15}, {40, 16}, {41, 16}, {41, 17}, {43, 17}, {44, 17}, {45, 17}, {41, 18}, {43, 18}, {44, 18}, {45, 18}, {41, 19}, {44, 19}, {41, 20}}
	for _, b := range blocked {
		grid.SetWalkableAt(b[0], b[1], false)
	}
	SetPathTrace(func(ev string, x, y int, f, g float64) { t.Logf("%s (%d,%d) f=%.3f g=%.3f", ev, x, y, f, g) })
	defer SetPathTrace(nil)
	FindPath(42, 18, 40, 18, grid)
}
