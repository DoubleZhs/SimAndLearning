package simulator

import (
	"math"

	"golang.org/x/exp/rand"
)

func TripDistanceLim() int {
	const MILETOKM float64 = 1.60934
	const CELLLENGTH float64 = 7.5
	dice := rand.Float64()
	var lim float64
	switch {
	case dice <= 0.51:
		lim = 3.85
	case dice <= 0.71:
		lim = 7.65
	case dice <= 0.81:
		lim = 11.59
	case dice <= 0.92:
		lim = 19.68
	case dice <= 0.95:
		lim = 30.00
	case dice <= 0.99:
		lim = 70.00
	default:
		lim = 100.00
	}
	return int(math.Round(lim * MILETOKM * 1000 / CELLLENGTH))
}

func TripDistanceRange() (int, int) {
	dice := rand.Float64()
	const MILETOKM float64 = 1.60934
	const CELLLENGTH float64 = 7.5
	var minDis, maxDis float64
	switch {
	case dice <= 0.51/0.95:
		minDis, maxDis = 1.00, 3.85
	case dice <= 0.71/0.95:
		minDis, maxDis = 3.85, 7.65
	case dice <= 0.81/0.95:
		minDis, maxDis = 7.65, 11.59
	case dice <= 0.92/0.95:
		minDis, maxDis = 11.59, 19.68
	default:
		minDis, maxDis = 19.68, 30.00
		// case dice <= 0.95:
		// 	minDis, maxDis = 19.68, 30.00
		// case dice <= 0.99:
		// 	minDis, maxDis = 30.00, 70.00
		// default:
		// 	minDis, maxDis = 70.00, 100.00
	}
	minLength, maxLength := int(math.Round(minDis*MILETOKM*1000/CELLLENGTH)), int(math.Round(maxDis*MILETOKM*1000/CELLLENGTH))
	return minLength, maxLength
}

// test
// func TripDistanceRange() (int, int) {
// 	return 200, 6000
// }
