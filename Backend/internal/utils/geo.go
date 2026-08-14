package utils

import "math"

// Haversine distance in meters between two lat/lng points
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func toRad(deg float64) float64 { return deg * math.Pi / 180 }

// IsValidLatLng checks basic bounds
func IsValidLatLng(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// ShouldIgnorePoint filters noisy GPS points
func ShouldIgnorePoint(accuracy *float64, distanceFromPrev float64) bool {
	if accuracy != nil && *accuracy > 50 {
		return true
	}
	// GPS jump filter: ignore unrealistic jumps > 500m between consecutive points
	// (could be tuned; 500m in one interval is suspicious for walking/driving at normal speed)
	if distanceFromPrev > 5000 {
		return true
	}
	return false
}
