package game

import "testing"

// Wave 1f-4：AI 决策循环冒烟——推演 60 秒，AI（side1）应产生建造/生产活动。
func TestAIRunsAndBuilds(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{Difficulty: "veteran"})
	// 静音事件，仅驱动模拟。
	for i := 0; i < 20*60 && g.Running; i++ {
		g.Update(0.05)
	}
	b1 := g.OwnedBuildings(1, "")
	if len(b1) <= 5 { // 开局 5 座，60 秒内 AI 应至少动工
		t.Fatalf("AI 60 秒后建筑数 = %d（应 >5）credits=%v", len(b1), g.Players[1].Credits)
	}
	u1 := g.OwnedUnits(1, "")
	if len(u1) <= 4 {
		t.Fatalf("AI 60 秒后单位数 = %d（应 >4）", len(u1))
	}
}

// AI 配兵反制：面对空中威胁出 aa/防空。
func TestAICounterAir(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	choices := []string{"rifle", "tank", "aa", "apc"}
	// 无威胁 → 常规轮换。
	first := g.aiCounterUnit(1, choices, "ground")
	if first == "aa" {
		t.Fatal("无空中威胁不应优先 aa")
	}
	// 放 4 架敌机（side0）在 AI 视野内。
	g.Players[0].SatelliteUntil = g.Time + 100 // 让 side1 也能看到？不——satellite 是 side0 的。
	g.UpdateFog()
	for i := 0; i < 4; i++ {
		g.AddUnit(0, "fighter", 1600, 600+float64(i)*40, false)
	}
	// side1 视野覆盖敌机位置。
	for _, u := range g.ActiveUnits(1, "") {
		u.X, u.Y = 1600, 700
	}
	g.Players[1].SatelliteUntil = g.Time + 100
	g.UpdateFog()
	g.AddBuilding(1, "radar", 1600, 900)  // aa 需要 radar（源码 requires 过滤）
	g.AddBuilding(1, "armory", 1500, 1000) // aa 由 armory 生产（源码 producer 过滤）
	got := g.aiCounterUnit(1, choices, "ground")
	if got != "aa" {
		t.Fatalf("4 架敌机威胁下应出 aa，得 %q", got)
	}
}

// queueAIUnit：满队列时拒绝。
func TestQueueAIUnit(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	b := g.OwnedBuildings(1, "barracks")[0]
	for i := 0; i < 5; i++ {
		b.Queue = append(b.Queue, "rifle")
	}
	b.Active = &Active{Type: "rifle"}
	if g.queueAIUnit("rifle", 1) {
		t.Fatal("队列+active ≥ 2 时应拒绝（阈值 2，见源码 facilities.some(...<2)）")
	}
	_ = g.queueAIUnit("engineer", 1) // engineer 走 QueueUnit 直接路径
}

// placeAIBuilding：dock 特例（水图）。
func TestPlaceAIBuildingDock(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{MapID: "ocean"})
	before := g.Players[1].Credits
	g.placeAIBuilding("dock", 1)
	docks := g.OwnedBuildings(1, "dock")
	if len(docks) == 0 && g.Players[1].Credits <= before {
		t.Fatal("dock 部署失败时应退款 60%")
	}
}

// 玩家指令面：StartBuild/QueueUnit/PlaceBuilding 闭环。
func TestPlayerBuildLoop(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	if !g.StartBuild(0, "power") {
		t.Fatal("应能开始建造 power")
	}
	if g.StartBuild(0, "radar") {
		t.Fatal("已有队列应拒绝二次建造")
	}
	p := g.Players[0]
	p.Credits += 10000
	// 推进到完工。
	for i := 0; i < 20*20 && p.BuildQueue != nil; i++ {
		g.Update(0.05)
	}
	if p.BuildQueue != nil {
		t.Fatalf("20 秒后应完工（time=14s），仍 %v", p.BuildQueue)
	}
	if !g.QueueUnit(0, "rifle") {
		t.Fatal("兵营应可训练")
	}
	barracks := g.OwnedBuildings(0, "barracks")[0]
	if len(barracks.Queue) != 1 {
		t.Fatal("队列应有 1 项")
	}
	// 取消退款。
	if !g.CancelUnitProduction(0, barracks.ID) {
		t.Fatal("取消生产应成功")
	}
	if len(barracks.Queue) != 0 {
		t.Fatal("取消后队列应空")
	}
}
