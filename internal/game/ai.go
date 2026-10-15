package game

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// ai.go 对齐 game.js L1227-1348：updateAI/queueAIUnit/placeAIBuilding/launchAIWave。
// 依赖 tactical-ai.js 的 aiCounterUnit/retreatAIUnits（本包内实现）。

// knownThreats 对齐 tactical-ai.js:3-6。
type threatInfo struct {
	Air, Drones, Armor, Subs, Ships, Forts int
}

func (g *Game) knownThreats(side int) threatInfo {
	var t threatInfo
	for _, u := range g.EnemyUnits(side, "") {
		if !g.CanSeeEntity(side, u) {
			continue
		}
		if gpdata.HasTag(u.Type, "air") {
			t.Air++
		}
		if gpdata.HasTag(u.Type, "drone") {
			t.Drones++
		}
		if gpdata.HasTag(u.Type, "armor") {
			t.Armor++
		}
		if gpdata.HasTag(u.Type, "submerged") {
			t.Subs++
		}
		if gpdata.HasTag(u.Type, "ship") {
			t.Ships++
		}
	}
	for _, b := range g.EnemyBuildings(side, "turret") {
		if g.CanSeeEntity(side, b) {
			t.Forts++
		}
	}
	return t
}

// aiCounterUnit 对齐 tactical-ai.js:8-21。
func (g *Game) aiCounterUnit(side int, choices []string, domain string) string {
	threats := g.knownThreats(side)
	counts := map[string]int{}
	for _, u := range g.OwnedUnits(side, "") {
		counts[u.Type]++
	}
	for _, b := range g.OwnedBuildings(side, "") {
		for _, typ := range b.Queue {
			counts[typ]++
		}
		if b.Active != nil {
			counts[b.Active.Type]++
		}
	}
	ground := 0
	for _, u := range g.ActiveUnits(side, "") {
		if gpdata.UNITS[u.Type].Damage > 0 && !gpdata.HasTag(u.Type, "air") && !gpdata.HasTag(u.Type, "ship") {
			ground++
		}
	}
	type demand struct {
		typ     string
		desired int
		weight  int
	}
	var demands []demand
	ceil := func(v float64) int { return int(v) + boolInt(frac(v) > 0) }
	switch domain {
	case "air":
		demands = []demand{{"fighter", ceil(float64(threats.Air) / 2), 4}, {"navalFighter", ceil(float64(threats.Air) / 2), 4}}
	case "naval":
		demands = []demand{{"frigate", ceil(float64(threats.Subs+threats.Air) / 2), 4}, {"submarine", ceil(float64(threats.Ships) / 3), 2}}
	default:
		supply := 0
		if ground >= 4 {
			supply = 1
		}
		demands = []demand{
			{"aa", ceil(float64(threats.Air) / 2), 3},
			{"laser", ceil(float64(threats.Drones) / 3), 4},
			{"loiterer", ceil(float64(threats.Armor) / 3), 2},
			{"rocket", ceil(float64(threats.Armor+threats.Forts*2) / 4), 2},
			{"supply", supply, 1},
		}
	}
	var available []demand
	inChoices := func(t string) bool {
		for _, c := range choices {
			if c == t {
				return true
			}
		}
		return false
	}
	for _, dm := range demands {
		d := gpdata.UNITS[dm.typ]
		if !inChoices(dm.typ) || dm.desired <= counts[dm.typ] {
			continue
		}
		if !gpdata.SupportsMap(d.Map, g.MapID) {
			continue
		}
		if d.Faction != "" && d.Faction != g.Players[side].Faction {
			continue
		}
		if d.Requires != "" && !g.HasBuilding(side, d.Requires) {
			continue
		}
		if !g.HasBuilding(side, d.Producer) {
			continue
		}
		available = append(available, dm)
	}
	if len(available) == 0 {
		return ""
	}
	best := available[0]
	bestScore := (available[0].desired - counts[available[0].typ]) * available[0].weight
	for _, dm := range available[1:] {
		score := (dm.desired - counts[dm.typ]) * dm.weight
		if score > bestScore {
			best, bestScore = dm, score
		}
	}
	return best.typ
}

