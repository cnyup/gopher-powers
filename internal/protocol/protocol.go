// Package protocol 对齐 src/network-protocol.js（协议 v2/v3 公共面）。
package protocol

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// Version 对齐 PROTOCOL_VERSION（双人 v2；六方走 v3）。
const Version = 2

var roomCodeRe = regexp.MustCompile(`^[0-9]{6}$`)

// RoomCode 对齐 roomCode()：NFKC 归一 + 去空白 + 六位数字。
func RoomCode(value string) (string, error) {
	code := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, normNFKC(value))
	if !roomCodeRe.MatchString(code) {
		return "", errors.New("请输入六位数字房间码")
	}
	return code, nil
}

// Nickname 对齐 nickname()：NFKC + trim + 1..16 可显示字符，拒控制/格式字符与 <>。
func Nickname(value string) (string, error) {
	name := strings.TrimSpace(normNFKC(value))
	if name == "" || len([]rune(name)) > 16 {
		return "", errors.New("昵称请使用 1 到 16 个可显示字符")
	}
	for _, r := range name {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) || r == '<' || r == '>' {
			return "", errors.New("昵称请使用 1 到 16 个可显示字符")
		}
	}
	return name, nil
}

// Fog 对齐快照迷雾（2bit/cell：bit1=visible, bit0=explored）。
type Fog struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
	Bits string `json:"bits"`
}

// PackFog 对齐 packFog()：visible/explored 布尔数组 → base64 位包。
func PackFog(cols, rows int, visible, explored []bool) Fog {
	n := cols * rows
	bytes := make([]byte, (n+3)/4)
	for i := 0; i < n; i++ {
		b := byte(0)
		if visible[i] {
			b |= 2
		}
		if explored[i] {
			b |= 1
		}
		bytes[i>>2] |= b << ((i % 4) * 2)
	}
	return Fog{Cols: cols, Rows: rows, Bits: base64.StdEncoding.EncodeToString(bytes)}
}

// UnpackFog 对齐 unpackFog()。
func UnpackFog(f Fog) (visible, explored []bool) {
	bytes, _ := base64.StdEncoding.DecodeString(f.Bits)
	n := f.Cols * f.Rows
	visible = make([]bool, n)
	explored = make([]bool, n)
	for i := 0; i < n; i++ {
		b := bytes[i>>2]
		visible[i] = b&(2<<((i%4)*2)) != 0
		explored[i] = b&(1<<((i%4)*2)) != 0
	}
	return visible, explored
}

// normNFKC：JS normalize('NFKC') 的 Go 近似——全角→半角与兼容分解。
// 覆盖协议面常见输入（房间码数字、昵称字符）；极端兼容序列用兼容表逐项对齐。
func normNFKC(s string) string {
	if isASCII(s) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(nfkcRune(r))
	}
	return b.String()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// nfkcRune 覆盖常用兼容映射（全角 ASCII、％‰等）。
func nfkcRune(r rune) rune {
	switch {
	case r >= 0xFF01 && r <= 0xFF5E: // ！-～ → !-~
		return r - 0xFF00 + 0x20
	case r == 0x3000: // 全角空格
		return ' '
	default:
		return r
	}
}
