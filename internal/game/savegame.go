package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/report"
)

// savegame.go 对齐 src/savegame.js：SAVE_VERSION/SAVE_LIMIT/SAVE_KEYS/SNAPSHOT_FIELDS/
// validateSave（全量校验树）/parseSave/createSave。JSON schema 字段名一字不改。

const (
	SaveVersion = 1
	SaveLimit   = 8 * 1024 * 1024
)

var SaveKeys = map[string]string{
	"manual": "great-powers-save-manual-v1",
	"auto":   "great-powers-save-auto-auto-v1", // 占位修正见下
}
// 对齐源 SAVE_KEYS。
func init() { SaveKeys["auto"] = "great-powers-save-auto-v1" }

var SnapshotFields = []string{"time", "players", "units", "buildings", "ore", "oil", "beacons", "fogs", "projectiles", "nextProjectileId", "selected", "pendingBuilding", "pendingAbility", "aiTimer", "aiWaveTimer", "fogTimer"}

// intelligence 常量（对齐 intelligence.js INTELLIGENCE_RULES）。
const (
	bombDelay    = 45
	cyberCharge  = 120
	cyberWarning = 12
	cyberDuration = 4
)

var errInvalid = errors.New("存档内容不完整或已损坏，原有战局和存档未被替换")

// saveMap 松散 JSON 对象（校验用）。
type saveMap map[string]json.RawMessage

func asMap(raw json.RawMessage) (saveMap, bool) {
	if len(raw) == 0 || raw[0] != '{' {
		return nil, false
	}
	var m saveMap
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, false
	}
	return m, true
}

func finite(m saveMap, key string) (float64, bool) {
	raw, ok := m[key]
	if !ok {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, false
	}
	return f, true
}

func has(m saveMap, key string) bool { _, ok := m[key]; return ok }

