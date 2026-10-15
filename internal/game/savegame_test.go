package game

import (
	"encoding/json"
	"strings"
	"testing"
)

// Wave 1e：存档 schema 兼容（源锚点 savegame.js SAVE_KEYS/版本/校验）。

func TestSaveKeysCompat(t *testing.T) {
	// 对齐源 SAVE_KEYS 逐字。
	if SaveKeys["manual"] != "great-powers-save-manual-v1" {
		t.Fatalf("manual key = %s", SaveKeys["manual"])
	}
	if SaveKeys["auto"] != "great-powers-save-auto-v1" {
		t.Fatalf("auto key = %s", SaveKeys["auto"])
	}
}

// createSave → validateSave roundtrip：合法战局自产自校验。
func TestCreateSaveRoundtrip(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	g.Update(0.05) // 推进一帧产生状态
	data, err := g.CreateSave(json.RawMessage(`{"camera":{"x":1}}`), 12345)
	if err != nil {
		t.Fatalf("CreateSave: %v", err)
	}
	// 必须能被自己的校验器接受（roundtrip）。
	if err := ValidateSave(data); err != nil {
		t.Fatalf("roundtrip 校验失败: %v（doc 头=%s）", err, string(data[:120]))
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["format"] != "great-powers-rts" || doc["version"].(float64) != 1 {
		t.Fatalf("format/version 错: %v", doc["format"])
	}
}

// 已结束战局拒绝保存（源语义）。
func TestCreateSaveRejectEnded(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	g.Running = false
	if _, err := g.CreateSave(nil, 1); err == nil || !strings.Contains(err.Error(), "尚未结束") {
		t.Fatalf("应拒绝：%v", err)
	}
}

// parseSave：坏 JSON / 超限 / 格式错三条报错路径（错误文案逐字）。
func TestParseSaveErrors(t *testing.T) {
	if _, err := ParseSave("{bad json"); err == nil || err.Error() != "无法读取存档：不是有效的 JSON 文件" {
		t.Fatalf("坏 JSON 报错文案: %v", err)
	}
	if _, err := ParseSave(`{"format":"other","version":1}`); err == nil || err.Error() != "无法读取此存档：文件类型或存档版本不兼容" {
		t.Fatalf("格式错报错文案: %v", err)
	}
	// 超限。
	big := `{"a":"` + strings.Repeat("x", SaveLimit) + `"}`
	if _, err := ParseSave(big); err == nil || err.Error() != "存档文件过大，无法读取" {
		t.Fatalf("超限报错: %v", err)
	}
}

// 校验器拒损坏存档（源 test/fixtures/corrupt-save.json 同族）。
func TestValidateRejectCorrupt(t *testing.T) {
	g, _ := New("china", "russia", nil, Options{})
	data, _ := g.CreateSave(nil, 1)
	var doc map[string]json.RawMessage
	json.Unmarshal(data, &doc)
	var state map[string]json.RawMessage
	json.Unmarshal(doc["state"], &state)
	// 抽走 players 字段 → SNAPSHOT_FIELDS 校验失败。
	delete(state, "players")
	doc["state"], _ = json.Marshal(state)
	fixed, _ := json.Marshal(doc)
	if err := ValidateSave(fixed); err == nil {
		t.Fatal("缺 players 应被拒")
	}
	// 未知地图。
	var cfgMap map[string]json.RawMessage
	json.Unmarshal(doc["config"], &cfgMap)
	cfgMap["mapId"], _ = json.Marshal("nonsense")
	doc["config"], _ = json.Marshal(cfgMap)
	fixed2, _ := json.Marshal(doc)
	if err := ValidateSave(fixed2); err == nil {
		t.Fatal("未知地图应被拒")
	}
}

// 六方存档 version=2。
func TestSaveVersionSix(t *testing.T) {
	cfg := make([]PlayerConfig, 6)
	factions := []string{"china", "russia", "nato", "asia", "middleeast"}
	for i := range cfg {
		cfg[i] = PlayerConfig{Faction: factions[i%5], ControlType: map[bool]string{true: "human", false: "ai"}[i == 0]}
	}
	g, _ := New("china", "russia", nil, Options{MapID: "sixfront", Players: cfg})
	data, err := g.CreateSave(nil, 1)
	if err != nil {
		t.Fatalf("六方存档: %v", err)
	}
	var doc map[string]any
	json.Unmarshal(data, &doc)
	if doc["version"].(float64) != 2 {
		t.Fatalf("六方 version = %v, want 2", doc["version"])
	}
}
