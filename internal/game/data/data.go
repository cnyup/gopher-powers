// Package data 承载游戏静态数据表，逐字段对齐源项目 src/data.js。
// 规模锚点：5 阵营 / 8 地图 / 33 单位 / 12 建筑 / 3 胜利模式（node -e 实测）。
package data

// World 对齐 data.js:4 WORLD。
type World struct {
	Width  float64
	Height float64
	Cell   float64
	Fog    float64
}

var WORLD = World{Width: 2240, Height: 1440, Cell: 40, Fog: 80}

// CORE_BUILDINGS 对齐 data.js:5。
var CORE_BUILDINGS = []string{"hq", "refinery", "barracks", "factory", "armory", "airfield", "dock"}

// PRODUCERS 对齐 data.js:6。
var PRODUCERS = []string{"barracks", "factory", "armory", "airfield", "dock"}

// ORE_VALUES 对齐 data.js:7。
var ORE_VALUES = map[string]float64{"gold": 1, "gem": 2}

// VictoryMode 对齐 VICTORY_MODES 条目。
type VictoryMode struct {
	Name        string
	Description string
}

var VICTORY_MODES = map[string]VictoryMode{
	"quick":       {Name: "快速对战", Description: "摧毁敌方全部生产核心"},
	"annihilation": {Name: "全域歼灭", Description: "清除敌方全部建筑与单位"},
	"control":     {Name: "据点控制", Description: "占领信标积累 240 分，或摧毁敌方生产核心"},
}

// AIDifficulty 对齐 AI_DIFFICULTIES 条目。
type AIDifficulty struct {
	Name           string
	WaveStart      float64
	WaveInterval   float64
	WaveSize       int
	ThinkInterval  float64
	CombatLimit    int
	EngineerStart  float64
}

var AI_DIFFICULTIES = map[string]AIDifficulty{
	"recruit":  {Name: "新兵", WaveStart: 125, WaveInterval: 45, WaveSize: 5, ThinkInterval: 3.5, CombatLimit: 22, EngineerStart: 58},
	"standard": {Name: "标准", WaveStart: 92, WaveInterval: 36, WaveSize: 7, ThinkInterval: 2, CombatLimit: 28, EngineerStart: 38},
	"veteran":  {Name: "专家", WaveStart: 78, WaveInterval: 30, WaveSize: 9, ThinkInterval: 1.5, CombatLimit: 34, EngineerStart: 32},
}

// Faction 对齐 FACTIONS 条目。
type Faction struct {
	Name       string
	Short      string
	Color      string
	Accent     string
	Symbol     string
	Role       string
	Summary    string
	Perk       string
	Elite      string
	Ability    string
	AbilityDesc string
}

var FACTIONS = map[string]Faction{
	"china": {Name: "华夏防卫", Short: "华夏", Color: "#57d7c2", Accent: "#9bf4df", Symbol: "盾", Role: "防守反击", Summary: "防空和阵地防御占优，推进速度偏慢。", Perk: "防御建筑与防空单位生命 +15%", Elite: "陆盾防空车", Ability: "协同电子防护", AbilityDesc: "友军获得 12 秒减伤。这是电子防御的游戏化抽象，并非真实能量护盾；网络攻击为独立战术。"},
	"russia": {Name: "北境联邦", Short: "北境", Color: "#e96558", Accent: "#ffac8e", Symbol: "焰", Role: "攻坚突击", Summary: "攻城火力强，但无人机与防空能力较弱。", Perk: "对建筑伤害 +15%，防空伤害 -12%", Elite: "破城火箭车", Ability: "震荡轰击", AbilityDesc: "轰击指定区域，重创建筑并压制步兵。"},
	"nato": {Name: "西陆同盟", Short: "西陆", Color: "#86a9ec", Accent: "#c3d4ff", Symbol: "钢", Role: "重装推进", Summary: "装甲正面强势，展开速度略慢。", Perk: "坦克生命 +15%，重坦可压制无人机群", Elite: "堡垒重坦", Ability: "精确轰炸", AbilityDesc: "快速轰炸指定区域，对轻甲尤其有效。"},
	"asia": {Name: "东亚科技体", Short: "东亚", Color: "#e8bc63", Accent: "#ffe4a1", Symbol: "蜂", Role: "蜂群游击", Summary: "无人机灵活廉价，无法承受持续火力。", Perk: "无人机成本 -15%，生命 -15%", Elite: "蜂群母机", Ability: "蜂群突袭", AbilityDesc: "向指定区域投放无人机群，持续袭扰。"},
	"middleeast": {Name: "新月能源体", Short: "新月", Color: "#bd9bdb", Accent: "#e4cdf9", Symbol: "磁", Role: "电子压制", Summary: "干扰敌方高科技单位，重装正面对抗较弱。", Perk: "电子单位射程 +15%，坦克伤害 -10%", Elite: "电磁干扰车", Ability: "区域脉冲", AbilityDesc: "使指定区域的敌军载具瘫痪 8 秒。"},
}

