package flyf4f

import "math"

// Convert difference between two geo coordinates into distances on X-axis and Y-axis (meters)
func deltaXY(lonFrom float64, latFrom float64, lonTo float64, latTo float64) (float64, float64) {
	dlon := lonTo - lonFrom
	dlat := latTo - latFrom
	dx := distance(lonFrom, latFrom, lonTo, latFrom)
	dy := distance(lonFrom, latFrom, lonFrom, latTo)

	return math.Copysign(dx, dlon), math.Copysign(dy, dlat)
}

const earthRadiusMetres float64 = 6371000

// https://play.golang.org/p/MZVh5bRWqN
func distance(lonFrom float64, latFrom float64, lonTo float64, latTo float64) (distance float64) {
	var deltaLat = (latTo - latFrom) * (math.Pi / 180)
	var deltaLon = (lonTo - lonFrom) * (math.Pi / 180)

	var a = math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latFrom*(math.Pi/180))*math.Cos(latTo*(math.Pi/180))*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMetres * c
}
