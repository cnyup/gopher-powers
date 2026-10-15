// Package equip 对齐源项目 src/equipment.js。
// 原型资料只影响名称、简介与外形，不进入模拟数值。
package equip

import (
	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/lunar"
)

// Entry 对齐 EQUIPMENT_SYSTEMS 条目 [名称, 简介, 来源键]。
type Entry struct {
	Name      string
	Desc      string
	SourceKey string
}

// equipmentSystems 对齐 equipment.js:7-12（含尾部 china.destroyer 覆写）。
var equipmentSystems = map[string]map[string]Entry{
	"china": {
		"tank":   {"99A 主战坦克", "数字化主战坦克原型；在游戏中负责装甲推进。", "china-tank"},
		"rocket": {"PHL-03 远程火箭炮", "多管远程火箭炮原型；侦察后展开射击，停驻补弹。", "china-rocket"},
		"fighter": {"歼-20 制空编组", "原创鸭翼双垂尾简化外形；游戏以制空为主，兼顾较弱对地任务，不代表真实挂载或性能。", "china-air"},
		"aa":     {"红旗-17 防空车", "野战防空系统参考；护卫地面纵队，拦截空中目标。", "china-aa"},
		"destroyer": {"052D 导弹驱逐舰", "052D 公开外观参考：相控阵雷达、垂发甲板与直升机平台；原创简化模型，战力仍遵循统一平衡。", "china-destroyer"},
	},
	"russia": {
		"tank":   {"T-90MS 主战坦克", "公开出口型号参考，不代表俄军实际编制；强调装甲突击。", "russia-tank"},
		"rocket": {"BM-30 龙卷风", "多管火箭炮原型；展开时间与补给制约持续火力。", "russia-rocket"},
		"fighter": {"Su-35 制空编组", "多用途战斗机参考；本游戏按制空任务配置。", "russia-air"},
		"apc":    {"BTR-82A 运输编组", "轮式装甲输送车参考；本游戏侧重运载，不模拟其机炮。", "russia-tank"},
	},
	"nato": {
		"tank":   {"M1A2 艾布拉姆斯", "主战坦克原型；重装推进依赖防空与补给护卫。", "nato-tank"},
		"rocket": {"M142 HIMARS", "轮式模块化火箭炮参考；有限弹药，展开后远距打击。", "nato-rocket"},
		"fighter": {"F-22 制空编组", "制空战斗机参考；空中优势不能代替地面占领。", "nato-air"},
		"strike": {"F-35 对地编组", "多用途战斗机参考；游戏按对地载荷配置，不代表只能对地。", "nato-strike"},
		"apc":    {"Stryker 运输编组", "轮式装甲车族参考；快速运输与部署步兵。", "nato-tank"},
	},
	"asia": {
		"tank":   {"K2 黑豹主战坦克", "韩国主战坦克原型；机动、护卫与侧翼推进相配合。", "asia-tank"},
		"rocket": {"K239 天舞", "模块化多管火箭炮参考；需保护展开位置和补给线。", "asia-rocket"},
		"fighter": {"KF-16 制空编组", "韩国F-16体系参考；游戏中担任制空与空中护航。", "asia-air"},
	},
	"middleeast": {
		"tank":     {"Merkava Mk.4 梅卡瓦", "以色列主战坦克参考；保留本阵营的游戏平衡弱点。", "middleeast-tank"},
		"rocket":   {"PULS 火箭炮", "模块化火箭炮系统参考；共享侦察，消耗有限弹药。", "middleeast-rocket"},
		"fighter":  {"F-16I 雷暴编组", "以色列F-16I参考；游戏化制空配置。", "middleeast-air"},
		"loiterer": {"HAROP 巡飞弹编组", "巡飞弹原型参考；一次性弹药，可被拦截与干扰。", "middleeast-loiterer"},
	},
}

// countries 对齐各阵营 country 字段。
var countries = map[string]string{
	"china": "中国装备", "russia": "俄系装备", "nato": "北约／美系装备",
	"asia": "韩国装备", "middleeast": "以色列装备",
}

