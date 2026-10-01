package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Tnembull/go-core-backend/internal/model"
)

var startTime = time.Now()

type HealthService interface {
	Check() model.HealthStatus
}

type healthService struct {
	version     string
	environment string
}

func NewHealthService(version, environment string) HealthService {
	return &healthService{
		version:     version,
		environment: environment,
	}
}

func (s *healthService) Check() model.HealthStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(startTime).Round(time.Second).String()

	return model.HealthStatus{
		Status:      "healthy",
		Version:     s.version,
		Environment: s.environment,
		Uptime:      uptime,
		Timestamp:   time.Now().UTC(),
		Runtime: map[string]string{
			"go_version":    runtime.Version(),
			"num_cpu":       fmt.Sprintf("%d", runtime.NumCPU()),
			"num_goroutine": fmt.Sprintf("%d", runtime.NumGoroutine()),
			"alloc_memory":  fmt.Sprintf("%.2f MB", float64(memStats.Alloc)/1024/1024),
			"sys_memory":    fmt.Sprintf("%.2f MB", float64(memStats.Sys)/1024/1024),
		},
	}
}