// BuildingSpec 对齐 BUILDINGS 条目。零值表示字段缺省（JS 无此键）。
type BuildingSpec struct {
	Name     string
	Cost     float64
	Time     float64
	HP       float64
	Power    float64
	Size     float64
	Icon     string
	Desc     string
	Requires string
	Naval    bool
}

var BUILDINGS = map[string]BuildingSpec{
	"hq":       {Name: "指挥中心", Cost: 1500, Time: 45, HP: 1350, Power: 0, Size: 92, Icon: "castle", Desc: "生产核心，失去后将无法继续建造。"},
	"power":    {Name: "电力站", Cost: 350, Time: 14, HP: 560, Power: 100, Size: 66, Icon: "zap", Desc: "提供 100 电力，保障全军运行。"},
	"refinery": {Name: "后勤中心", Cost: 650, Time: 21, HP: 760, Power: -20, Size: 84, Icon: "warehouse", Desc: "接收外部运输机物资，完成卸货后到账；子午月表战区负责矿石精炼。"},
	"barracks": {Name: "兵营", Cost: 450, Time: 17, HP: 600, Power: -15, Size: 66, Icon: "users", Desc: "训练步兵与工程师。"},
	"factory":  {Name: "战车工厂", Cost: 850, Time: 27, HP: 850, Power: -35, Size: 88, Icon: "truck", Desc: "生产坦克、运输车、维修补给车与采矿车，独立装甲生产线。", Requires: "refinery"},
	"armory":   {Name: "兵工厂", Cost: 780, Time: 25, HP: 760, Power: -25, Size: 82, Icon: "warehouse", Desc: "生产防空、无人机、电子战、远程火力与阵营装备，独立生产队列。", Requires: "factory"},
	"dock":     {Name: "海军港口", Cost: 850, Time: 27, HP: 800, Power: -25, Size: 92, Icon: "anchor", Desc: "生产舰艇、接收集装箱物资；有电且脱战后维修舰艇和潜艇、补弹及补充舰载机。", Requires: "factory", Naval: true},
	"radar":    {Name: "雷达站", Cost: 550, Time: 22, HP: 580, Power: -30, Size: 68, Icon: "radar", Desc: "扩大视野并解锁防空、特色单位。", Requires: "factory"},
	"airfield": {Name: "空军基地", Cost: 980, Time: 31, HP: 760, Power: -45, Size: 92, Icon: "plane", Desc: "生产制空与对地战机；有电时可补充弹药。", Requires: "radar"},
	"turret":   {Name: "防御炮塔", Cost: 440, Time: 16, HP: 600, Power: -20, Size: 56, Icon: "crosshair", Desc: "自动攻击地面目标；高空战机需要防空车拦截。", Requires: "barracks"},
	"lab":      {Name: "作战实验室", Cost: 950, Time: 33, HP: 680, Power: -45, Size: 70, Icon: "flask-conical", Desc: "解锁阵营特色单位和终极技能。", Requires: "radar"},
	"super":    {Name: "战略武器站", Cost: 1200, Time: 38, HP: 700, Power: -70, Size: 74, Icon: "orbit", Desc: "蓄能后释放阵营终极技能；另有网络战：120 秒充能、500 资金，预警 12 秒后干扰敌方新指令 4 秒。摧毁源站或断电可解除。", Requires: "lab"},
}

// UnitSpec 对齐 UNITS 条目。零值表示字段缺省。
type UnitSpec struct {
	Name       string
	Cost       float64
	Time       float64
	HP         float64
	Speed      float64
	Range      float64
	MinRange   float64
	Damage     float64
	Cooldown   float64
	Sight      float64
	Ammo       float64
	AmmoCost   float64
	RearmTime  float64
	Sonar      float64
	Capacity   float64
	Stock      float64
	Splash     float64
	DeployTime float64
	Wing       float64
	Icon       string
	Producer   string
	Requires   string
	Faction    string
	Map        string
	Naval      bool
	Tags       []string
	Desc       string
}