// sources 对齐 EQUIPMENT_SOURCES（含 china-destroyer 追加）。
var sources = map[string]string{
	"china-tank":    "https://www.mod.gov.cn/gfbw/wzll/16144039.html",
	"china-rocket":  "https://eng.mod.gov.cn/xb/Home/Focus/4845434.html",
	"china-air":     "https://www.81.cn/kt/10198589.html",
	"china-aa":      "https://tv.81.cn/zgjs/jskj/10183166.html",
	"china-destroyer": "https://www.mod.gov.cn/djzx/4809824.html",
	"russia-tank":   "https://roe.ru/upload/pdf/11402_post.pdf",
	"russia-rocket": "https://roe.ru/pdfs/pdf_2699.pdf",
	"russia-air":    "https://roe.ru/pdfs/pdf_2699.pdf",
	"nato-tank":     "https://www.gdls.com/",
	"nato-rocket":   "https://www.lockheedmartin.com/en-us/products/himars/media-kit.html",
	"nato-air":      "https://www.lockheedmartin.com/en-us/products/f-22.html",
	"nato-strike":   "https://www.lockheedmartin.com/en-us/products/f-35/f-35-about.html",
	"asia-tank":     "https://www.hyundai-rotem.co.kr/en/business/defense/details.do?productCt03=defense0101",
	"asia-rocket":   "https://www.hanwhaaerospace.com/eng/whoweare/history.do",
	"asia-air":      "https://www.lockheedmartin.com/en-kr/index.html",
	"middleeast-tank": "https://www.idf.il/en/mini-sites/our-units/armored-corps/armored-corps/",
	"middleeast-rocket": "https://www.elbitsystems.com/product/puls/",
	"middleeast-air": "https://www.lockheedmartin.com/en-il/index.html",
	"middleeast-loiterer": "https://www.iai.co.il/product/harop/",
}

// Profile 对齐 equipment.js:34 equipmentProfile 返回结构。
type Profile struct {
	Name        string
	Description string
	Country     string
	Category    string
	Source      string // 空串 = null
	HasSource   bool
}

var spaceCraftNames = map[string]string{
	"fighter": "云隼制空穿梭机", "strike": "月弧对地穿梭机", "bomber": "月隼轨道轰炸机",
	"airlift": "星舟战术运输机", "freightPlane": "月表物资穿梭机",
}

var conceptTypes = map[string]bool{
	"elite": true, "ghost": true, "drone": true, "jammer": true, "laser": true,
	"railgun": true, "aegis": true, "relay": true, "ewPlane": true,
}

// ProfileOf 对齐 equipment.js:34-40 equipmentProfile。
func ProfileOf(faction, typ string, future bool) *Profile {
	system, hasSystem := equipmentSystems[faction]
	var entry Entry
	hasEntry := false
	if hasSystem {
		entry, hasEntry = system[typ]
	}
	d, hasData := gpdata.UNITS[typ]
	if !hasData && !lunar.IsRobot(future, typ) {
		return nil
	}
	if lunar.IsRobot(future, typ) {
		robot := lunar.RobotSpecs[typ]
		return &Profile{Name: robot.Name, Description: robot.Description, Country: "原创月表无人体系", Category: "原创／机器人"}
	}
	if future {
		if n, ok := spaceCraftNames[typ]; ok {
			return &Profile{Name: n, Description: "原创科幻无人平台，推进舱、姿态喷口与功能模块；游戏职责和成本仍遵循双方统一规则。", Country: "原创月表无人体系", Category: "原创／科幻装备"}
		}
	}
	if hasEntry {
		return &Profile{Name: entry.Name, Description: entry.Desc, Country: countries[faction], Category: "现实原型", Source: sources[entry.SourceKey], HasSource: true}
	}
	country := "通用体系"
	if hasSystem {
		country = countries[faction]
	}
	name := d.Name
	if typ == "elite" {
		name = gpdata.FACTIONS[faction].Elite
	}
	category := "通用战术单位"
	if conceptTypes[typ] {
		category = "原创／概念装备"
	}
	return &Profile{Name: name, Description: d.Desc, Country: country, Category: category}
}

// ModelOf 对齐 equipment.js:42-54 equipmentModel。
func ModelOf(faction, typ string, future bool) string {
	if lunar.IsRobot(future, typ) {
		return lunar.RobotSpecs[typ].Model
	}
	if typ == "ewPlane" {
		if future {
			return "space_ewPlane"
		}
		return "ewPlane"
	}
	if future {
		switch typ {
		case "fighter", "strike", "bomber", "airlift", "freightPlane", "aegis":
			return "space_" + typ
		}
	}
	switch {
	case typ == "navalFighter":
		return "fighter"
	case typ == "navalStrike":
		return "strike"
	case typ == "refinery" && !future:
		return "logistics_depot"
	case typ == "destroyer" && faction == "china":
		return "destroyer_china"
	case typ == "tank", typ == "rocket":
		return typ + "_" + faction
	case typ == "fighter" && faction == "china":
		return "fighter_china"
	}
	if future {
		if _, ok := gpdata.UNITS[typ]; !ok {
			return "future_" + typ
		}
	}
	if typ == "elite" {
		return "elite_" + faction
	}
	return typ
}
