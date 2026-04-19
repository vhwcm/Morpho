package agents

import (
	"context"
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func RunMetricsAgent(ctx context.Context) (MetricsResult, error) {
	cpuP, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return MetricsResult{}, err
	}

	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return MetricsResult{}, err
	}

	result := MetricsResult{
		CPUPercent:   firstOrZero(cpuP),
		MemoryUsedMB: float64(v.Used) / 1024 / 1024,
		Goroutines:   runtime.NumGoroutine(),
	}

	result.Summary = fmt.Sprintf("Uso atual monitorado do host: CPU %.2f%% e memória %.2f MB.", result.CPUPercent, result.MemoryUsedMB)
	return result, nil
}

func firstOrZero(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return v[0]
}