var UNITS = map[string]*UnitSpec{
	"rifle":      {Name: "突击兵", Cost: 120, Time: 8, HP: 110, Speed: 82, Range: 140, Damage: 12, Cooldown: 0.75, Sight: 260, Icon: "user-round", Producer: "barracks", Tags: []string{"infantry"}, Desc: "经济、可靠的基础作战单位。"},
	"engineer":   {Name: "工程师", Cost: 220, Time: 12, HP: 80, Speed: 76, Range: 0, Damage: 0, Cooldown: 1, Sight: 230, Icon: "wrench", Producer: "barracks", Tags: []string{"infantry", "engineer"}, Desc: "右键油井、信标或能源仓占领，成功后驻留；可回收物资。右键己方被潜入设施，停驻 3 秒拆弹，工程师保留。"},
	"scout":      {Name: "间谍", Cost: 320, Time: 16, HP: 75, Speed: 104, Range: 0, Damage: 0, Cooldown: 1, Sight: 480, Icon: "binoculars", Producer: "barracks", Tags: []string{"infantry", "scout"}, Desc: "无武装，480 视野与 260 范围反隐；右键可见敌方设施一次性潜入，45 秒后引爆。敌方工程师可在 3 秒内拆弹；不伪装敌军，不隐身。"},
	"tank":       {Name: "主战坦克", Cost: 480, Time: 20, HP: 380, Speed: 62, Range: 205, Damage: 38, Cooldown: 1.3, Sight: 280, Icon: "shield", Producer: "factory", Tags: []string{"armor"}, Desc: "重装核心，克制步兵和防空单位。"},
	"drone":      {Name: "攻击无人机", Cost: 260, Time: 13, HP: 130, Speed: 116, Range: 180, Damage: 20, Cooldown: 0.85, Sight: 330, Icon: "scan", Producer: "armory", Tags: []string{"drone", "air"}, Desc: "机动灵活，对装甲目标有额外伤害。"},
	"ghost":      {Name: "隐形侦察无人机", Cost: 340, Time: 18, HP: 85, Speed: 124, Range: 0, Damage: 0, Cooldown: 1, Sight: 450, Icon: "eye-off", Producer: "armory", Requires: "radar", Faction: "middleeast", Tags: []string{"drone", "air", "scout", "stealth"}, Desc: "新月专属；深入敌后提供视野，接近侦察兵、防空车或雷达信标会暴露。"},
	"aa":         {Name: "防空车", Cost: 360, Time: 17, HP: 230, Speed: 72, Range: 250, Damage: 22, Cooldown: 0.8, Sight: 320, Icon: "radar", Producer: "armory", Requires: "radar", Tags: []string{"vehicle", "anti-air"}, Desc: "防空与弹药拦截，不能对地；需要装甲或步兵护卫。"},
	"fighter":    {Name: "制空战机", Cost: 620, Time: 24, HP: 215, Speed: 210, Range: 275, Damage: 37, Cooldown: 0.85, Sight: 470, Ammo: 9, RearmTime: 8, Icon: "plane", Producer: "airfield", Tags: []string{"air", "jet", "anti-air"}, Desc: "优先制空；可对地对舰，但伤害仅基础的35%；各任务共用有限弹药，返场整备。"},
	"strike":     {Name: "对地攻击机", Cost: 760, Time: 29, HP: 235, Speed: 170, Range: 245, Damage: 62, Cooldown: 1.55, Sight: 420, Ammo: 3, RearmTime: 10, Icon: "plane-takeoff", Producer: "airfield", Tags: []string{"air", "jet", "bomber"}, Desc: "优先对地对舰；具备基础伤害35%的空战自卫能力，各任务共用有限弹药，返场整备。"},
	"patrol":     {Name: "近海巡逻艇", Cost: 380, Time: 18, HP: 270, Speed: 110, Range: 235, Damage: 26, Cooldown: 0.9, Sight: 355, Sonar: 150, Icon: "ship", Producer: "dock", Naval: true, Tags: []string{"ship", "naval", "anti-sub"}, Desc: "高速夺取海面优势，近距声呐发现潜艇；对建筑伤害较低。"},
	"frigate":    {Name: "防空护卫舰", Cost: 760, Time: 29, HP: 470, Speed: 72, Range: 335, Damage: 38, Cooldown: 1.55, Sight: 435, Sonar: 230, Icon: "sailboat", Producer: "dock", Naval: true, Tags: []string{"ship", "naval", "anti-air", "anti-sub"}, Desc: "区域防空与反潜护航；可拦截巡飞弹和舰载机。"},
	"harvester":  {Name: "采矿车", Cost: 440, Time: 19, HP: 350, Speed: 59, Range: 0, Damage: 0, Cooldown: 1, Sight: 240, Icon: "pickaxe", Producer: "factory", Tags: []string{"vehicle", "harvester"}, Desc: "自动采集矿石，扩张经济。"},
	"elite":      {Name: "阵营特色单位", Cost: 650, Time: 27, HP: 330, Speed: 65, Range: 230, Damage: 35, Cooldown: 1.25, Sight: 320, Icon: "sparkles", Producer: "armory", Requires: "lab", Tags: []string{"elite"}, Desc: "阵营专属战术单位。"},
	"loiterer":   {Name: "巡飞弹发射车", Cost: 620, Time: 26, HP: 250, Speed: 64, Range: 520, MinRange: 100, Damage: 95, Cooldown: 4.5, Sight: 260, Ammo: 4, RearmTime: 8, Icon: "send", Producer: "armory", Requires: "radar", Tags: []string{"vehicle", "artillery"}, Desc: "共享侦察视野发射追踪巡飞弹；可被激光、防空与电子干扰反制。"},
	"jammer":     {Name: "电子干扰车", Cost: 520, Time: 23, HP: 240, Speed: 70, Range: 235, Damage: 0, Cooldown: 1, Sight: 300, Icon: "radio-tower", Producer: "armory", Requires: "radar", Tags: []string{"vehicle", "support"}, Desc: "235 范围压制敌方无人机与导引弹药；不造成直接伤害，不能反制无制导火箭。"},
	"laser":      {Name: "激光反无人机车", Cost: 580, Time: 25, HP: 220, Speed: 67, Range: 250, Damage: 45, Cooldown: .75, Sight: 320, Icon: "focus", Producer: "armory", Requires: "radar", Tags: []string{"vehicle", "anti-drone"}, Desc: "拦截巡飞弹、击落无人机；连续射击会过热，不能攻击坦克或高空战机。"},
	"rocket":     {Name: "远程火箭炮", Cost: 880, Time: 34, HP: 230, Speed: 52, Range: 680, MinRange: 180, Damage: 85, Cooldown: 3.8, Sight: 240, Ammo: 6, RearmTime: 10, DeployTime: 2, Icon: "rocket", Producer: "armory", Requires: "lab", Tags: []string{"vehicle", "artillery"}, Desc: "展开 2 秒后发射区域火箭；需要侦察，180 最小射程，有限弹药。"},
	"apc":        {Name: "装甲运输车", Cost: 420, Time: 19, HP: 430, Speed: 90, Range: 0, Damage: 0, Cooldown: 1, Sight: 280, Capacity: 4, Icon: "bus-front", Producer: "factory", Tags: []string{"vehicle", "armor", "transport"}, Desc: "运载 4 名步兵或工程师；车毁乘员受伤撤出，无法无敌穿越封锁线。"},
	"supply":     {Name: "维修补给车", Cost: 460, Time: 21, HP: 230, Speed: 64, Range: 115, Damage: 0, Cooldown: 1, Sight: 270, Stock: 120, Icon: "wrench", Producer: "factory", Tags: []string{"vehicle", "support"}, Desc: "自动寻找可达的友军步兵和陆地载具，脱战后治疗、维修和补弹；消耗库存与资金，缺货自动回厂。"},
	"destroyer":  {Name: "导弹驱逐舰", Cost: 1150, Time: 40, HP: 800, Speed: 67, Range: 470, Damage: 72, Cooldown: 2.6, Sight: 440, Sonar: 290, Ammo: 8, RearmTime: 10, Icon: "ship", Producer: "dock", Requires: "radar", Map: "ocean", Tags: []string{"ship", "naval", "anti-air", "anti-sub", "artillery"}, Desc: "远程对舰、防空与反潜；导弹有飞行时间，须返回有电船坞付费补弹。"},
	"carrier":    {Name: "舰队航空母舰", Cost: 1900, Time: 58, HP: 1400, Speed: 43, Range: 720, Damage: 95, Cooldown: 4, Sight: 420, Ammo: 6, RearmTime: 14, Wing: 3, Icon: "plane-takeoff", Producer: "dock", Requires: "lab", Map: "ocean", Tags: []string{"ship", "naval", "artillery"}, Desc: "3 架可被击落的舰载机执行远程打击；飞机返舰，损失后在船坞付费补充，惧怕潜艇。"},
	"submarine":  {Name: "攻击潜艇", Cost: 920, Time: 35, HP: 420, Speed: 64, Range: 340, Damage: 110, Cooldown: 3.5, Sight: 360, Sonar: 190, Ammo: 5, RearmTime: 10, Icon: "waves", Producer: "dock", Requires: "radar", Map: "ocean", Tags: []string{"ship", "naval", "submerged", "stealth"}, Desc: "潜航伏击舰艇；声呐或发射后短暂暴露，可被反潜舰反制，不能攻击陆地。"},
	// 以下对齐 data.js Object.assign(UNITS, {...}) 段。
	"ewPlane":       {Name: "电子压制机", Cost: 1350, Time: 43, HP: 185, Speed: 168, Range: 230, Damage: 0, Cooldown: 1, Sight: 410, Producer: "airfield", Requires: "lab", Tags: []string{"air", "jet", "support"}, Icon: "radio-tower", Desc: "原创概念：每 10 秒在 230 范围内压制敌方防空 2.5 秒；自身无武装，制空机与范围外防空可反制。"},
	"navalFighter":  {Name: "舰载制空机", Cost: 760, Time: 29, HP: 225, Speed: 210, Range: 275, Damage: 37, Cooldown: 0.85, Sight: 470, Ammo: 8, RearmTime: 8, Icon: "plane", Producer: "airfield", Requires: "lab", Map: "ocean", Tags: []string{"air", "jet", "anti-air", "deck"}, Desc: "优先制空，兼顾35%基础伤害对地；可编入本方航母，停航着舰、付费维修补弹。"},
	"navalStrike":   {Name: "舰载攻击机", Cost: 880, Time: 33, HP: 245, Speed: 170, Range: 245, Damage: 62, Cooldown: 1.55, Sight: 420, Ammo: 4, RearmTime: 10, Icon: "plane-takeoff", Producer: "airfield", Requires: "lab", Map: "ocean", Tags: []string{"air", "jet", "bomber", "deck"}, Desc: "通用游戏舰载型，对地与对舰；可编入航母，每舰最多三架真实机体，与自带攻击编组互斥。"},
	"freightPlane":  {Name: "后勤运输机", Cost: 0, Time: 1, HP: 290, Speed: 160, Range: 0, Damage: 0, Cooldown: 1, Sight: 90, Icon: "package", Tags: []string{"air", "jet", "logistics"}, Desc: "外部援助航班，按固定航线自动运抵后勤中心；无武装，可被防空和战机拦截，不受战斗指令控制。"},
	"containerShip": {Name: "集装箱补给船", Cost: 0, Time: 1, HP: 620, Speed: 92, Range: 0, Damage: 0, Cooldown: 1, Sight: 100, Icon: "container", Tags: []string{"ship", "naval", "logistics"}, Desc: "沿海运路线自动回港卸货；无武装，可被舰艇和潜艇袭击，需要护航，不可载兵或刷补给资金。"},
	"landing":       {Name: "坦克登陆舰", Cost: 900, Time: 36, HP: 850, Speed: 59, Range: 0, Damage: 0, Cooldown: 1, Sight: 310, Capacity: 12, Icon: "ship", Producer: "dock", Requires: "radar", Map: "ocean", Tags: []string{"ship", "naval", "transport"}, Desc: "12 格载重：步兵 1 格、陆地车辆 4 格；靠近海岸装卸，深海无法卸载，需舰队护航。"},
	"bomber":        {Name: "远程轰炸机", Cost: 1250, Time: 44, HP: 320, Speed: 142, Range: 300, Damage: 105, Splash: 60, Cooldown: 2.8, Sight: 380, Ammo: 4, RearmTime: 14, Icon: "plane", Producer: "airfield", Requires: "lab", Tags: []string{"air", "jet", "bomber"}, Desc: "有限炸弹，对地范围打击；无法空战，友军免伤是游戏化规则，返场付费维修补弹。"},
	"airlift":       {Name: "重型运输机", Cost: 1000, Time: 38, HP: 380, Speed: 158, Range: 0, Damage: 0, Cooldown: 1, Sight: 360, Capacity: 8, RearmTime: 10, Icon: "plane-takeoff", Producer: "airfield", Requires: "radar", Tags: []string{"air", "jet", "transport"}, Desc: "8 格载重，可运 8 名步兵或 2 辆坦克；停驻在安全陆地上装卸，不能海上空投，无武装。"},
	"railgun":       {Name: "凌霄电磁炮车", Cost: 940, Time: 35, HP: 290, Speed: 49, Range: 430, MinRange: 110, Damage: 96, Cooldown: 3.2, Sight: 270, Ammo: 4, RearmTime: 10, DeployTime: 1.8, Producer: "armory", Requires: "lab", Map: "meridian", Tags: []string{"armor", "artillery"}, Icon: "zap", Desc: "原创科幻；展开后直射穿甲，受地形遮挡，不能防空，有限弹药。"},
	"aegis":         {Name: "云隼无人制空机", Cost: 790, Time: 30, HP: 195, Speed: 215, Range: 295, Damage: 40, Cooldown: .9, Sight: 475, Ammo: 6, RearmTime: 10, Producer: "airfield", Requires: "lab", Map: "meridian", Tags: []string{"air", "jet", "anti-air"}, Icon: "plane", Desc: "原创科幻；仅制空，需返场补弹，离子扰动期间需中继保护。"},
	"relay":         {Name: "子午通信中继车", Cost: 540, Time: 24, HP: 220, Speed: 65, Range: 0, Damage: 0, Cooldown: 1, Sight: 420, Producer: "factory", Requires: "radar", Map: "meridian", Tags: []string{"vehicle", "support", "scout"}, Icon: "radio-tower", Desc: "原创科幻；260 范围保护友军空中链路与制导弹药，不造成伤害。"},
}

