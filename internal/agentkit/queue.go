package agentkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/vhwcm/Morpho/internal/gemini"
	"github.com/vhwcm/Morpho/internal/logger"
)

type QueueRequest struct {
	AI             AIClient
	Spec           Spec
	Task           string
	History        []gemini.ChatMessage
	AttemptTimeout time.Duration
}

type queueResult struct {
	res gemini.ChatResult
	err error
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

func (q *QueueManager) Enqueue(ctx context.Context, req QueueRequest) (gemini.ChatResult, error) {
	if req.AI == nil {
		return gemini.ChatResult{}, errors.New("cliente de IA inválido")
	}
	
	if len(req.History) == 0 && strings.TrimSpace(req.Task) == "" {
		return gemini.ChatResult{}, errors.New("tarefa vazia")
	}

	logger.Debug("Nova tarefa enfileirada", map[string]interface{}{
		"agent": req.Spec.Name,
	})

	job := queueJob{
		req: req,
		res: make(chan queueResult, 1),
	}

	select {
	case <-ctx.Done():
		return gemini.ChatResult{}, ctx.Err()
	case q.jobs <- job:
	}

	select {
	case <-ctx.Done():
		logger.Error("Contexto cancelado enquanto aguardava resposta da fila", ctx.Err())
		return gemini.ChatResult{}, ctx.Err()
	case result := <-job.res:
		logger.Debug("Tarefa finalizada na fila", map[string]interface{}{
			"agent": req.Spec.Name,
			"success": result.err == nil,
		})
		return result.res, result.err
	}
}

func (q *QueueManager) worker() {
	defer logger.RecoverPanic()
	logger.Debug("Worker da fila de IA iniciado")
	lastRequest := time.Now().Add(-1 * time.Second)
	for job := range q.jobs {
		elapsed := time.Since(lastRequest)
		if elapsed < 500*time.Millisecond {
			logger.Debug("Cooldown da fila ativo", map[string]interface{}{"wait": (500*time.Millisecond - elapsed).String()})
			time.Sleep(500*time.Millisecond - elapsed)
		}

		logger.Info("Worker processando tarefa", map[string]interface{}{"agent": job.req.Spec.Name})
		res, err := q.execute(job.req)
		lastRequest = time.Now()
		job.res <- queueResult{res: res, err: err}
	}
}

func (q *QueueManager) execute(req QueueRequest) (gemini.ChatResult, error) {
	maxRetries, retryDelay := q.config()

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retentando tarefa", map[string]interface{}{
				"agent":   req.Spec.Name,
				"attempt": attempt,
				"delay":   (retryDelay * time.Duration(attempt)).String(),
			})
		}

		attemptCtx := context.Background()
		cancel := func() {}
		if req.AttemptTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(context.Background(), req.AttemptTimeout)
		}

		var res gemini.ChatResult
		var err error

		if len(req.History) > 0 {
			res, err = RunWithResult(attemptCtx, req.AI, req.Spec, req.History)
		} else {
			history := []gemini.ChatMessage{{Role: "user", Content: req.Task}}
			res, err = RunWithResult(attemptCtx, req.AI, req.Spec, history)
		}

		cancel()

		if err == nil {
			return res, nil
		}

		lastErr = err
		if !isRetryableQueueError(err) || attempt == maxRetries {
			if attempt == maxRetries {
				logger.Error("Limite de retries atingido", err, map[string]interface{}{"agent": req.Spec.Name})
			}
			break
		}

		time.Sleep(retryDelay * time.Duration(attempt+1))
	}

	return gemini.ChatResult{}, fmt.Errorf("falha ao executar agente '%s' após retries: %w", req.Spec.Name, lastErr)
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
	if strings.Contains(msg, "429") || strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") {
		return true
	}
	return strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "timeout")
}

var defaultQueue = NewQueueManager(128, 3, 2*time.Second)

func ConfigureDefaultQueue(maxRetries int, retryDelay time.Duration) {
	defaultQueue.Configure(maxRetries, retryDelay)
}

func RunQueued(ctx context.Context, req QueueRequest) (gemini.ChatResult, error) {
	return defaultQueue.Enqueue(ctx, req)
}
