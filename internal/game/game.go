// Package game 承载模拟核心，逐方法对齐源项目 src/game.js（1383 行）。
// 浮点纪律：所有运算走 float64 运行时（对齐 JS Number）；距离一律
// sqrt(dx²+dy²)（对齐 Math.hypot 两参快路径，坐标域无溢出差异）。
package game

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"sync/atomic"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/lunar"
	"github.com/cnyup/gopher-powers/internal/game/report"
)

var nextID atomic.Int64

// DeckApproach 着舰进近。
type DeckApproach struct {
	CarrierID int
	Start, Until float64
}

// Order 单位指令。type 决定字段语义（对齐 JS order 对象）。
type Order struct {
	Type          string  `json:"type"`
	X             float64 `json:"x,omitempty"`
	Y             float64 `json:"y,omitempty"`
	TargetID      int     `json:"targetId,omitempty"`
	OriginX       float64 `json:"originX,omitempty"`
	OriginY       float64 `json:"originY,omitempty"`
	DestinationX  float64 `json:"destinationX,omitempty"`
	DestinationY  float64 `json:"destinationY,omitempty"`
	Progress      float64 `json:"progress,omitempty"`
	HomeID        int     `json:"homeId,omitempty"`
}

// Unit 对齐 addUnit 产出的单位对象（字段名即 JSON 序列化名，协议兼容锚点）。
type Unit struct {
	ID             int     `json:"id"`
	Kind           string  `json:"kind"`
	Owner          int     `json:"owner"`
	Type           string  `json:"type"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Angle          float64 `json:"angle"`
	TurretAngle    float64 `json:"turretAngle"`
	HP             float64 `json:"hp"`
	MaxHP          float64 `json:"maxHp"`
	Order          *Order  `json:"order"`
	TargetID       int     `json:"targetId"` // 0 = null
	Path           []Point `json:"path"`
	PathTimer      float64 `json:"pathTimer"`
	FireTimer      float64 `json:"fireTimer"`
	MovePulse      float64 `json:"movePulse"`
	Cargo          float64 `json:"cargo"`
	CargoValue     float64 `json:"cargoValue"`
	HarvestState   string  `json:"harvestState"`
	Ammo           *float64 `json:"ammo"` // nil = null
	RearmProgress  float64 `json:"rearmProgress"`
	ResumeOrder    *Order  `json:"resumeOrder"`
	StunUntil      float64 `json:"stunUntil"`
	TemporaryUntil float64 `json:"temporaryUntil"`
	Selected       bool    `json:"selected"`
	Passengers     []int   `json:"passengers"`
	EmbarkedIn     int     `json:"embarkedIn"` // 0 = null
	Stock          *float64 `json:"stock"`
	Wing           *float64 `json:"wing"`
	Heat           float64  `json:"heat"`
	DeployProgress float64 `json:"deployProgress"`
	// 动态衍生字段（JS 运行时挂载，Go 显式声明）：
	OreTargetID   string  `json:"oreTargetId,omitempty"`
	HarvestHome   bool    `json:"-"`
	AutoSupply    *bool   `json:"autoSupply,omitempty"`
	Battery       *float64 `json:"battery,omitempty"`
	Freight       *Freight `json:"freight,omitempty"`
	HomeCarrierID int     `json:"homeCarrierId,omitempty"`
	Deployment    *Deployment `json:"deployment,omitempty"`
	ServiceTargetID int    `json:"serviceTargetId,omitempty"`
	LastMovedAt   float64 `json:"lastMovedAt,omitempty"`
	LastDamageAt  float64 `json:"lastDamageAt,omitempty"`
	LastFireAt    float64 `json:"lastFireAt,omitempty"`
	JammedUntil   float64 `json:"jammedUntil,omitempty"`
	EWSuppressedUntil float64 `json:"ewSuppressedUntil,omitempty"`
	Overheated    bool    `json:"overheated,omitempty"`
	Stunned       bool    `json:"-"`
	Sabotage      *Sabotage `json:"sabotage,omitempty"` // 仅建筑用；保留在 Unit 供视图统一
	ExposedUntil  float64  `json:"exposedUntil,omitempty"`
	AvoidSide     int      `json:"avoidSide,omitempty"`
	AvoidUntil    float64  `json:"avoidUntil,omitempty"`
	AvoidRepathAt float64  `json:"avoidRepathAt,omitempty"`
	SonarFXAt     float64 `json:"-"`
	PulseAt       float64 `json:"-"`
	DeckApproach  *DeckApproach `json:"-"`
	TransportStage string `json:"-"`
	LoadStartedAt float64 `json:"-"`
	LandingID     int    `json:"-"`
	WingRearm     float64 `json:"-"`
	ServiceSearchAt float64 `json:"-"`
	ServiceTimer    float64 `json:"-"`
	LastAttacker    int    `json:"-"`
	BlockedRepathAt float64 `json:"blockedRepathAt,omitempty"`
	PathGoal      string   `json:"-"`
	MotionGoal    *Point   `json:"-"`
	CollectionFX  float64  `json:"-"`
	DropFXAt      float64  `json:"-"` // freight 用
	ServiceFXAt   float64  `json:"-"`
	SpawnRetryAt  float64  `json:"-"` // building 侧也有；冗余声明以便迁移
}

// Point 坐标（JSON 数组形式由调用方处理）。
type Point struct {
	X, Y float64
}

// Freight 对齐 u.freight。
type Freight struct {
	HomeID     int     `json:"homeId"`
	Phase      string  `json:"phase"`
	Progress   float64 `json:"progress"`
	Entry      Point   `json:"entry"`
	Value      float64 `json:"value"`
	Holding    bool    `json:"holding,omitempty"`
	OrbitAngle *float64 `json:"-"`
	DropFXAt   float64  `json:"-"`
}

// Deployment 对齐 u.deployment（出厂滑行）。
type Deployment struct {
	BuildingID int     `json:"buildingId"`
	Start      float64 `json:"start"`
	Until      float64 `json:"until"`
	FromX      float64 `json:"fromX"`
	FromY      float64 `json:"fromY"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
}

