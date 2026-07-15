package jobs

import (
	"bulk-domain-checker/server/internal/whois"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Result struct {
	Domain    string       `json:"domain"`
	Status    whois.Status `json:"status"`
	CheckedAt time.Time    `json:"checkedAt"`
	Error     *string      `json:"error"`
}

type Progress struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	Percentage int `json:"percentage"`
}

type Summary struct {
	Available int `json:"available"`
	Taken     int `json:"taken"`
	Error     int `json:"error"`
}

type Snapshot struct {
	JobID    string   `json:"jobId"`
	Status   string   `json:"status"`
	Progress Progress `json:"progress"`
	Summary  Summary  `json:"summary"`
	Results  []Result `json:"results"`
}

type job struct {
	mu         sync.RWMutex
	id         string
	status     string
	domains    []string
	results    []Result
	completed  int
	summary    Summary
	finishedAt time.Time
}

type Manager struct {
	mu              sync.RWMutex
	jobs            map[string]*job
	client          *whois.Client
	concurrency     int
	retention       time.Duration
	cleanupInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

func New(client *whois.Client, concurrency int, retention, cleanupInterval time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{
		jobs:            make(map[string]*job),
		client:          client,
		concurrency:     concurrency,
		retention:       retention,
		cleanupInterval: cleanupInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		manager.cleanup()
	}()
	return manager
}

func (m *Manager) Close() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) Create(domains []string) Snapshot {
	copiedDomains := append([]string(nil), domains...)
	currentJob := &job{
		id:      uuid.NewString(),
		status:  "QUEUED",
		domains: copiedDomains,
		results: make([]Result, 0, len(copiedDomains)),
	}

	m.mu.Lock()
	m.jobs[currentJob.id] = currentJob
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.process(currentJob)
	}()

	return m.snapshot(currentJob)
}

func (m *Manager) Get(id string) (Snapshot, bool) {
	m.mu.RLock()
	currentJob, ok := m.jobs[id]
	m.mu.RUnlock()
	if !ok {
		return Snapshot{}, false
	}
	return m.snapshot(currentJob), true
}

func (m *Manager) process(currentJob *job) {
	currentJob.mu.Lock()
	currentJob.status = "PROCESSING"
	currentJob.mu.Unlock()

	queue := make(chan string)
	var workers sync.WaitGroup

	for workerID := 0; workerID < m.concurrency; workerID++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-m.ctx.Done():
					return
				case domain, ok := <-queue:
					if !ok {
						return
					}
					status, checkError := m.client.Check(m.ctx, domain)
					m.recordResult(currentJob, Result{
						Domain:    domain,
						Status:    status,
						CheckedAt: time.Now().UTC(),
						Error:     checkError,
					})
				}
			}
		}()
	}

sendDomains:
	for _, domain := range currentJob.domains {
		select {
		case <-m.ctx.Done():
			break sendDomains
		case queue <- domain:
		}
	}
	close(queue)
	workers.Wait()

	currentJob.mu.Lock()
	if currentJob.completed == len(currentJob.domains) {
		currentJob.status = "COMPLETED"
	} else {
		currentJob.status = "FAILED"
	}
	currentJob.finishedAt = time.Now()
	currentJob.mu.Unlock()
}

func (m *Manager) recordResult(currentJob *job, result Result) {
	currentJob.mu.Lock()
	defer currentJob.mu.Unlock()

	currentJob.results = append(currentJob.results, result)
	currentJob.completed++
	switch result.Status {
	case whois.Available:
		currentJob.summary.Available++
	case whois.Taken:
		currentJob.summary.Taken++
	default:
		currentJob.summary.Error++
	}
}

func (m *Manager) snapshot(currentJob *job) Snapshot {
	currentJob.mu.RLock()
	defer currentJob.mu.RUnlock()

	progress := Progress{Total: len(currentJob.domains), Completed: currentJob.completed}
	if progress.Total > 0 {
		progress.Percentage = progress.Completed * 100 / progress.Total
	}

	results := append([]Result(nil), currentJob.results...)
	return Snapshot{
		JobID:    currentJob.id,
		Status:   currentJob.status,
		Progress: progress,
		Summary:  currentJob.summary,
		Results:  results,
	}
}

func (m *Manager) cleanup() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-m.retention)
			m.mu.Lock()
			for id, currentJob := range m.jobs {
				currentJob.mu.RLock()
				terminal := currentJob.status == "COMPLETED" || currentJob.status == "FAILED"
				expired := terminal && !currentJob.finishedAt.IsZero() && currentJob.finishedAt.Before(cutoff)
				currentJob.mu.RUnlock()
				if expired {
					delete(m.jobs, id)
				}
			}
			m.mu.Unlock()
		}
	}
}
