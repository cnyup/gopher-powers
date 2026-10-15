// parity-replay：Go 侧对拍工具。
// 用法：parity-replay <script.json>——stdout 输出与 JS replay.mjs 同构的快照 JSON。
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cnyup/gopher-powers/internal/game"
)

type script struct {
	FactionA string `json:"factionA"`
	FactionB string `json:"factionB"`
	Options  game.Options `json:"options"`
	Ticks    int `json:"ticks"`
	Commands []struct {
		AtTick     int     `json:"atTick"`
		Action     string  `json:"action"`
		UnitType   string  `json:"unitType"`
		X         float64 `json:"x"`
		Y         float64 `json:"y"`
		AttackMove bool   `json:"attackMove"`
		Selected   []int   `json:"selected"`
	} `json:"commands"`
}

type unitView struct {
	ID    int     `json:"id"`
	Type  string  `json:"type"`
	Owner int     `json:"owner"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	HP    float64 `json:"hp"`
}

type snap struct {
	Tick      int         `json:"tick"`
	Time      float64     `json:"time"`
	Running   bool        `json:"running"`
	Winner    interface{} `json:"winner"`
	Credits   []float64   `json:"credits"`
	Units     []int       `json:"units"`
	Buildings []int       `json:"buildings"`
	UnitList  []unitView  `json:"unitList"`
	Ore       []float64   `json:"ore"`
}

func round3(v float64) float64 { return float64(int64(v*1000+sign(v)*0.5)) / 1000 }

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// lcg 与 JS 侧 makeLCG 同实现同种子。
func makeLCG(seed uint32) func() float64 {
	s := seed
	return func() float64 {
		s = (s + 0x6D2B79F5)
		t := s
		t = (t ^ (t >> 15)) * (t | 1)
		t ^= t + (t^(t>>7))*(t|61)
		return float64((t^(t>>14))) / 4294967296
	}
}

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var sc script
	if err := json.Unmarshal(data, &sc); err != nil {
		panic(err)
	}
	g, err := game.New(sc.FactionA, sc.FactionB, &game.Events{}, sc.Options)
	if err != nil {
		panic(err)
	}
	g.RNG = makeLCG(20261014)

	cmdIdx := 0
	var snaps []snap
	take := func(tick int) {
		s := snap{
			Tick: tick, Time: round3(g.Time), Running: g.Running, Winner: g.Winner,
			Credits:   []float64{round3(g.Players[0].Credits), round3(g.Players[1].Credits)},
			Units:     []int{len(g.OwnedUnits(0, "")), len(g.OwnedUnits(1, ""))},
			Buildings: []int{len(g.OwnedBuildings(0, "")), len(g.OwnedBuildings(1, ""))},
		}
		for _, u := range g.Units {
			if u.HP > 0 {
				s.UnitList = append(s.UnitList, unitView{u.ID, u.Type, u.Owner, round3(u.X), round3(u.Y), round3(u.HP)})
			}
		}
		s.Ore = []float64{} // 对齐 JS：空数组而非 null
		for _, o := range g.Ore {
			s.Ore = append(s.Ore, round3(o.Amount))
		}
		snaps = append(snaps, s)
	}
	for tick := 0; tick <= sc.Ticks; tick++ {
		for cmdIdx < len(sc.Commands) && sc.Commands[cmdIdx].AtTick == tick {
			c := sc.Commands[cmdIdx]
			switch c.Action {
			case "build":
				g.StartBuild(0, c.UnitType)
			case "place":
				g.PlaceBuilding(0, c.UnitType, c.X, c.Y)
			case "train":
				g.QueueUnit(0, c.UnitType)
			case "move":
				g.Selected = c.Selected
				g.Command(c.X, c.Y, c.AttackMove, 0)
			case "stop":
				g.Selected = c.Selected
				g.StopSelected(0)
			}
			cmdIdx++
		}
		if tick%20 == 0 {
			take(tick)
		}
		if !g.Running {
			break
		}
		g.Update(0.05)
	}
	take(sc.Ticks + 1)
	out, _ := json.MarshalIndent(snaps, "", " ")
	fmt.Println(string(out))
	fmt.Fprintf(os.Stderr, "OK ticks=%d snapshots=%d running=%v winner=%v\n", sc.Ticks, len(snaps), g.Running, g.Winner)
}
