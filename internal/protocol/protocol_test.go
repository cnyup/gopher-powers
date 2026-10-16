package protocol

import (
	"testing"
)

// 黄金向量：与 JS node 实测逐字对齐（src/network-protocol.js）。
func TestNickname(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"玩家一", "玩家一", false},
		{"  alice  ", "alice", false},
		{"", "", true},
		{"０１２３４５６７８９０１２３４５６７", "", true}, // 17 全角数字 → NFKC 后 17 字符超限
		{"ａｂｃ", "abc", false}, // 全角→半角
		{"a<b", "", true},
		{"a\tb", "", true},
	}
	for _, c := range cases {
		got, err := Nickname(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Nickname(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Nickname(%q) = (%q,%v), want (%q,nil)", c.in, got, err, c.want)
		}
	}
}

func TestRoomCode(t *testing.T) {
	if got, err := RoomCode(" 123456 "); err != nil || got != "123456" {
		t.Errorf("RoomCode 去空白失败: %q %v", got, err)
	}
	if _, err := RoomCode("12345"); err == nil {
		t.Error("5 位应报错")
	}
	if _, err := RoomCode("12345a"); err == nil {
		t.Error("非数字应报错")
	}
}

func TestPackUnpackFog(t *testing.T) {
	// JS 实测：visible=[T,F,T,F,...] explored=[F,T,F,T,...] 的 4x4。
	cols, rows := 4, 4
	n := cols * rows
	visible := make([]bool, n)
	explored := make([]bool, n)
	for i := 0; i < n; i++ {
		visible[i] = i%2 == 0
		explored[i] = i%2 == 1
	}
	f := PackFog(cols, rows, visible, explored)
	if f.Cols != cols || f.Rows != rows {
		t.Fatalf("尺寸错: %+v", f)
	}
	v2, e2 := UnpackFog(f)
	for i := 0; i < n; i++ {
		if v2[i] != visible[i] || e2[i] != explored[i] {
			t.Fatalf("roundtrip 失败 @%d: v=%v e=%v (want %v %v)", i, v2[i], e2[i], visible[i], explored[i])
		}
	}
	// 位模式锚点：cell0 visible=1 explored=0 → 低 2 位 = 0b10 = 2。
	if f.Bits == "" {
		t.Fatal("bits 为空")
	}
}

// 黄金向量：JS packFog 实测 bits=ZmZmZg==（visible 偶/explored 奇，4x4）。
func TestFogGoldenBits(t *testing.T) {
	cols, rows := 4, 4
	n := cols * rows
	visible := make([]bool, n)
	explored := make([]bool, n)
	for i := 0; i < n; i++ {
		visible[i] = i%2 == 0
		explored[i] = i%2 == 1
	}
	f := PackFog(cols, rows, visible, explored)
	if f.Bits != "ZmZmZg==" {
		t.Fatalf("bits = %q, want ZmZmZg==", f.Bits)
	}
}
