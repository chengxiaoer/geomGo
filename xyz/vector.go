package xyz

import (
	"math"

	"github.com/chengxiaoer/geomGo"
)

// VectorDot 计算两向量的数量积
func VectorDot(v1Start, v1End, v2Start, v2End geom.Coord) float64 {
	v1Startv2Endx := v1End[0] - v1Start[0]
	v1Startv2Endy := v1End[1] - v1Start[1]
	v1Startv2Endz := v1End[2] - v1Start[2]
	v2Startv2Endx := v2End[0] - v2Start[0]
	v2Startv2Endy := v2End[1] - v2Start[1]
	v2Startv2Endz := v2End[2] - v2Start[2]
	return v1Startv2Endx*v2Startv2Endx + v1Startv2Endy*v2Startv2Endy + v1Startv2Endz*v2Startv2Endz
}

// VectorNormalize函数 创建一个坐标，它是一个归一化向量 从0,0,0 to vector
func VectorNormalize(vector geom.Coord) geom.Coord {
	vLen := VectorLength(vector)
	return geom.Coord{vector[0] / vLen, vector[1] / vLen, vector[2] / vLen}
}

// VectorLength函数 计算该向量的长度 从0，0，0起算
func VectorLength(vector geom.Coord) float64 {
	return math.Sqrt(vector[0]*vector[0] + vector[1]*vector[1] + vector[2]*vector[2])
}
