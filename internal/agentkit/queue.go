package agentkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/vhwcm/Gopher/internal/gemini"
)

type QueueRequest struct {
	AI             AIClient
	Spec           Spec
	Task           string
	AttemptTimeout time.Duration
}

type queueResult struct {
	output string
	err    error
}

type queueJob struct {
	req QueueRequest
	res chan queueResult
}

type QueueManager struct {
	jobs       chan queueJob
	maxRetries int
	retryDelay time.Duration

	mu sync.RWMutex
}

func NewQueueManager(bufferSize, maxRetries int, retryDelay time.Duration) *QueueManager {
	if bufferSize <= 0 {
		bufferSize = 64
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	if retryDelay <= 0 {
		retryDelay = 2 * time.Second
	}

	q := &QueueManager{
		jobs:       make(chan queueJob, bufferSize),
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}

	go q.worker()
	return q
}

func (q *QueueManager) Configure(maxRetries int, retryDelay time.Duration) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if retryDelay <= 0 {
		retryDelay = 2 * time.Second
	}

	q.mu.Lock()
	q.maxRetries = maxRetries
	q.retryDelay = retryDelay
	q.mu.Unlock()
}

func (q *QueueManager) config() (int, time.Duration) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.maxRetries, q.retryDelay
}

func (q *QueueManager) Enqueue(ctx context.Context, req QueueRequest) (string, error) {
	if req.AI == nil {
		return "", errors.New("cliente de IA inválido")
	}
	if strings.TrimSpace(req.Task) == "" {
		return "", errors.New("tarefa vazia")
	}

	job := queueJob{
		req: req,
		res: make(chan queueResult, 1),
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case q.jobs <- job:
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-job.res:
		return result.output, result.err
	}
}

func (q *QueueManager) worker() {
	for job := range q.jobs {
		output, err := q.execute(job.req)
		job.res <- queueResult{output: output, err: err}
	}
}

func (q *QueueManager) execute(req QueueRequest) (string, error) {
	maxRetries, retryDelay := q.config()

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx := context.Background()
		cancel := func() {}
		if req.AttemptTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(context.Background(), req.AttemptTimeout)
		}

		out, err := Run(attemptCtx, req.AI, req.Spec, req.Task)
		cancel()

		if err == nil {
			return out, nil
		}

		lastErr = err
		if !isRetryableQueueError(err) || attempt == maxRetries {
			break
		}

		time.Sleep(retryDelay * time.Duration(attempt+1))
	}

	return "", fmt.Errorf("falha ao executar agente '%s' após retries: %w", req.Spec.Name, lastErr)
}

func isRetryableQueueError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var apiErr *gemini.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || apiErr.StatusCode >= 500 {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "timeout")
}

var defaultQueue = NewQueueManager(128, 3, 2*time.Second)

func ConfigureDefaultQueue(maxRetries int, retryDelay time.Duration) {
	defaultQueue.Configure(maxRetries, retryDelay)
}

func RunQueued(ctx context.Context, req QueueRequest) (string, error) {
	return defaultQueue.Enqueue(ctx, req)
}
