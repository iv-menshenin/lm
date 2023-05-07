package types

import "time"

type (
	// RPT represents Request Per Time
	RPT float64
)

func PerHour(rph int64) RPT {
	return RPT(rph)
}

func PerMinute(rpm int64) RPT {
	return RPT(rpm) * 60
}

func PerSecond(rps int64) RPT {
	return RPT(rps) * 3600
}

func PerTime(r int64, t time.Duration) RPT {
	return RPT(r) * RPT(time.Hour/t)
}

func (r RPT) PerHour() float64 {
	return float64(r)
}

func (r RPT) PerMinute() float64 {
	return float64(r) / 60
}

func (r RPT) PerSecond() float64 {
	return float64(r) / 3600
}

func (r RPT) PerTime(d time.Duration) float64 {
	return float64(r) * d.Hours()
}
