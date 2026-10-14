package data

// BridgeBlocksDirectShot 对齐 bridge-fire.js:2-11：
// 桥面挡住正处于同一桥面投影内的上下层直射；不改变沿桥航行或岸边射界。
// source/target 为参与判定的实体（kind/type/x/y），units 为 UNITS 表。
type shotEntity struct {
	Kind string // "unit" / "building"
	Type string
	X, Y float64
}

func BridgeBlocksDirectShot(m *MapSpec, source, target shotEntity) bool {
	if m.Water == nil || source.Kind != "unit" || target.Kind != "unit" {
		return false
	}
	a := UNITS[source.Type].Tags
	b := UNITS[target.Type].Tags
	hasAir := func(tags []string) bool { return HasTag(source.Type, "") || false }
	_ = hasAir
	if slicesContains(a, "air") || slicesContains(b, "air") {
		return false
	}
	switch source.Type {
	case "rocket", "bomber", "loiterer", "carrier":
		return false
	}
	if slicesContains(a, "ship") == slicesContains(b, "ship") {
		return false
	}
	water := m.Water
	onDeck := func(e shotEntity) bool { return e.X >= water.X1 && e.X <= water.X2 }
	if !onDeck(source) || !onDeck(target) {
		return false
	}
	for _, bridge := range m.Bridges {
		if source.Y >= bridge.Y1 && source.Y <= bridge.Y2 && target.Y >= bridge.Y1 && target.Y <= bridge.Y2 {
			return true
		}
	}
	return false
}

func slicesContains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
