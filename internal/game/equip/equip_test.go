package equip

import "testing"

// 对齐 equipment.js equipmentProfile 主要分支。
func TestProfileOf(t *testing.T) {
	// 现实原型：china tank。
	p := ProfileOf("china", "tank", false)
	if p == nil || p.Name != "99A 主战坦克" || p.Category != "现实原型" || !p.HasSource {
		t.Fatalf("china/tank: %+v", p)
	}
	// 覆写条目：china destroyer。
	p = ProfileOf("china", "destroyer", false)
	if p == nil || p.Name != "052D 导弹驱逐舰" || p.Source != "https://www.mod.gov.cn/djzx/4809824.html" {
		t.Fatalf("china/destroyer: %+v", p)
	}
	// 未来图机器人。
	p = ProfileOf("china", "rifle", true)
	if p == nil || p.Name != "月卫战斗机器人" || p.Category != "原创／机器人" {
		t.Fatalf("future rifle: %+v", p)
	}
	// 非未来图 rifle = 通用战术单位。
	p = ProfileOf("china", "rifle", false)
	if p == nil || p.Name != "突击兵" || p.Category != "通用战术单位" {
		t.Fatalf("rifle: %+v", p)
	}
	// 未来图战机 → 科幻名。
	p = ProfileOf("china", "fighter", true)
	if p == nil || p.Name != "云隼制空穿梭机" {
		t.Fatalf("future fighter: %+v", p)
	}
	// elite 用阵营特色名。
	p = ProfileOf("nato", "elite", false)
	if p == nil || p.Name != "堡垒重坦" || p.Category != "原创／概念装备" {
		t.Fatalf("nato/elite: %+v", p)
	}
	// 未知类型 nil。
	if ProfileOf("china", "nonexistent", false) != nil {
		t.Fatal("未知类型应返回 nil")
	}
}

// 对齐 equipment.js equipmentModel 映射表。
func TestModelOf(t *testing.T) {
	cases := []struct {
		faction, typ string
		future       bool
		want         string
	}{
		{"china", "tank", false, "tank_china"},
		{"nato", "tank", false, "tank_nato"},
		{"china", "rocket", false, "rocket_china"},
		{"china", "fighter", false, "fighter_china"},
		{"usa", "fighter", false, "fighter"},
		{"china", "navalFighter", false, "fighter"},
		{"any", "navalStrike", false, "strike"},
		{"any", "refinery", false, "logistics_depot"},
		{"any", "refinery", true, "future_refinery"}, // 建筑不在 UNITS，future 走 future_ 前缀
		{"china", "destroyer", false, "destroyer_china"},
		{"russia", "destroyer", false, "destroyer"},
		{"any", "ewPlane", false, "ewPlane"},
		{"any", "ewPlane", true, "space_ewPlane"},
		{"any", "fighter", true, "space_fighter"},
		{"any", "rifle", true, "robot_rifle"},
		{"nato", "elite", false, "elite_nato"},
		{"any", "rifle", false, "rifle"},
		{"any", "supply", false, "supply"},
	}
	for _, c := range cases {
		if got := ModelOf(c.faction, c.typ, c.future); got != c.want {
			t.Errorf("ModelOf(%s,%s,%v) = %s, want %s", c.faction, c.typ, c.future, got, c.want)
		}
	}
}
