package integration

import (
	"context"
	"sync"
	"time"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

// Job 描述一个可周期执行的后台任务。
type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

// JobFunc 便于通过函数快速声明任务。
type JobFunc struct {
	name     string
	interval time.Duration
	fn       func(ctx context.Context) error
}

// NewJobFunc 构造一个基于函数的 Job。
func NewJobFunc(name string, interval time.Duration, fn func(ctx context.Context) error) JobFunc {
	return JobFunc{name: name, interval: interval, fn: fn}
}

// Name 返回任务名称。
func (j JobFunc) Name() string { return j.name }

// Interval 返回执行间隔。
func (j JobFunc) Interval() time.Duration { return j.interval }

// Run 调用任务函数。
func (j JobFunc) Run(ctx context.Context) error {
	if j.fn == nil {
		return nil
	}
	return j.fn(ctx)
}

// Scheduler 负责调度 Integration 背景任务。
type Scheduler struct {
	logger     *pxlog.Entry
	dispatcher EventDispatcher

	mu      sync.Mutex
	jobs    []Job
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler 构造 Scheduler。
func NewScheduler(logger *pxlog.Entry) *Scheduler {
	if logger == nil {
		logger = pxlog.WithComponent("integration.scheduler")
	}
	return &Scheduler{
		logger: logger,
	}
}

// SetDispatcher injects the unified event dispatcher for cron triggers.
func (s *Scheduler) SetDispatcher(dispatcher EventDispatcher) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatcher = dispatcher
}

// Register 添加新的后台任务（需在 Start 前调用）。
func (s *Scheduler) Register(job Job) {
	if job == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
			"module":     "integration",
			"biz_scene":  "scheduler_register",
			"biz_domain": "integration",
			"component":  "integration.scheduler",
			"job":        job.Name(),
		}), "attempted to register job after scheduler start; ignoring")
		return
	}
	s.jobs = append(s.jobs, job)
}

// Start 启动所有已注册任务。
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	if s.stopCh == nil {
		s.stopCh = make(chan struct{})
	}
	jobs := append([]Job(nil), s.jobs...)
	s.mu.Unlock()

	for _, job := range jobs {
		s.wg.Add(1)
		go s.runJob(ctx, job)
	}
}

// Stop 停止所有任务并等待退出。
func (s *Scheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.started = false
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":     "integration",
			"biz_scene":  "scheduler_stop",
			"biz_domain": "integration",
			"component":  "integration.scheduler",
			"error":      ctx.Err().Error(),
		}), "integration scheduler stop timed out")
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	defer s.wg.Done()
	interval := job.Interval()
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.execute(ctx, job) // immediate run on start

	for {
		select {
		case <-ticker.C:
			s.execute(ctx, job)
		case <-s.stopCh:
			pxlog.DebugCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":     "integration",
				"biz_scene":  "scheduler_run",
				"biz_domain": "integration",
				"component":  "integration.scheduler",
				"job":        job.Name(),
			}), "scheduler stop signal received")
			return
		case <-ctx.Done():
			pxlog.DebugCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":     "integration",
				"biz_scene":  "scheduler_run",
				"biz_domain": "integration",
				"component":  "integration.scheduler",
				"job":        job.Name(),
				"error":      ctx.Err().Error(),
			}), "scheduler context cancelled")
			return
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			pxlog.ErrorCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":     "integration",
				"biz_scene":  "scheduler_execute",
				"biz_domain": "integration",
				"component":  "integration.scheduler",
				"job":        job.Name(),
				"panic":      r,
			}), "integration job panicked")
		}
	}()

	runCtx := ctx
	if traceID, err := s.dispatchTrigger(ctx, job); err != nil {
		pxlog.ErrorCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":     "integration",
			"biz_scene":  "scheduler_trigger_dispatch",
			"biz_domain": "integration",
			"component":  "integration.scheduler",
			"job":        job.Name(),
			"topic":      SchedulerTriggeredTopic,
			"error":      err.Error(),
		}), "scheduler trigger dispatch failed")
		return
	} else if traceID != "" {
		runCtx = context.WithValue(ctx, "request_id", traceID)
	}

	start := time.Now()
	if err := job.Run(runCtx); err != nil {
		pxlog.ErrorCtx(pxlog.WithLogFields(runCtx, map[string]interface{}{
			"module":     "integration",
			"biz_scene":  "scheduler_execute",
			"biz_domain": "integration",
			"component":  "integration.scheduler",
			"job":        job.Name(),
			"error":      err.Error(),
		}), "integration job execution failed")
		return
	}
	pxlog.DebugCtx(pxlog.WithLogFields(runCtx, map[string]interface{}{
		"module":     "integration",
		"biz_scene":  "scheduler_execute",
		"biz_domain": "integration",
		"component":  "integration.scheduler",
		"job":        job.Name(),
		"elapsed":    time.Since(start),
	}), "integration job executed")
}

func (s *Scheduler) dispatchTrigger(ctx context.Context, job Job) (string, error) {
	if s == nil || s.dispatcher == nil {
		return "", nil
	}
	return s.dispatcher.DispatchCronTrigger(ctx, job.Name(), map[string]any{
		"status": "queued",
	})
}