// Sabotage 对齐 b.sabotage（间谍定时破坏）。
type Sabotage struct {
	Owner      int     `json:"owner"`
	PlantedAt  float64 `json:"plantedAt"`
	DetonateAt float64 `json:"detonateAt"`
}

// Active 对齐生产队列中的条目。
type Active struct {
	Type     string  `json:"type"`
	Progress float64 `json:"progress"`
	Paid     float64 `json:"paid"`
}

// Building 对齐 addBuilding 产出的建筑对象。
type Building struct {
	ID          int     `json:"id"`
	Kind        string  `json:"kind"`
	Owner       int     `json:"owner"`
	Type        string  `json:"type"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	HP          float64 `json:"hp"`
	MaxHP       float64 `json:"maxHp"`
	Size        float64 `json:"size"`
	Angle       float64 `json:"angle"`
	SpawnAngle  *float64 `json:"spawnAngle"`
	Queue       []string `json:"queue"`
	Active      *Active `json:"active"`
	FireTimer   float64  `json:"fireTimer"`
	Repairing   bool     `json:"repairing"`
	SpawnAt     float64  `json:"spawnAt"`
	RallyPoint  *Point   `json:"rallyPoint"`
	Sabotage    *Sabotage `json:"sabotage"`
	StunUntil   float64  `json:"stunUntil,omitempty"`
	ExitUntil   float64  `json:"exitUntil,omitempty"`
	SpawnRetryAt float64 `json:"-"`
	LastDamageAt float64  `json:"lastDamageAt,omitempty"`
}

// Site 油井/信标实体。
type Site struct {
	ID    string `json:"id"`
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
	Owner int    `json:"owner"` // -1 = null
}

// OreSite 矿点。
type OreSite struct {
	ID     string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Amount float64 `json:"amount"`
	Max    float64 `json:"max"`
	Kind   string  `json:"kind"`
	Owner  int     `json:"-"`
}

// ResourceSiteT 对齐 resourceSites 条目（battlefield.go 的 ResourceSite 镜像，Owner -1=null）。
type ResourceSiteT = gpdata.ResourceSite

// Player 对齐 makePlayer。
type Player struct {
	Side            int     `json:"side"`
	Faction         string  `json:"faction"`
	Credits         float64 `json:"credits"`
	PowerIn         float64 `json:"powerIn"`
	PowerOut        float64 `json:"powerOut"`
	ControlScore    float64 `json:"controlScore"`
	BuildQueue      *BuildQueue `json:"buildQueue"`
	ReadyBuilding   string  `json:"readyBuilding"` // "" = null
	AbilityCharge   float64 `json:"abilityCharge"`
	AbilityCooldown float64 `json:"abilityCooldown"`
	ShieldUntil     float64 `json:"shieldUntil"`
	AIPlanIndex     int     `json:"aiPlanIndex"`
	AIUnitCount     int     `json:"aiUnitCount"`
	AIAirUnitCount  int     `json:"aiAirUnitCount"`
	AINavyUnitCount int     `json:"aiNavyUnitCount"`
	AIEngineerRetryAt float64 `json:"aiEngineerRetryAt"`
	AIScoutRetryAt    float64 `json:"aiScoutRetryAt"`
	AIGhostRetryAt    float64 `json:"aiGhostRetryAt"`
	UnitCount        int     `json:"unitCount"`
	ControlType      string  `json:"controlType"`
	Eliminated       bool    `json:"eliminated"`
	CyberCharge      float64 `json:"cyberCharge,omitempty"`
	CyberPending     *CyberPending `json:"cyberPending,omitempty"`
	CyberSourceID    int     `json:"cyberSourceId,omitempty"`
	CyberLockedUntil float64 `json:"cyberLockedUntil,omitempty"`
	SatelliteUntil   float64 `json:"satelliteUntil,omitempty"`
	SatelliteReadyAt float64 `json:"satelliteReadyAt,omitempty"`
	RobotChargeLoad  float64 `json:"robotChargeLoad,omitempty"`
	AIFirstStrikeProduced bool `json:"-"`
}

// BuildQueue 对齐 p.buildQueue。
type BuildQueue struct {
	Type     string  `json:"type"`
	Progress float64 `json:"progress"`
	Paid     float64 `json:"paid"`
}

// CyberPending 对齐 p.cyberPending。
type CyberPending struct {
	SourceID  int     `json:"sourceId"`
	TargetSide int    `json:"targetSide"`
	ExecuteAt float64 `json:"executeAt"`
}

// Fog 单方迷雾网格。
type Fog struct {
	Cols, Rows int
	Visible    []bool
	Explored   []bool
}

// Effect 战场特效（快照协议字段）。
type Effect struct {
	Type     string  `json:"type"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	ToX      *float64 `json:"toX,omitempty"`
	ToY      *float64 `json:"toY,omitempty"`
	Age      float64 `json:"age"`
	Duration float64 `json:"duration"`
	Owner    int     `json:"owner"`
	Style    string  `json:"style,omitempty"`
	Faction  string  `json:"faction,omitempty"`
	SourceID int     `json:"sourceId,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
	TargetType string `json:"targetType,omitempty"`
	Amount   *float64 `json:"amount,omitempty"`
	Size     *float64 `json:"size,omitempty"`
	NetworkID int     `json:"networkId,omitempty"`
}

// Projectile 弹道（modern-combat 维护）。
type Projectile struct {
	ID           int     `json:"id"`
	SourceID     int     `json:"sourceId"`
	TargetID     int     `json:"targetId"`
	TargetType   string  `json:"targetType"`
	Kind         string  `json:"kind"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	ToX          float64 `json:"toX"`
	ToY          float64 `json:"toY"`
	StartX       float64 `json:"startX"`
	StartY       float64 `json:"startY"`
	Angle        float64 `json:"angle"`
	Owner        int     `json:"owner"`
	Amount       float64 `json:"amount"`
	Style        string  `json:"style"`
	HP           float64 `json:"hp"`
	Speed        float64 `json:"speed"`
	Age          float64 `json:"age"`
	Jam          float64 `json:"jam"`
	SourceHeight float64 `json:"sourceHeight,omitempty"`
	TargetHeight float64 `json:"targetHeight,omitempty"`
	FlightRole   string  `json:"flightRole,omitempty"`
	FallDuration float64 `json:"fallDuration,omitempty"`
	Returning    bool    `json:"returning"`
	Finished     bool    `json:"finished"`
}

