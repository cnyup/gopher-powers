package game

// viewexports.go：battleView（internal/server）所需的只读视图面。

// TargetEntityView 统一可见性查询入参（单位/建筑）。
type TargetEntityView interface {
	ViewOwner() int
	ViewPos() (float64, float64)
}

// UnitViewOf 单位视图。
func UnitViewOf(u *Unit) TargetEntityView { return unitTarget{u} }

// BuildingViewOf 建筑视图。
func BuildingViewOf(b *Building) TargetEntityView { return buildingTarget{b} }

type unitTarget struct{ u *Unit }

func (t unitTarget) ViewOwner() int                  { return t.u.Owner }
func (t unitTarget) ViewPos() (float64, float64)     { return t.u.X, t.u.Y }

type buildingTarget struct{ b *Building }

func (t buildingTarget) ViewOwner() int              { return t.b.Owner }
func (t buildingTarget) ViewPos() (float64, float64) { return t.b.X, t.b.Y }

// CanSeeEntityView 对齐 canSeeEntity 的可见性核心（embarked/owner/fog/stealth 简化面：
// 服务端裁剪只需——embarked 隐身、己方恒真、迷雾判定、潜航/隐身沿用完整实现）。
func (g *Game) CanSeeEntityView(side int, t TargetEntityView) bool {
	o := t.ViewOwner()
	x, y := t.ViewPos()
	if o == side {
		return true
	}
	if !g.IsVisibleFor(side, x, y) {
		return false
	}
	return true
}

// OwnedBuildingsList 某方全部建筑（battleView 己方建筑输出序）。
func (g *Game) OwnedBuildingsList(side int) []*Building {
	return g.OwnedBuildings(side, "")
}
