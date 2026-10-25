package utils

import "math"

const (
	PI               = 3.141592653589793
	Factor       int = 100000
	RAD              = PI / 180.0
	EARTH_RADIUS     = 6378137
)

type Grid struct {
	XMin     float64
	YMin     float64
	XMax     float64
	YMax     float64
	XLen     int64
	YLen     int64
	Gap      float64
	BeginPos int64
}

func (g Grid) GetAxisIndex(v, begin, step float64, mx int) int64 {
	if Less(v, begin) {
		return 0
	}
	ind := int64((v - begin) / step)
	if ind > int64(mx) {
		ind = int64(mx)
	}
	return ind + 1
}

func (g Grid) Size() int64 {
	if g.XLen == 0 && g.YLen == 0 {
		return 0
	}
	return (g.XLen + 2) * (g.YLen + 2)
}

func (g *Grid) SetBeginPos(p int64) {
	g.BeginPos = p
}

func (g Grid) GetPos(x, y float64) int64 {
	xInd := g.GetAxisIndex(x, g.XMin, g.Gap, int(g.XLen))
	yInd := g.GetAxisIndex(y, g.YMin, g.Gap, int(g.YLen))
	pos := yInd*(g.XLen+2) + xInd
	return pos + g.BeginPos
}

func GetAngle(xBeginLng, xBeginLat, xEndLng, xEndLat, yBeginLng, yBeginLat, yEndLng, yEndLat float64) float64 {
	directionX1 := xEndLng - xBeginLng
	directionY1 := xEndLat - xBeginLat
	directionX2 := yEndLng - yBeginLng
	directionY2 := yEndLat - yBeginLat

	module1 := math.Sqrt(directionX1*directionX1 + directionY1*directionY1)
	module2 := math.Sqrt(directionX2*directionX2 + directionY2*directionY2)
	inner := directionX1*directionX2 + directionY1*directionY2
	mo := module1 * module2

	if mo < 1e-7 {
		return -1
	}

	cosineAngle := inner / mo
	radianAngle := math.Acos(cosineAngle)
	angle := (radianAngle / PI) * 180
	return angle
}

func GetDirectDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := lat1 - lat2
	dLng := lng1 - lng2
	d := math.Sqrt(dLat*dLat + dLng*dLng)
	return d * float64(Factor)
}

func GetSphereDistance(lat1, lng1, lat2, lng2 float64) float64 {
	radLat1 := lat1 * RAD
	radLat2 := lat2 * RAD
	a := radLat1 - radLat2
	b := (lng1 - lng2) * RAD
	s := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
	s = s * EARTH_RADIUS
	return s
}
