package agents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func RunDiagnostic(ctx context.Context, input DiagnosticInput) (DiagnosticReport, error) {
	start := time.Now()

	plan, err := RunPlanAgent(ctx, input.AI, input.Problem)
	if err != nil {
		return DiagnosticReport{}, fmt.Errorf("plan agent: %w", err)
	}

	var (
		logs    LogResult
		metrics MetricsResult
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		res, err := RunLogAgent(gctx, input.AI, input.LogFile)
		if err != nil {
			return fmt.Errorf("log agent: %w", err)
		}
		logs = res
		return nil
	})
	g.Go(func() error {
		res, err := RunMetricsAgent(gctx)
		if err != nil {
			return fmt.Errorf("metrics agent: %w", err)
		}
		metrics = res
		return nil
	})

	if err := g.Wait(); err != nil {
		return DiagnosticReport{}, err
	}

	solution, err := RunSolutionAgent(ctx, input.AI, input, plan, logs, metrics)
	if err != nil {
		return DiagnosticReport{}, fmt.Errorf("solution agent: %w", err)
	}

	if solution == "" {
		return DiagnosticReport{}, errors.New("solution agent retornou vazio")
	}

	return DiagnosticReport{
		Problem:  input.Problem,
		Plan:     plan,
		Logs:     logs,
		Metrics:  metrics,
		Solution: solution,
		Duration: time.Since(start),
	}, nil
}
