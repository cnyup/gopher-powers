package game

import (
	"math"

	gpdata "github.com/cnyup/gopher-powers/internal/game/data"
	"github.com/cnyup/gopher-powers/internal/game/pathfind"
	"github.com/cnyup/gopher-powers/internal/game/spacing"
)

// moveUnitNow 对齐 game.js L1000-1031 moveUnitNow（air/ship/地面寻路三分支）。
func (g *Game) moveUnitNow(u *Unit, goal Point, dt, stopDistance float64) {
	if Distance(u.X, u.Y, goal.X, goal.Y) <= stopDistance {
		return
	}
	if gpdata.HasTag(u.Type, "air") {
		g.moveWithAvoidance(u, goal, dt, stopDistance)
		return
	}
	if gpdata.HasTag(u.Type, "ship") {
		dest := g.navalGoal(goal.X, goal.Y, unitRadiusOf(u.Type))
		g.moveWithAvoidance(u, dest, dt, stopDistance)
		return
	}
	// 地面：pathTimer + 阻挡触发重寻路 + A* 寻路（对齐 L1003-1030）。
	u.PathTimer -= dt
	neighbors := g.nearbyUnits(u)
	ahead := goal
	if len(u.Path) > 0 {
		ahead = u.Path[0]
	}
	direction := math.Atan2(ahead.Y-u.Y, ahead.X-u.X)
	ur := unitRadiusOf(u.Type)
	speed := gpdata.UNITS[u.Type].Speed
	blocksAheadUnit := func(o *Unit, radius float64) bool {
		dx, dy := o.X-u.X, o.Y-u.Y
		forward := dx*math.Cos(direction) + dy*math.Sin(direction)
		across := math.Abs(dx*math.Sin(direction) - dy*math.Cos(direction))
		return forward > 0 && forward < ur+radius+speed*.65 && across < ur+radius+5
	}
	var blockers []*Unit
	for _, other := range neighbors {
		if unitLayerOf(other.Type) != unitLayerOf(u.Type) {
			continue
		}
		if Distance(other.X, other.Y, goal.X, goal.Y) > unitRadiusOf(other.Type)+stopDistance &&
			(other.Owner == u.Owner || g.CanSeeEntity(u.Owner, other)) && blocksAheadUnit(other, unitRadiusOf(other.Type)) {
			blockers = append(blockers, other)
		}
	}
	buildingAhead := false
	for _, b := range g.Buildings {
		if b.HP <= 0 || !g.CanSeeEntity(u.Owner, b) {
			continue
		}
		if Distance(b.X, b.Y, goal.X, goal.Y) > b.Size*.5+stopDistance && blocksAheadBuilding(b, b.Size*.45, direction, u.X, u.Y, ur, speed) {
			buildingAhead = true
			break
		}
	}
	if (len(blockers) > 0 || buildingAhead) && g.Time >= u.AvoidRepathAt {
		u.PathTimer = 0
		u.AvoidRepathAt = g.Time + .3
	}
	goalCell := itoa(int(math.Floor(goal.X/g.World.Cell))) + "," + itoa(int(math.Floor(goal.Y/g.World.Cell)))
	if len(u.Path) == 0 || u.PathTimer <= 0 || u.PathGoal != goalCell {
		u.Path = g.findPath(u, goal, stopDistance, blockers)
		u.PathTimer = 1.4 + g.Random(0, 0.3)
		u.PathGoal = goalCell
	}
	point := goal
	if len(u.Path) > 0 {
		point = u.Path[0]
	}
	if len(u.Path) > 0 && Distance(u.X, u.Y, point.X, point.Y) < 13 {
		u.Path = u.Path[1:]
		if len(u.Path) > 0 {
			point = u.Path[0]
		} else {
			point = goal
		}
	}
	sd := stopDistance
	if len(u.Path) > 0 {
		sd = 0
	}
	if !g.moveWithAvoidance(u, point, dt, sd) && g.Time >= u.BlockedRepathAt {
		u.PathTimer = 0
		u.Path = nil
		u.BlockedRepathAt = g.Time + .35
	}
}

func blocksAheadBuilding(b *Building, radius, direction, ux, uy, ur, speed float64) bool {
	dx, dy := b.X-ux, b.Y-uy
	forward := dx*math.Cos(direction) + dy*math.Sin(direction)
	across := math.Abs(dx*math.Sin(direction) - dy*math.Cos(direction))
	return forward > 0 && forward < ur+radius+speed*.65 && across < ur+radius+5
}

