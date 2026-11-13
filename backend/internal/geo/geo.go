package geo

import (
	"math"
	"time"
)

// Haversine возвращает расстояние в километрах между двумя точками
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Радиус Земли в км
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1 = lat1 * math.Pi / 180.0
	lat2 = lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// TravelTimeMin возвращает примерное время в пути в минутах
// Берем расстояние по прямой, умножаем на 1.3 (кривизна дорог) и делим на среднюю скорость (30 км/ч)
func TravelTimeMin(lat1, lon1, lat2, lon2 float64) int {
	distKm := Haversine(lat1, lon1, lat2, lon2) * 1.3
	speedKmH := 30.0 // Средняя скорость по городу
	hours := distKm / speedKmH
	return int(hours * 60)
}

// ParseTime парсит время из строки "15:04" (часы:минуты) в time.Time для текущего дня
func ParseTime(baseDate time.Time, timeStr string) time.Time {
	t, _ := time.Parse("15:04", timeStr)
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), t.Hour(), t.Minute(), 0, 0, baseDate.Location())
}