func frac(v float64) float64 { return v - float64(int(v)) }
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// retreatAIUnits 对齐 tactical-ai.js:23-36。
func (g *Game) retreatAIUnits(side int) {
	if !g.HasPower(side) || g.Players[side].Credits < 60 {
		return
	}
	for _, u := range g.ActiveUnits(side, "") {
		d := gpdata.UNITS[u.Type]
		if d.Damage == 0 || u.TemporaryUntil != 0 || len(u.Passengers) > 0 {
			continue
		}
		if u.Order != nil && (u.Order.Type == "rearm" || u.Order.Type == "board" || u.Order.Type == "restock") {
			continue
		}
		depleted := d.Ammo > 0 && u.Ammo != nil && *u.Ammo <= float64(int(d.Ammo*.25))
		if u.HP/u.MaxHP >= .32 && !depleted {
			continue
		}
		station := "factory"
		switch {
		case gpdata.HasTag(u.Type, "air"):
			station = "airfield"
		case gpdata.HasTag(u.Type, "ship"):
			station = "dock"
		case gpdata.HasTag(u.Type, "infantry"):
			station = "barracks"
		}
		if !g.HasBuilding(side, station) && !(station == "factory" && g.HasBuilding(side, "armory")) {
			continue
		}
		g.RequestResupplyFull(u, nil)
	}
}

// RequestResupply 桩（modern-combat 1d 完整化）。
func (g *Game) RequestResupply(u *Unit, home *Building) {
	u.ResumeOrder = u.Order
	u.Order = &Order{Type: "rearm"}
	u.Path = nil
}