// Events 对齐 events 回调（notice/voice/selection/shot/end）。
type Events struct {
	Notice    func(msg string)
	Voice     func(key string)
	Selection func()
	Shot      func(style string, owner int)
	End       func(winner any) // int 或 "draw"
}

// LogisticsRoute 对齐 initLogistics 条目。
type LogisticsRoute struct {
	Side     int     `json:"side"`
	NextAir  float64 `json:"nextAir"`
	NextSea  float64 `json:"nextSea"`
	Delivered float64 `json:"delivered"`
	Lost     float64 `json:"lost"`
}

// Game 对齐 src/game.js Game 类。
type Game struct {
	Events           *Events
	RNG              func() float64 // 对拍注入；默认 math/rand
	Multiplayer      bool
	BattleID         string
	VictoryMode      string
	Difficulty       gpdata.AIDifficulty
	MapID            string
	BattlefieldScale float64
	Map              *gpdata.MapSpec
	World            gpdata.World
	PlayerCount      int
	PlayerConfigs    []PlayerConfig
	HomeY            float64
	EconomyMode      string
	Projectiles      []*Projectile
	NextProjectileID int
	Time             float64
	Running          bool
	BattleStats      *report.Stats
	Paused           bool
	Winner           any // nil | int | "draw" | "abandoned"
	Units            []*Unit
	Buildings        []*Building
	Ore              []*OreSite
	Oil              []*Site
	Beacons          []*Site
	ResourceSites    []gpdata.ResourceSite
	Effects          []*Effect
	PendingHits      []pendingHit
	CollectingHits   bool
	PendingMoves     []pendingMove
	CollectingMoves  bool
	Selected         []int
	OrderMode        string // "" = null；"patrol"
	PendingBuilding  string // "" = null
	PlacingBuilding  bool
	PendingAbility   bool
	NoticeQueue      []string
	Players          []*Player
	Fogs             []*Fog
	FogTimer         float64
	AIThinkTimer     float64
	AIWaveTimer      float64
	UnitIndex        *unitIndexView
	Online           bool
	BattleReport     any
	AttackAlerts     []AttackAlert
	LastAttackVoiceAt     float64
	LastAttackVoiceBuilding bool
	LogisticsRoutes []*LogisticsRoute
}

