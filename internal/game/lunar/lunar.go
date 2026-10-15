// Package lunar 对齐源项目 src/lunar-robots.js 的静态数据部分。
// 动态方法（robotChargeSite/updateRobotEnergy 等）挂在 Game 上下文上，
// 将在 internal/game 主体（Wave 1f）中以方法移植；此处只承载 ROBOT_SPECS/ROBOT_ENERGY/判定函数。
package lunar

import "math"

// Specs 对齐 lunar-robots.js:3-7 ROBOT_SPECS。
type Spec struct {
	Name        string
	Model       string
	ShotCost    float64
	Description string
}

var RobotSpecs = map[string]Spec{
	"rifle":    {Name: "月卫战斗机器人", Model: "robot_rifle", ShotCost: 4, Description: "原创月表无人战斗机体；电池驱动机动与脉冲武器，无需氧气、食物或常规弹药。低电量返场充电。"},
	"engineer": {Name: "天工工程机器人", Model: "robot_engineer", ShotCost: 0, Description: "原创月表工程机体；电池驱动，接管资源站与信标后驻留设施。无需氧气与食物。"},
	"scout":    {Name: "巡星侦察机器人", Model: "robot_scout", ShotCost: 0, Description: "原创月表侦察机体；无武装，保留侦察、反隐与一次性潜入职责。45 秒定时破坏可被工程师拆除。电池驱动，低电量自动返场。"},
}

// Energy 对齐 lunar-robots.js:8 ROBOT_ENERGY。
type Energy struct {
	Capacity, ReturnAt, IdleDrain, MoveDrain, ChargeRate, ChargeLoad float64
}

var RobotEnergy = Energy{Capacity: 100, ReturnAt: 25, IdleDrain: .08, MoveDrain: .22, ChargeRate: 12, ChargeLoad: 4}

// IsRobot 对齐 lunar-robots.js:9 isLunarRobot：未来图 + 三型步兵映射机器人。
// mapFuture 对应 map === true || map?.future。
func IsRobot(mapFuture bool, typ string) bool {
	if !mapFuture {
		return false
	}
	_, ok := RobotSpecs[typ]
	return ok
}

// BuildingName 对齐 lunar-robots.js:11-12 lunarBuildingProfile：
// 仅 barracks 在未来图改名，其余原样。
type BuildingData struct {
	Name string
	Desc string
	// 其余字段（cost/hp/power/size/time/icon）由调用方持有，此处不复制。
}

func BuildingProfile(future bool, typ string, base BuildingData) BuildingData {
	if future && typ == "barracks" {
		return BuildingData{Name: "机器人装配站", Desc: "装配月卫、天工与巡星机器人；提供充电和机体维修，充电依赖电网。"}
	}
	return base
}

// Distance 供 Game 侧复用的 hypot 语义。
func Distance(ax, ay, bx, by float64) float64 {
	dx, dy := ax-bx, ay-by
	return math.Sqrt(dx*dx + dy*dy)
}
