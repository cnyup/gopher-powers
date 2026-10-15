package spacing

import "math"

// 对齐 V8：Math.hypot 两参快路径 = sqrt(dx²+dy²)（坐标域 [0,6000] 无溢出差异）。
func mathSqrt(v float64) float64 { return math.Sqrt(v) }
func mathAtan2(y, x float64) float64 {
	return math.Atan2(y, x)
}
func mathCos(v float64) float64 { return math.Cos(v) }
func mathSin(v float64) float64 { return math.Sin(v) }

var mathPi = math.Pi