// ValidateSave 对齐 savegame.js validateSave（关键路径全量）。
// 返回 nil = 合法。
func ValidateSave(save json.RawMessage) error {
	var root saveMap
	if err := json.Unmarshal(save, &root); err != nil { return errInvalid
	}
	format, ok := root["format"]
	if !ok || string(format) != `"great-powers-rts"` {
		return errors.New("无法读取此存档：文件类型或存档版本不兼容")
	}
	var version float64
	if err := json.Unmarshal(root["version"], &version); err != nil || (version != 1 && version != 2) {
		return errors.New("无法读取此存档：文件类型或存档版本不兼容")
	}
	s, ok1 := asMap(root["state"])
	config, ok2 := asMap(root["config"])
	if !ok1 || !ok2 { return errInvalid
	}
	count := 2
	if version == 2 {
		count = 6
		var pc float64
		if err := json.Unmarshal(config["playerCount"], &pc); err != nil || pc != 6 { return errInvalid
		}
		var mapID string
		if err := json.Unmarshal(config["mapId"], &mapID); err != nil { return errInvalid
		}
		m, ok := gpdata.MAPS[mapID]
		if !ok || m.MaxPlayers != 6 { return errInvalid
		}
	}
	var mapID string
	if err := json.Unmarshal(config["mapId"], &mapID); err != nil { return errInvalid
	}
	if _, ok := gpdata.MAPS[mapID]; !ok { return errInvalid
	}
	var difficulty string
	if err := json.Unmarshal(config["difficulty"], &difficulty); err != nil { return errInvalid
	}
	if _, ok := gpdata.AI_DIFFICULTIES[difficulty]; !ok { return errInvalid
	}
	var victoryMode string
	if err := json.Unmarshal(config["victoryMode"], &victoryMode); err != nil ||
		(victoryMode != "quick" && victoryMode != "annihilation" && victoryMode != "control") { return errInvalid
	}
	if _, ok := finite(root, "savedAt"); !ok { return errInvalid
	}
	sTime, ok := finite(s, "time")
	if !ok || sTime < 0 { return errInvalid
	}
	// economyMode / battleId / battlefieldScale 可选字段。
	if raw, ok := config["economyMode"]; ok {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil || (v != "mining" && v != "convoy") { return errInvalid
		}
	}
	if raw, ok := config["battlefieldScale"]; ok {
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil || (v != 1 && v != 1.5) { return errInvalid
		}
	}
	for _, key := range SnapshotFields {
		if !has(s, key) { return errInvalid
		}
	}
	// players。
	var players []saveMap
	if err := json.Unmarshal(s["players"], &players); err != nil || len(players) != count { return errInvalid
	}
	for i, p := range players {
		side, ok := finite(p, "side")
		if !ok || side != float64(i) { return errInvalid
		}
		var faction string
		if err := json.Unmarshal(p["faction"], &faction); err != nil { return errInvalid
		}
		if _, ok := gpdata.FACTIONS[faction]; !ok { return errInvalid
		}
		for _, key := range []string{"credits", "abilityCharge", "abilityCooldown", "shieldUntil", "controlScore", "unitCount", "aiPlanIndex", "aiUnitCount", "aiAirUnitCount", "aiNavyUnitCount", "aiEngineerRetryAt", "aiScoutRetryAt", "aiGhostRetryAt"} {
			if _, ok := finite(p, key); !ok { return errInvalid
			}
		}
		if c, _ := finite(p, "credits"); c < 0 { return errInvalid
		}
		if raw, ok := p["buildQueue"]; ok && string(raw) != "null" {
			if err := checkProductionSave(raw, true); err != nil {
				return err
			}
		}
		for _, key := range []string{"satelliteUntil", "satelliteReadyAt"} {
			if raw, ok := p[key]; ok {
				var v float64
				if err := json.Unmarshal(raw, &v); err != nil || v < 0 { return errInvalid
				}
			}
		}
	}
	// units / buildings。
	m, _ := gpdata.BattlefieldMap(mapID, 1)
	world := m.World
	if world.Width == 0 {
		world = gpdata.WORLD
	}
	ids := map[float64]bool{}
	for _, key := range []string{"units", "buildings"} {
		var arr []saveMap
		if err := json.Unmarshal(s[key], &arr); err != nil || len(arr) > 5000 { return errInvalid
		}
		for _, e := range arr {
			id, ok := finite(e, "id")
			if !ok || id < 1 || ids[id] { return errInvalid
			}
			ids[id] = true
			var kind, typ string
			if err := json.Unmarshal(e["kind"], &kind); err != nil { return errInvalid
			}
			if err := json.Unmarshal(e["type"], &typ); err != nil { return errInvalid
			}
			wantKind := "unit"
			defKnown := func(t string) bool { _, ok := gpdata.UNITS[t]; return ok }
			if key == "buildings" {
				wantKind = "building"
				defKnown = func(t string) bool { _, ok := gpdata.BUILDINGS[t]; return ok }
			}
			if kind != wantKind || !defKnown(typ) { return errInvalid
			}
			x, ok1 := finite(e, "x")
			y, ok2 := finite(e, "y")
			_, ok3 := finite(e, "hp")
			maxHp, ok4 := finite(e, "maxHp")
			_, ok5 := finite(e, "angle")
			_, ok6 := finite(e, "fireTimer")
			owner, ok7 := finite(e, "owner")
			if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 || owner < 0 || owner >= float64(count) ||
				x < 0 || x > world.Width || y < 0 || y > world.Height || maxHp <= 0 { return errInvalid
			}
		}
	}
	// fogs。
	var fogs []saveMap
	if err := json.Unmarshal(s["fogs"], &fogs); err != nil || len(fogs) != count { return errInvalid
	}
	cols := world.Width / world.Fog
	rows := world.Height / world.Fog
	for _, fog := range fogs {
		c, ok1 := finite(fog, "cols")
		r, ok2 := finite(fog, "rows")
		if !ok1 || !ok2 || c != cols || r != rows { return errInvalid
		}
		for _, key := range []string{"visible", "explored"} {
			var arr []bool
			if err := json.Unmarshal(fog[key], &arr); err != nil || len(arr) != int(cols*rows) { return errInvalid
			}
		}
	}
	// logistics / battleStats 可选。
	if raw, ok := s["logistics"]; ok {
		var arr []saveMap
		if err := json.Unmarshal(raw, &arr); err != nil || len(arr) != count { return errInvalid
		}
		for _, route := range arr {
			for _, key := range []string{"nextAir", "nextSea", "delivered", "lost"} {
				v, ok := finite(route, key)
				if !ok || v < 0 { return errInvalid
				}
			}
		}
	}
	return nil
}

func checkProductionSave(raw json.RawMessage, building bool) error {
	q, ok := asMap(raw)
	if !ok { return errInvalid
	}
	var typ string
	if err := json.Unmarshal(q["type"], &typ); err != nil { return errInvalid
	}
	if building {
		if _, ok := gpdata.BUILDINGS[typ]; !ok { return errInvalid
		}
	} else {
		if _, ok := gpdata.UNITS[typ]; !ok { return errInvalid
		}
	}
	progress, ok1 := finite(q, "progress")
	paid, ok2 := finite(q, "paid")
	if !ok1 || !ok2 || progress < 0 || paid < 0 { return errInvalid
	}
	return nil
}