// 弹药循环对齐 data.js:186-189：七型加弹药字段并追加描述。
func init() {
	for _, e := range []struct {
		Type      string
		Ammo      float64
		AmmoCost  float64
		RearmTime float64
	}{
		{"rifle", 18, 1, 9}, {"tank", 12, 6, 12}, {"aa", 16, 4, 10}, {"elite", 12, 6, 12},
		{"drone", 8, 4, 8}, {"patrol", 24, 3, 12}, {"frigate", 16, 5, 12},
	} {
		u := UNITS[e.Type]
		u.Ammo, u.AmmoCost, u.RearmTime = e.Ammo, e.AmmoCost, e.RearmTime
		u.Desc += " 有限弹药，耗尽后自动补给。"
	}
	// data.js:194 carrier 覆写。
	UNITS["carrier"].Capacity = 3
	UNITS["carrier"].Desc = "可驻泊三架自行生产的兼容舰载机；停航着舰、维修补弹、弹射起飞。编入真实舰载机后不再同时出动自带攻击编组，需反潜护卫。"
}

// BUILD_ORDER 对齐 data.js:191。
var BUILD_ORDER = []string{"power", "refinery", "barracks", "factory", "armory", "dock", "radar", "airfield", "turret", "lab", "super"}

// UNIT_ORDER 对齐 data.js:192-196（顺序即 UI 与遍历序）。
var UNIT_ORDER = append([]string{
	"rifle", "engineer", "scout", "tank", "apc", "supply", "harvester", "drone", "ghost", "aa",
	"loiterer", "jammer", "laser", "rocket", "elite", "fighter", "strike", "patrol", "frigate",
	"destroyer", "carrier", "submarine",
}, "railgun", "aegis", "relay", "landing", "bomber", "airlift", "navalFighter", "navalStrike", "ewPlane")

// SupportsMap 对齐 data.js:197。
func SupportsMap(required, mapID string) bool {
	return required == "" || required == mapID || (required == "ocean" && mapID == "archipelago")
}

// OreEntry 对齐 ore 数组 [x, y, amount, kind]。
type OreEntry struct {
	X      float64
	Y      float64
	Amount float64
	Kind   string
}

// ORE_LAYOUT 对齐 data.js:199-209。
var ORE_LAYOUT = []OreEntry{
	{330, 510, 2100, "gold"}, {330, 930, 2100, "gold"},
	{760, 320, 1500, "gem"}, {760, 1120, 1500, "gem"},
	{1110, 590, 2500, "gold"}, {1130, 850, 2500, "gold"},
	{1480, 320, 1500, "gem"}, {1480, 1120, 1500, "gem"},
	{1910, 510, 2100, "gold"}, {1910, 930, 2100, "gold"},
}