// PlayerConfig 席位配置。
type PlayerConfig struct {
	Faction     string
	ControlType string // human / ai
}

// AttackAlert 对齐 attackAlerts 条目。
type AttackAlert struct {
	ID       int     `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	At       float64 `json:"at"`
	Building bool    `json:"building"`
	Type     string  `json:"type"`
}

type pendingHit struct {
	target targetEntity
	amount float64
	side   int
	stun   float64
}

type pendingMove struct {
	unit         *Unit
	goal         Point
	dt           float64
	stopDistance float64
}

// New 对齐 constructor(playerFaction, enemyFaction, events, options)。
func New(playerFaction, enemyFaction string, ev *Events, options Options) (*Game, error) {
	if ev == nil {
		ev = &Events{}
	}
	g := &Game{Events: ev, RNG: defaultRNG}
	g.Multiplayer = options.Multiplayer
	g.BattleID = createBattleID()
	g.VictoryMode = "quick"
	if options.VictoryMode == "annihilation" || options.VictoryMode == "control" {
		g.VictoryMode = options.VictoryMode
	}
	g.Difficulty = gpdata.AI_DIFFICULTIES["standard"]
	if d, ok := gpdata.AI_DIFFICULTIES[options.Difficulty]; ok {
		g.Difficulty = d
	}
	g.MapID = "valley"
	if _, ok := gpdata.MAPS[options.MapID]; ok {
		g.MapID = options.MapID
	}
	g.BattlefieldScale = 1
	if options.BattlefieldScale == 1.5 {
		g.BattlefieldScale = 1.5
	}
	m, err := gpdata.BattlefieldMap(g.MapID, g.BattlefieldScale)
	if err != nil {
		return nil, err
	}
	g.Map = m
	g.World = m.World
	if g.World.Width == 0 {
		g.World = gpdata.WORLD
	}
	g.PlayerCount = 2
	if m.MaxPlayers == 6 {
		g.PlayerCount = 6
	}
	if options.Players != nil {
		if len(options.Players) != g.PlayerCount {
			return nil, errors.New("席位配置无效")
		}
		for _, p := range options.Players {
			if _, ok := gpdata.FACTIONS[p.Faction]; !ok {
				return nil, errors.New("席位配置无效")
			}
			if p.ControlType != "human" && p.ControlType != "ai" {
				return nil, errors.New("席位配置无效")
			}
		}
		g.PlayerConfigs = options.Players
	} else {
		factions := []string{"china", "russia", "nato", "asia", "middleeast"}
		g.PlayerConfigs = make([]PlayerConfig, g.PlayerCount)
		for i := 0; i < g.PlayerCount; i++ {
			ct := "ai"
			if g.Multiplayer {
				ct = "human"
			} else if i == 0 {
				ct = "human"
			}
			f := factions[i%5]
			switch {
			case i == 0:
				f = playerFaction
			case i == 1:
				f = enemyFaction
			}
			g.PlayerConfigs[i] = PlayerConfig{Faction: f, ControlType: ct}
		}
	}
	g.HomeY = g.World.Height / 2
	g.EconomyMode = m.EconomyMode
	if g.EconomyMode == "" {
		if g.MapID == "meridian" {
			g.EconomyMode = "mining"
		} else {
			g.EconomyMode = "convoy"
		}
	}
	g.Running = true
	g.BattleStats = report.Create(0, g.PlayerCount)
	g.Ore = []*OreSite{}
	if g.EconomyMode == "mining" {
		for i, e := range m.Ore {
			g.Ore = append(g.Ore, &OreSite{ID: "ore-" + itoa(i), X: e.X, Y: e.Y, Amount: e.Amount, Max: e.Amount, Kind: e.Kind, Owner: -1})
		}
	}
	g.Oil = make([]*Site, len(m.Oil))
	for i, s := range m.Oil {
		g.Oil[i] = &Site{ID: s.ID, X: s.X, Y: s.Y, Owner: -1}
	}
	g.Beacons = make([]*Site, len(m.Beacons))
	for i, s := range m.Beacons {
		g.Beacons[i] = &Site{ID: s.ID, X: s.X, Y: s.Y, Owner: -1}
	}
	g.ResourceSites = gpdata.ResourceLayout(m)
	g.Players = make([]*Player, g.PlayerCount)
	for i, cfg := range g.PlayerConfigs {
		p := g.makePlayer(i, cfg.Faction)
		p.ControlType = cfg.ControlType
		g.Players[i] = p
	}
	cols := int(g.World.Width / g.World.Fog)
	rows := int(g.World.Height / g.World.Fog)
	g.Fogs = make([]*Fog, g.PlayerCount)
	for i := range g.Fogs {
		g.Fogs[i] = &Fog{Cols: cols, Rows: rows, Visible: make([]bool, cols*rows), Explored: make([]bool, cols*rows)}
	}
	g.AIThinkTimer = 0
	g.AIWaveTimer = g.Difficulty.WaveStart
	g.initLogistics()
	g.initBases()
	g.UpdateFog()
	return g, nil
}

// Options 对齐 constructor options。
type Options struct {
	Multiplayer      bool
	VictoryMode      string
	Difficulty       string
	MapID            string
	BattlefieldScale float64
	Players          []PlayerConfig
}

func defaultRNG() float64 {
	// 对拍 harness 会注入同种子 LCG；默认走 math/rand 全局源。
	return rand.Float64()
}

// createBattleID 对齐 battle-identity.js createBattleId（crypto UUID 形态）。
func createBattleID() string {
	return "battle-" + itoa(int(timeNowMs())) + "-" + randToken(8) + "-" + randToken(8)
}

func (g *Game) makePlayer(side int, faction string) *Player {
	return &Player{
		Side: side, Faction: faction, Credits: 1550,
		AIEngineerRetryAt: g.Difficulty.EngineerStart,
		AIScoutRetryAt:    30, AIGhostRetryAt: 65,
	}
}

// Random 对齐 random(min,max)（依赖注入的 RNG）。
func (g *Game) Random(min, max float64) float64 {
	return min + g.RNG()*(max-min)
}

// --- 基础查询（game.js L135-148）---

// GetEntity 对齐 getEntity：units→buildings→oil→beacons→resourceSites。
// 油井/信标 ID 是字符串，与数字 ID 不冲突；JS 用 == 无类型匹配，此处按同型匹配。
func (g *Game) GetEntity(id int) any {
	for _, u := range g.Units {
		if u.ID == id {
			return u
		}
	}
	for _, b := range g.Buildings {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// GetUnitOrBuilding 数字 id 的单位/建筑查找。
func (g *Game) GetUnit(id int) *Unit {
	for _, u := range g.Units {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (g *Game) GetBuilding(id int) *Building {
	for _, b := range g.Buildings {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// OwnedBuildings 对齐 ownedBuildings(side, type)。
func (g *Game) OwnedBuildings(side int, typ string) []*Building {
	var out []*Building
	for _, b := range g.Buildings {
		if b.Owner == side && b.HP > 0 && (typ == "" || b.Type == typ) {
			out = append(out, b)
		}
	}
	return out
}

// OwnedUnits 对齐 ownedUnits(side, type)。
func (g *Game) OwnedUnits(side int, typ string) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.Owner == side && u.HP > 0 && (typ == "" || u.Type == typ) {
			out = append(out, u)
		}
	}
	return out
}

func (g *Game) HasBuilding(side int, typ string) bool {
	return len(g.OwnedBuildings(side, typ)) > 0
}

func (g *Game) HasPower(side int) bool {
	p := g.Players[side]
	return p.PowerIn >= p.PowerOut
}

func (g *Game) HasRadarIntel(side int) bool {
	for _, s := range g.Beacons {
		if s.Owner == side {
			return true
		}
	}
	return g.HasBuilding(side, "radar") && g.HasPower(side)
}

// Sides 对齐 sides()。
func (g *Game) Sides() []int {
	out := make([]int, len(g.Players))
	for i, p := range g.Players {
		out[i] = p.Side
	}
	return out
}

// EnemySides 对齐 enemySides(side)。
func (g *Game) EnemySides(side int) []int {
	var out []int
	for _, s := range g.Sides() {
		if s != side && !g.Players[s].Eliminated {
			out = append(out, s)
		}
	}
	return out
}

// EnemyUnits 对齐 enemyUnits(side,type)。
func (g *Game) EnemyUnits(side int, typ string) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.Owner != side && u.HP > 0 && u.EmbarkedIn == 0 && (typ == "" || u.Type == typ) {
			out = append(out, u)
		}
	}
	return out
}

// EnemyBuildings 对齐 enemyBuildings(side,type)。
func (g *Game) EnemyBuildings(side int, typ string) []*Building {
	var out []*Building
	for _, b := range g.Buildings {
		if b.Owner != side && b.HP > 0 && (typ == "" || b.Type == typ) {
			out = append(out, b)
		}
	}
	return out
}

// IsAI 对齐 isAI(side)。
func (g *Game) IsAI(side int) bool {
	p := g.Players[side]
	return p.ControlType == "ai" || (p.ControlType == "" && !g.Multiplayer && side == 1)
}

// Distance 对齐 distance(a,b)。
func Distance(ax, ay, bx, by float64) float64 {
	dx, dy := ax-bx, ay-by
	return math.Sqrt(dx*dx + dy*dy)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func itoa(i int) string { return strconv.Itoa(i) }

// IsRobot 便捷委托。
func (g *Game) IsRobot(typ string) bool {
	return lunar.IsRobot(g.Map.Future, typ)
}