// ParseSave 对齐 parseSave。
func ParseSave(text string) (json.RawMessage, error) {
	if len(text) > SaveLimit {
		return nil, errors.New("存档文件过大，无法读取")
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, errors.New("无法读取存档：不是有效的 JSON 文件")
	}
	if err := ValidateSave(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// saveView 快照视图（对齐 view 字段：镜头/选中态由前端持有，Go 侧透传）。
type SaveDoc struct {
	Format   string          `json:"format"`
	Version  int             `json:"version"`
	SavedAt  float64         `json:"savedAt"`
	Config   json.RawMessage `json:"config"`
	State    json.RawMessage `json:"state"`
	View     json.RawMessage `json:"view,omitempty"`
}

// CreateSave 对齐 createSave（进行中方可持续保存）。
func (g *Game) CreateSave(view json.RawMessage, savedAt float64) ([]byte, error) {
	if !g.Running || g.Winner != nil {
		return nil, errors.New("只能保存尚未结束的战局")
	}
	difficulty := "standard"
	for k, d := range gpdata.AI_DIFFICULTIES {
		if d == g.Difficulty {
			difficulty = k
			break
		}
	}
	version := SaveVersion
	playerCountCfg := ""
	if g.PlayerCount == 6 {
		version = 2
		playerCountCfg = `,"playerCount":6`
	}
	scale := g.BattlefieldScale
	if scale == 0 {
		scale = 1
	}
	cfg := fmt.Sprintf(`{"battlefieldScale":%v,"battleId":%q,"mapId":%q,"victoryMode":%q,"difficulty":%q,"economyMode":%q%s}`,
		scale, g.BattleID, g.MapID, g.VictoryMode, difficulty, g.EconomyMode, playerCountCfg)
	// state：SNAPSHOT_FIELDS 直取（json 序列化由结构体 tag 保证字段名一致）。
	stateObj := map[string]any{}
	for _, key := range SnapshotFields {
		switch key {
		case "time":
			stateObj[key] = g.Time
		case "players":
			stateObj[key] = g.Players
		case "units":
			stateObj[key] = g.Units
		case "buildings":
			stateObj[key] = g.Buildings
		case "ore":
			stateObj[key] = g.Ore
		case "oil":
			stateObj[key] = g.Oil
		case "beacons":
			stateObj[key] = g.Beacons
		case "fogs":
			stateObj[key] = g.Fogs
		case "projectiles":
			stateObj[key] = g.Projectiles
		case "nextProjectileId":
			stateObj[key] = g.NextProjectileID
		case "selected":
			stateObj[key] = g.Selected
		case "pendingBuilding":
			stateObj[key] = nilIfEmpty(g.PendingBuilding)
		case "pendingAbility":
			stateObj[key] = g.PendingAbility
		case "aiTimer":
			stateObj[key] = g.AIThinkTimer
		case "aiWaveTimer":
			stateObj[key] = g.AIWaveTimer
		case "fogTimer":
			stateObj[key] = g.FogTimer
		}
	}
	stateObj["logistics"] = g.LogisticsRoutes
	stateObj["resourceSites"] = g.ResourceSites
	stateObj["battleStats"] = g.BattleStats
	stateBytes, err := json.Marshal(stateObj)
	if err != nil {
		return nil, err
	}
	doc := fmt.Sprintf(`{"format":"great-powers-rts","version":%d,"savedAt":%v,"config":%s,"state":%s,"view":%s}`,
		version, savedAt, cfg, stateBytes, viewOrEmpty(view))
	if err := ValidateSave(json.RawMessage(doc)); err != nil {
		// 探针：定位失败段（临时）。
		probeLogSaveErr(doc, cfg)
		return nil, err
	}
	return []byte(doc), nil
}

// probeLogSaveErr 临时探针：把 config 与 state 关键段打到 stderr。
func probeLogSaveErr(doc, cfg string) {
	println("PROBE config:", cfg)
	println("PROBE doc head:", doc[:min2(300, len(doc))])
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func viewOrEmpty(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(`{}`)
	}
	return v
}

var _ = report.Validate // 引用锚（battleStats 校验在 1e 细化接 report.Validate）
var _ = math.Abs