// nearbyUnits 对齐 (unitIndex||new SpatialIndex).nearby(u)。
func (g *Game) nearbyUnits(u *Unit) []*Unit {
	if g.UnitIndex != nil {
		return g.UnitIndex.Nearby(u)
	}
	ents := make([]spacing.Entity, 0, len(g.Units))
	for _, o := range g.Units {
		if o.HP > 0 && o.EmbarkedIn == 0 {
			ents = append(ents, unitView{o})
		}
	}
	idx := &unitIndexView{idx: spacing.NewSpatialIndex(ents)}
	return idx.Nearby(u)
}

// moveWithAvoidance 对齐 game.js L1043-1081 十一方向避让。
func (g *Game) moveWithAvoidance(u *Unit, point Point, dt, stopDistance float64) bool {
	dist := Distance(u.X, u.Y, point.X, point.Y)
	speed := gpdata.UNITS[u.Type].Speed
	amount := speed * dt
	if u.JammedUntil > g.Time {
		amount *= .6
	}
	if max := dist - stopDistance; amount > max {
		amount = max
	}
	if amount <= .001 {
		return true
	}
	angle := math.Atan2(point.Y-u.Y, point.X-u.X)
	radius := unitRadiusOf(u.Type)
	neighbors := g.nearbyUnits(u)
	layer := unitLayerOf(u.Type)
	reach := math.Max(amount, math.Min(dist, amount+speed*.4))
	type nearInfo struct {
		x, y                                     float64
		minimum, minimumSquared                  float64
		collisionSquared, previousSquared        float64
		known                                    bool
	}
	nearby := make([]nearInfo, 0, len(neighbors))
	for _, other := range neighbors {
		otherRadius := unitRadiusOf(other.Type)
		minimum := radius + otherRadius + 4
		separation := radius + otherRadius + 2
		squared := (u.X-other.X)*(u.X-other.X) + (u.Y-other.Y)*(u.Y-other.Y)
		if squared > (minimum+reach)*(minimum+reach) {
			continue
		}
		nearby = append(nearby, nearInfo{
			x: other.X, y: other.Y, minimum: minimum, minimumSquared: minimum * minimum,
			collisionSquared:  (separation - .01) * (separation - .01),
			previousSquared:   (math.Sqrt(squared) + .001) * (math.Sqrt(squared) + .001),
			known:             unitLayerOf(other.Type) == layer && (other.Owner == u.Owner || g.CanSeeEntity(u.Owner, other)),
		})
	}
	turns := [11]float64{0, .35, -.35, .7, -.7, 1.1, -1.1, 1.5, -1.5, 2.1, -2.1}
	var best *Point
	bestHeading, bestScore := 0.0, math.Inf(1)
	for _, turn := range turns {
		heading := angle + turn
		next := Point{u.X + math.Cos(heading)*amount, u.Y + math.Sin(heading)*amount}
		if !g.canOccupyUnit(u, next) {
			continue
		}
		if layer == "ground" {
			blocked := false
			for _, other := range nearby {
				sq := (next.X-other.x)*(next.X-other.x) + (next.Y-other.y)*(next.Y-other.y)
				if sq < other.collisionSquared && sq <= other.previousSquared {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
		}
		score := Distance(next.X, next.Y, point.X, point.Y) + math.Abs(turn)*amount*.08
		if turn < 0 {
			score += amount * .015
		}
		if layer == "ground" && u.AvoidUntil > g.Time && turn*float64(u.AvoidSide) < -.1 {
			score += amount * .85
		}
		look := Point{
			u.X + math.Cos(heading)*math.Min(dist, amount+speed*.4),
			u.Y + math.Sin(heading)*math.Min(dist, amount+speed*.4),
		}
		for _, other := range nearby {
			nextSq := (next.X-other.x)*(next.X-other.x) + (next.Y-other.y)*(next.Y-other.y)
			if nextSq < other.minimumSquared {
				w := 3.0
				if u.Type == "harvester" {
					w = 8
				}
				score += (other.minimum - math.Sqrt(nextSq)) * w
			}
			if other.known {
				lookSq := (look.X-other.x)*(look.X-other.x) + (look.Y-other.y)*(look.Y-other.y)
				if lookSq < other.minimumSquared {
					score += (other.minimum - math.Sqrt(lookSq)) * amount * .12
				}
			}
		}
		if score < bestScore {
			p := next
			best = &p
			bestHeading, bestScore = heading, score
		}
	}
	if best == nil {
		return false
	}
	chosenTurn := math.Atan2(math.Sin(bestHeading-angle), math.Cos(bestHeading-angle))
	if layer == "ground" && math.Abs(chosenTurn) > .3 && !(u.AvoidUntil > g.Time) {
		if chosenTurn > 0 {
			u.AvoidSide = 1
		} else {
			u.AvoidSide = -1
		}
		u.AvoidUntil = g.Time + 1.25
	}
	u.X, u.Y = best.X, best.Y
	u.Angle = bestHeading
	u.MovePulse += dt * 7
	u.LastMovedAt = g.Time
	u.DeployProgress = 0
	return true
}

// findPath 对齐 game.js L1084-1123。
func (g *Game) findPath(u *Unit, goal Point, stopDistance float64, obstacles []*Unit) []Point {
	cell := g.World.Cell
	cols := int(g.World.Width / cell)
	rows := int(g.World.Height / cell)
	grid := pathfind.NewGrid(cols, rows)
	terrainRadius := 14.0
	if gpdata.HasTag(u.Type, "infantry") {
		terrainRadius = 6
	}
	if len(g.Map.Barriers) > 0 {
		for cy := 0; cy < rows; cy++ {
			for cx := 0; cx < cols; cx++ {
				if g.isGroundBlocked(float64(cx)*cell+cell/2, float64(cy)*cell+cell/2, terrainRadius) {
					grid.SetWalkableAt(cx, cy, false)
				}
			}
		}
	}
	for _, b := range g.Buildings {
		if b.HP <= 0 || !g.CanSeeEntity(u.Owner, b) {
			continue
		}
		r := b.Size*0.49 + 11.0
		if gpdata.HasTag(u.Type, "infantry") {
			r = b.Size*0.49 + 4
		}
		x0 := int(math.Max(0, math.Floor((b.X-r)/cell)))
		x1 := int(math.Min(float64(cols-1), math.Floor((b.X+r)/cell)))
		y0 := int(math.Max(0, math.Floor((b.Y-r)/cell)))
		y1 := int(math.Min(float64(rows-1), math.Floor((b.Y+r)/cell)))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if Distance(float64(x)*cell+20, float64(y)*cell+20, b.X, b.Y) < r {
					grid.SetWalkableAt(x, y, false)
				}
			}
		}
	}
	// 只绕开附近已知的同层单位（对齐源 L1100：遍历 obstacles=blockers，非全图单位，
	// 避免通过寻路提前泄露迷雾内的敌军）。
	for _, other := range obstacles {
		if other.Owner != u.Owner && !g.CanSeeEntity(u.Owner, other) {
			continue
		}
		r := math.Max(unitRadiusOf(other.Type), unitRadiusOf(u.Type)+unitRadiusOf(other.Type)-cell*.45)
		if Distance(other.X, other.Y, goal.X, goal.Y) < r+stopDistance {
			continue
		}
		x0 := int(math.Max(0, math.Floor((other.X-r)/cell)))
		x1 := int(math.Min(float64(cols-1), math.Floor((other.X+r)/cell)))
		y0 := int(math.Max(0, math.Floor((other.Y-r)/cell)))
		y1 := int(math.Min(float64(rows-1), math.Floor((other.Y+r)/cell)))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if Distance(float64(x)*cell+20, float64(y)*cell+20, other.X, other.Y) < r {
					grid.SetWalkableAt(x, y, false)
				}
			}
		}
	}
	sx := clampI(int(math.Floor(u.X/cell)), 0, cols-1)
	sy := clampI(int(math.Floor(u.Y/cell)), 0, rows-1)
	ex := clampI(int(math.Floor(goal.X/cell)), 0, cols-1)
	ey := clampI(int(math.Floor(goal.Y/cell)), 0, rows-1)
	grid.SetWalkableAt(sx, sy, true)
	if !grid.IsWalkableAt(ex, ey) {
		best := math.Inf(1)
		found := false
		fx, fy := 0, 0
		for radius := 1; radius <= 5; radius++ {
			for y := ey - radius; y <= ey+radius && y >= 0 && y <= rows-1; y++ {
				if y < 0 || y > rows-1 {
					continue
				}
				for x := ex - radius; x <= ex+radius; x++ {
					if x < 0 || x > cols-1 {
						continue
					}
					if !grid.IsWalkableAt(x, y) {
						continue
					}
					score := math.Hypot(float64(x-ex), float64(y-ey)) + math.Hypot(float64(x-sx), float64(y-sy))*0.02
					if score < best {
						best = score
						found = true
						fx, fy = x, y
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			ex, ey = fx, fy
		}
	}
	if !grid.IsWalkableAt(ex, ey) {
		return nil
	}
	raw := pathfind.FindPath(sx, sy, ex, ey, grid)
	if len(raw) < 2 {
		return nil
	}
	compressed := pathfind.CompressPath(raw)
	out := make([]Point, 0, len(compressed))
	for i, node := range compressed {
		if i == 0 {
			continue
		}
		out = append(out, Point{float64(node.X)*cell + 20, float64(node.Y)*cell + 20})
	}
	return out
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
