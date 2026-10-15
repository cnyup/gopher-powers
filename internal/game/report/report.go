// Package report 对齐源项目 src/battle-report.js。
package report

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
)

// ReportFields 对齐 battle-report.js:3（顺序即导出列序）。
var ReportFields = []string{"damageDealt", "damageTaken", "kills", "losses", "buildingsDestroyed", "buildingsLost", "unitsProduced", "repairHP", "resupplies", "oilSeconds", "beaconSeconds"}

// SideStats 单方统计。
type SideStats map[string]float64

// Event 关键战损事件。
type Event struct {
	Time  float64 `json:"time"`
	Owner int     `json:"owner"`
	Type  string  `json:"type"`
	Kind  string  `json:"kind"` // unitLost / buildingLost
}

// Stats 对齐 createBattleStats。
type Stats struct {
	Since    float64    `json:"since"`
	Complete bool       `json:"complete"`
	Sides    []SideStats `json:"sides"`
	Events   []Event    `json:"events"`
}

// Create 对齐 battle-report.js:5-7。
func Create(since float64, count int) *Stats {
	s := &Stats{Since: since, Complete: since == 0, Sides: make([]SideStats, count), Events: []Event{}}
	for i := range s.Sides {
		s.Sides[i] = SideStats{}
		for _, k := range ReportFields {
			s.Sides[i][k] = 0
		}
	}
	return s
}

// MetricView battleMetric 需要的最小游戏视图（running/battleStats 存在性）。
type MetricView interface {
	IsRunning() bool
	Stats() *Stats
}

// Metric 对齐 battle-report.js:8-10：running 且有限正数才累计。
func Metric(g MetricView, side int, key string, amount float64) {
	if g.IsRunning() && g.Stats() != nil && side < len(g.Stats().Sides) && !math.IsInf(amount, 0) && !math.IsNaN(amount) && amount > 0 {
		g.Stats().Sides[side][key] += amount
	}
}

// DamageTarget 对齐 recordBattleDamage 的目标视图。
type DamageTarget interface {
	Owner() int
	Kind() string
	Type() string
	HP() float64
}

// RecordDamage 对齐 battle-report.js:11-19：击杀判定 + 事件裁剪 32。
func RecordDamage(g MetricView, target DamageTarget, amount float64, attacker int) {
	actual := amount
	if target.HP() < actual {
		actual = target.HP()
	}
	Metric(g, target.Owner(), "damageTaken", actual)
	if attacker >= 0 && attacker != target.Owner() {
		Metric(g, attacker, "damageDealt", actual)
	}
	if amount < target.HP() || g.Stats() == nil || !g.IsRunning() {
		return
	}
	building := target.Kind() == "building"
	if building {
		Metric(g, target.Owner(), "buildingsLost", 1)
	} else {
		Metric(g, target.Owner(), "losses", 1)
	}
	if attacker >= 0 && attacker != target.Owner() {
		if building {
			Metric(g, attacker, "buildingsDestroyed", 1)
		} else {
			Metric(g, attacker, "kills", 1)
		}
	}
	// time 由调用方经 TimeView 提供；此处通过扩展接口取。
	tv := g.(interface{ Time() float64 })
	g.Stats().Events = append(g.Stats().Events, Event{Time: tv.Time(), Owner: target.Owner(), Type: target.Type(), Kind: kindLost(building)})
	if len(g.Stats().Events) > 32 {
		g.Stats().Events = g.Stats().Events[len(g.Stats().Events)-32:]
	}
}

func kindLost(building bool) string {
	if building {
		return "buildingLost"
	}
	return "unitLost"
}

// ReportRow 战后报表行（delivered/intercepted 来自 logistics）。
type ReportRow struct {
	Since       float64
	Complete    bool
	Sides       []SideStats
	Delivered   []float64 // 各方物流交付
	Intercepted []float64 // 各方物流损失
	Events      []Event
}

// Build 对齐 battle-report.js:20-22。
func Build(stats *Stats, delivered, intercepted []float64) *ReportRow {
	if stats == nil {
		return nil
	}
	sides := make([]SideStats, len(stats.Sides))
	for i, s := range stats.Sides {
		merged := SideStats{}
		for k, v := range s {
			merged[k] = v
		}
		merged["delivered"] = 0
		merged["intercepted"] = 0
		if i < len(delivered) {
			merged["delivered"] = delivered[i]
		}
		if i < len(intercepted) {
			merged["intercepted"] = intercepted[i]
		}
		sides[i] = merged
	}
	return &ReportRow{Since: stats.Since, Complete: stats.Complete, Sides: sides, Delivered: delivered, Intercepted: intercepted, Events: append([]Event{}, stats.Events...)}
}

// Validate 对齐 battle-report.js:23-26（存档校验）。
func Validate(stats *Stats, time float64, count int) bool {
	if stats == nil || math.IsNaN(stats.Since) || stats.Since < 0 || stats.Since > time {
		return false
	}
	if len(stats.Sides) != count {
		return false
	}
	for _, side := range stats.Sides {
		for _, k := range ReportFields {
			v, ok := side[k]
			if !ok || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				return false
			}
		}
	}
	if len(stats.Events) > 32 {
		return false
	}
	for _, e := range stats.Events {
		if math.IsNaN(e.Time) || e.Time < stats.Since || e.Time > time || e.Owner < 0 || e.Owner >= count {
			return false
		}
		if e.Kind == "unitLost" {
			if _, ok := gpdata.UNITS[e.Type]; !ok {
				return false
			}
		} else if e.Kind == "buildingLost" {
			if _, ok := gpdata.BUILDINGS[e.Type]; !ok {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
