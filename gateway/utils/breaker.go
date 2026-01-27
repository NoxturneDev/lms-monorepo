package utils

import (
	"time"

	"github.com/sony/gobreaker"
)

var (
	StudentBreaker *gobreaker.CircuitBreaker
	TeacherBreaker *gobreaker.CircuitBreaker
)

func InitBreakers() {
	var stSettings gobreaker.Settings
	stSettings.Name = "StudentService"
	stSettings.Timeout = 5 * time.Second
	stSettings.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	}
	StudentBreaker = gobreaker.NewCircuitBreaker(stSettings)

	var tcSettings gobreaker.Settings
	tcSettings.Name = "TeacherService"
	tcSettings.Timeout = 5 * time.Second
	tcSettings.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	}
	TeacherBreaker = gobreaker.NewCircuitBreaker(tcSettings)
}

func ExecuteWithBreaker(cb *gobreaker.CircuitBreaker, logic func() (interface{}, error)) (interface{}, error) {
	return cb.Execute(logic)
}
