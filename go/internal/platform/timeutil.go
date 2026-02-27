package platform

import "time"

func NowUnixSeconds() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second)
}