// queueAIUnit 对齐 game.js L1340-1345。
func (g *Game) queueAIUnit(typ string, side int) bool {
	if typ == "engineer" {
		return g.QueueUnit(side, typ)
	}
	ok := false
	for _, b := range g.OwnedBuildings(side, gpdata.UNITS[typ].Producer) {
		n := len(b.Queue)
		if b.Active != nil {
			n++
		}
		if n < 2 {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	return g.QueueUnit(side, typ)
}

// aiPlan 对齐 updateAI 的建筑计划表。
var aiBuildPlan = []string{"power", "radar", "power", "airfield", "turret", "lab", "super", "turret", "power", "factory", "refinery"}

// aiSlots 对齐 placeAIBuilding 的部署槽位表。
var aiSlots = [][2]float64{{350, 965}, {510, 550}, {600, 880}, {510, 1020}, {670, 720}, {245, 475}, {660, 1000}, {530, 400}, {710, 570}, {430, 1150}, {750, 1080}}

// UpdateAI 对齐 game.js L1227-1339（建筑/兵种/海空/工程/技能决策）。
func (g *Game) updateAI(side int) {
	if g.IsControlLocked(side) {
		return
	}
	p := g.Players[side]
	if len(g.OwnedBuildings(side, "")) == 0 {
		return
	}
	if p.Credits > 1800 {
		g.ActivateSatellite(side)
	}
	g.retreatAIUnits(side)
	airfield := g.OwnedBuildings(side, "airfield")
	var airfieldB *Building
	if len(airfield) > 0 {
		airfieldB = airfield[0]
	}
	dock := g.OwnedBuildings(side, "dock")
	var dockB *Building
	if len(dock) > 0 {
		dockB = dock[0]
	}
	prioritizingFirstStrike := airfieldB != nil && g.Time >= g.Difficulty.WaveStart && !p.AIFirstStrikeProduced
	// 建筑决策。
	if g.Map.Water != nil && dockB == nil && p.BuildQueue == nil && p.Credits > 600 && g.CanBuild(side, "dock") {
		g.StartBuild(side, "dock")
	} else if !g.HasBuilding(side, "armory") && p.BuildQueue == nil && !prioritizingFirstStrike && g.Time >= 18 && p.Credits > 600 && g.CanBuild(side, "armory") {
		g.StartBuild(side, "armory")
	} else if g.HasBuilding(side, "hq") && p.BuildQueue == nil && !prioritizingFirstStrike && p.AIPlanIndex < len(aiBuildPlan) {
		typ := aiBuildPlan[p.AIPlanIndex]
		if g.CanBuild(side, typ) && p.Credits > 300 {
			g.StartBuild(side, typ)
			p.AIPlanIndex++
		}
	}
	nextPlan := ""
	if p.AIPlanIndex < len(aiBuildPlan) {
		nextPlan = aiBuildPlan[p.AIPlanIndex]
	}
	savingForTech := p.BuildQueue == nil && (nextPlan == "lab" || nextPlan == "super") && g.CanBuild(side, nextPlan)
	fundingTech := prioritizingFirstStrike || savingForTech || (p.BuildQueue != nil && (p.BuildQueue.Type == "lab" || p.BuildQueue.Type == "super"))
	if fundingTech {
		for _, b := range g.OwnedBuildings(side, "") {
			if !isProducer(b.Type) {
				continue
			}
			if prioritizingFirstStrike && b.Type == "airfield" {
				continue
			}
			kept := b.Queue[:0]
			for _, t := range b.Queue {
				if t == "harvester" {
					kept = append(kept, t)
				}
			}
			b.Queue = kept
			if b.Active != nil && b.Active.Type != "harvester" {
				g.CancelUnitProduction(side, b.ID)
			}
		}
	}
	// 采矿车保有。
	harvesters := len(g.OwnedUnits(side, "harvester"))
	pendingHarvesters := 0
	for _, b := range g.OwnedBuildings(side, "factory") {
		for _, t := range b.Queue {
			if t == "harvester" {
				pendingHarvesters++
			}
		}
		if b.Active != nil && b.Active.Type == "harvester" {
			pendingHarvesters++
		}
	}
	if g.EconomyMode == "mining" && harvesters+pendingHarvesters < 2 && p.Credits > 450 {
		g.queueAIUnit("harvester", side)
	}
	// 间谍/资源情报调度（对齐 game.js:1234 updateIntelligenceAI）。
	g.updateIntelligenceAI(side)
	// 工程师占领。
	type siteRef struct {
		id    string
		kind  string // oil/beacon/depot
		owner int
		x, y  float64
	}
	var knownSites []siteRef
	addSites := func(list []*Site, kind string) {
		for _, s := range list {
			if s.Owner != side && g.HasExploredFor(side, s.X, s.Y) {
				knownSites = append(knownSites, siteRef{s.ID, kind, s.Owner, s.X, s.Y})
			}
		}
	}
	addSites(g.Oil, "oil")
	addSites(g.Beacons, "beacon")
	for _, s := range g.ResourceSites {
		if s.Type == "depot" && s.Owner != side && g.HasExploredFor(side, s.X, s.Y) {
			knownSites = append(knownSites, siteRef{s.ID, "depot", s.Owner, s.X, s.Y})
		}
	}
	if !fundingTech && len(knownSites) > 0 && g.Time >= p.AIEngineerRetryAt && len(g.OwnedUnits(side, "engineer")) == 0 && p.Credits > 380 {
		if g.queueAIUnit("engineer", side) {
			p.AIEngineerRetryAt = g.Time + 54
		}
	}
	for _, e := range g.OwnedUnits(side, "engineer") {
		if e.Order != nil && (e.Order.Type == "capture" || e.Order.Type == "defuse" || e.Order.Type == "collect") {
			continue
		}
		// 最近已探点。
		best := -1
		bestD := 1e18
		for i, s := range knownSites {
			if d := Distance(e.X, e.Y, s.x, s.y); d < bestD {
				bestD, best = d, i
			}
		}
		if best >= 0 {
			s := knownSites[best]
			e.Order = &Order{Type: "capture", X: s.x, Y: s.y}
			// targetId 为字符串 id：以负数编码存入（capture 时按距离重匹配）。
			e.Order.TargetID = 0
		}
	}
	// 侦察/幽灵。
	if !fundingTech && g.Time >= p.AIScoutRetryAt && len(g.OwnedUnits(side, "scout")) == 0 && p.Credits > 280 {
		if g.queueAIUnit("scout", side) {
			p.AIScoutRetryAt = g.Time + 60
		}
	}
	if !fundingTech && p.Faction == "middleeast" && g.HasBuilding(side, "radar") && g.Time >= p.AIGhostRetryAt && len(g.OwnedUnits(side, "ghost")) == 0 && p.Credits > 550 {
		if g.queueAIUnit("ghost", side) {
			p.AIGhostRetryAt = g.Time + 100
		}
	}
	// 维修。
	for _, b := range g.OwnedBuildings(side, "hq") {
		if b.HP < b.MaxHP*.65 && !b.Repairing && p.Credits > 450 {
			g.ToggleRepair(side, b.ID)
		}
	}
	for _, b := range g.OwnedBuildings(side, "") {
		if b.Type != "hq" && b.HP < b.MaxHP*.55 && !b.Repairing && p.Credits > 500 {
			g.ToggleRepair(side, b.ID)
		}
	}
	// 陆军配兵。
	combat := 0
	for _, u := range g.OwnedUnits(side, "") {
		if gpdata.UNITS[u.Type].Damage > 0 && !gpdata.HasTag(u.Type, "ship") && !gpdata.HasTag(u.Type, "jet") {
			combat++
		}
	}
	if !fundingTech && combat < g.Difficulty.CombatLimit {
		sequence := []string{"rifle", "tank", "drone", "apc", "supply", "aa", "loiterer", "jammer", "laser", "rocket", "elite"}
		if g.Map.Future {
			sequence = append(sequence, "railgun", "relay")
		}
		var avail []string
		for _, typ := range sequence {
			req := gpdata.UNITS[typ].Requires
			if req == "" || g.HasBuilding(side, req) {
				avail = append(avail, typ)
			}
		}
		typ := g.aiCounterUnit(side, avail, "ground")
		if typ == "" && len(avail) > 0 {
			typ = avail[p.AIUnitCount%len(avail)]
		}
		if typ != "" && p.Credits > g.unitCost(side, typ)*0.4 {
			if g.queueAIUnit(typ, side) {
				p.AIUnitCount++
			}
		}
	}
	// 空军。
	if airfieldB != nil && (!fundingTech || prioritizingFirstStrike) && g.Time >= g.Difficulty.WaveStart && p.Credits > 350 {
		airborne := 0
		pending := 0
		for _, u := range g.OwnedUnits(side, "") {
			if gpdata.HasTag(u.Type, "jet") && !freightTypes[u.Type] {
				airborne++
			}
		}
		pending = len(airfieldB.Queue)
		if airfieldB.Active != nil {
			pending++
		}
		if airborne+pending < 4 {
			typ := g.aiCounterUnit(side, []string{"fighter"}, "air")
			if typ == "" {
				switch {
				case g.HasBuilding(side, "lab") && p.AIAirUnitCount%6 == 5:
					typ = "ewPlane"
				case g.HasBuilding(side, "lab") && p.AIAirUnitCount%4 == 2:
					typ = "bomber"
				case g.HasBuilding(side, "radar") && p.AIAirUnitCount%4 == 3 && len(g.OwnedUnits(side, "airlift")) == 0:
					typ = "airlift"
				case g.Map.Future && g.HasBuilding(side, "lab") && p.AIAirUnitCount%3 == 2:
					typ = "aegis"
				case p.AIAirUnitCount%2 == 0:
					typ = "strike"
				default:
					typ = "fighter"
				}
			}
			if g.queueAIUnit(typ, side) {
				p.AIAirUnitCount++
			}
		}
	}
	// 海军。
	if dockB != nil && !fundingTech && g.Time >= 20 && p.Credits > 280 {
		afloat := 0
		for _, u := range g.OwnedUnits(side, "") {
			if gpdata.HasTag(u.Type, "ship") && !freightTypes[u.Type] {
				afloat++
			}
		}
		pending := len(dockB.Queue)
		if dockB.Active != nil {
			pending++
		}
		cap := 4
		if g.MapID == "ocean" || g.MapID == "archipelago" {
			cap = 9
		}
		if afloat+pending < cap {
			var fleet []string
			if g.MapID == "ocean" || g.MapID == "archipelago" {
				fleet = []string{"patrol", "frigate", "destroyer", "submarine", "carrier", "landing"}
			} else {
				fleet = []string{"patrol", "frigate"}
			}
			var avail []string
			for _, typ := range fleet {
				req := gpdata.UNITS[typ].Requires
				if req == "" || g.HasBuilding(side, req) {
					avail = append(avail, typ)
				}
			}
			typ := g.aiCounterUnit(side, avail, "naval")
			if typ == "" && len(avail) > 0 {
				typ = avail[p.AINavyUnitCount%len(avail)]
			}
			if typ != "" && g.queueAIUnit(typ, side) {
				p.AINavyUnitCount++
			}
		}
	}
	// 技能释放。
	if g.HasBuilding(side, "super") && g.HasPower(side) && p.AbilityCharge >= 100 && p.Credits >= 650 {
		if p.Faction == "china" {
			threatened := false
			for _, b := range g.OwnedBuildings(side, "") {
				for _, u := range g.EnemyUnits(side, "") {
					if g.IsVisibleFor(side, u.X, u.Y) && Distance(b.X, b.Y, u.X, u.Y) < 430 {
						threatened = true
						break
					}
				}
				if threatened {
					break
				}
			}
			if threatened {
				g.CastAbility(0, 0, side)
			}
		} else {
			var target *Building
			for _, b := range g.EnemyBuildings(side, "") {
				if !g.IsVisibleFor(side, b.X, b.Y) {
					continue
				}
				if target == nil || b.Type == "hq" {
					target = b
					if b.Type == "hq" {
						break
					}
				}
			}
			if target != nil {
				g.CastAbility(target.X+g.Random(-65, 65), target.Y+g.Random(-65, 65), side)
			}
		}
	}
}

// IsControlLocked / QueueUnit / StartBuild / CancelUnitProduction / ToggleRepair / CastAbility / CanBuild
// 在 commands.go（本批一并落地，对齐 game.js 对应方法）。
