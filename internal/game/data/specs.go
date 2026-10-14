package data

import "math"

// UnitCostFor 对齐 unit-specs.js:4-6：图鉴与真实生产共用阵营修正。
func UnitCostFor(faction, typ string) float64 {
	f := 1.0
	if typ == "drone" && faction == "asia" {
		f = .85
	}
	return math.Round(UNITS[typ].Cost * f)
}

// UnitHealthFor 对齐 unit-specs.js:8-15。
func UnitHealthFor(faction, typ string) float64 {
	hp := UNITS[typ].HP
	if typ == "tank" && faction == "nato" {
		hp *= 1.15
	}
	if typ == "drone" && faction == "asia" {
		hp *= .85
	}
	if typ == "aa" && faction == "china" {
		hp *= 1.15
	}
	if typ == "elite" && faction == "nato" {
		hp = 580
	}
	if typ == "elite" && faction == "asia" {
		hp = 250
	}
	return hp
}

// validBattleID 对齐 battle-identity.js:1。
func ValidBattleID(v string) bool {
	if len(v) < 12 || len(v) > 100 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// LegacyBattleID 对齐 battle-identity.js:8-16：FNV-1a 双哈希 36 进制指纹。
// JS Math.imul 为 32 位截断乘法；Go 用 uint32 溢出语义等价。
// JS charCodeAt 对 UTF-16 码元；JSON.stringify 产出的 ASCII/中文转义文本在此处
// 由调用方保证以 UTF-16 码元序列传入（runes 即 BMP 内等价）。
func LegacyBattleID(runes []rune) string {
	a := uint32(2166136261)
	b := uint32(3339675911)
	for _, c := range runes {
		a = imul32(a^uint32(c), 16777619)
		b = imul32(b^uint32(c), 2246822519)
	}
	return "legacy-" + base36(int64(len(runes))) + "-" + base36(int64(a)) + "-" + base36(int64(b))
}

func imul32(x, y uint32) uint32 { return x * y }

// base36 输出与 JS Number.prototype.toString(36) 一致（小写字母数字）。
func base36(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	pos := len(buf)
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	for n > 0 {
		pos--
		buf[pos] = digits[n%36]
		n /= 36
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
