package concurrent_task_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Dsouza10082/ConcurrentOrderedMap"
	"github.com/go-chi/chi/v5"
)

// AsyncResponse represents the immediate response when creating an async task
type AsyncResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TaskResult represents the full result of a task
type TaskResult struct {
	TaskID    string      `json:"task_id"`
	Status    string      `json:"status"` // queued, processing, completed, failed, cancelled
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	StartedAt time.Time   `json:"started_at"`
	EndedAt   *time.Time  `json:"ended_at,omitempty"`
	Priority  int         `json:"priority"`
	Duration  string      `json:"duration,omitempty"`
}

// TaskJob represents a job to be executed
type TaskJob struct {
	TaskID   string
	Priority int
	WorkFunc func(context.Context) (interface{}, error)
	Timeout  time.Duration
}

// Manager manages asynchronous tasks using Go 1.25 features
type Manager struct {
	tasks   *concurrentmap.ConcurrentOrderedMap[string, *TaskResult]
	workers int
	queue   chan *TaskJob
	// Go 1.25: sync.WaitGroup with new Go() method
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new task manager optimized for Go 1.25
func NewManager(workers int) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	tm := &Manager{
		tasks:   concurrentmap.NewConcurrentOrderedMap[string, *TaskResult](),
		workers: workers,
		queue:   make(chan *TaskJob, 100),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Go 1.25: Use WaitGroup.Go() - convenient new method
	// Eliminates the need to manually call Add(1) and Done()
	for i := 0; i < workers; i++ {
		workerID := i
		tm.wg.Go(func() {
			tm.worker(workerID)
		})
	}

	// Start periodic cleanup
	tm.wg.Go(func() {
		tm.cleanupOldTasks()
	})

	return tm
}

// worker processes jobs from the queue
func (tm *Manager) worker(id int) {
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-tm.ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case job, ok := <-tm.queue:
			if !ok {
				return
			}
			tm.processJob(id, job)
		}
	}
}

// processJob processes an individual job
func (tm *Manager) processJob(workerID int, job *TaskJob) {
	log.Printf("Worker %d processing task %s (priority: %d)", workerID, job.TaskID, job.Priority)

	ctx, cancel := context.WithTimeout(tm.ctx, job.Timeout)
	defer cancel()

	startTime := time.Now()
	result, err := job.WorkFunc(ctx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	task, exists := tm.tasks.Get(job.TaskID)
	if !exists {
		return
	}

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Result = result
	}

	task.EndedAt = &endTime
	task.Duration = duration.String()
	tm.tasks.Set(job.TaskID, task)
}

// AddTask adds a new task to the manager
func (tm *Manager) AddTask(taskID string, priority int, timeout time.Duration, workFunc func(context.Context) (interface{}, error)) {
	task := &TaskResult{
		TaskID:    taskID,
		Status:    "queued",
		StartedAt: time.Now(),
		Priority:  priority,
	}

	tm.tasks.Set(taskID, task)

	job := &TaskJob{
		TaskID:   taskID,
		Priority: priority,
		WorkFunc: workFunc,
		Timeout:  timeout,
	}

	// Go 1.25: Use WaitGroup.Go() for goroutine
	tm.wg.Go(func() {
		select {
		case tm.queue <- job:
			if t, exists := tm.tasks.Get(taskID); exists {
				t.Status = "processing"
				tm.tasks.Set(taskID, t)
			}
		case <-tm.ctx.Done():
			if t, exists := tm.tasks.Get(taskID); exists {
				t.Status = "cancelled"
				endTime := time.Now()
				t.EndedAt = &endTime
				tm.tasks.Set(taskID, t)
			}
		}
	})
}

// GetTask retrieves a task by ID
func (tm *Manager) GetTask(taskID string) (*TaskResult, bool) {
	return tm.tasks.Get(taskID)
}

// GetAllTasks returns all tasks in order
func (tm *Manager) GetAllTasks() []*TaskResult {
	tasks := make([]*TaskResult, 0)
	taskList := tm.tasks.GetOrderedV2()
	for _, task := range taskList {
		tasks = append(tasks, task.Value)
	}
	return tasks
}

// GetTasksByStatus returns tasks filtered by status
func (tm *Manager) GetTasksByStatus(status string) []*TaskResult {
	tasks := make([]*TaskResult, 0)
	taskList := tm.tasks.GetOrderedV2()
	for _, task := range taskList {
		if task.Value.Status == status {
			tasks = append(tasks, task.Value)
		}
	}
	return tasks
}

// GetRecentTasks returns the N most recent tasks
func (tm *Manager) GetRecentTasks(limit int) []*TaskResult {
	allTasks := tm.GetAllTasks()
	if len(allTasks) <= limit {
		return allTasks
	}
	return allTasks[len(allTasks)-limit:]
}

// CancelTask cancels a task
func (tm *Manager) CancelTask(taskID string) bool {
	if task, exists := tm.tasks.Get(taskID); exists {
		if task.Status == "queued" || task.Status == "processing" {
			endTime := time.Now()
			task.Status = "cancelled"
			task.EndedAt = &endTime
			tm.tasks.Set(taskID, task)
			return true
		}
	}
	return false
}

// cleanupOldTasks removes old completed tasks
func (tm *Manager) cleanupOldTasks() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-1 * time.Hour)
			var toDelete []string
			taskList := tm.tasks.GetOrderedV2()
			for _, task := range taskList {
				if task.Value.EndedAt != nil && task.Value.EndedAt.Before(cutoff) {
					toDelete = append(toDelete, task.Key)
				}
			}
			for _, key := range toDelete {
				tm.tasks.Delete(key)
			}
			log.Printf("Cleanup: %d tasks removed. Total: %d", len(toDelete), len(taskList))
		}
	}
}

// Shutdown gracefully shuts down the manager
func (tm *Manager) Shutdown() {
	log.Println("Starting TaskManager shutdown...")
	tm.cancel()
	close(tm.queue)
	tm.wg.Wait()
	log.Println("TaskManager shutdown complete")
}

// GetStats returns statistics about the task manager
func (tm *Manager) GetStats() map[string]interface{} {
	allTasks := tm.GetAllTasks()
	return map[string]interface{}{
		"total":      len(allTasks),
		"queued":     len(tm.GetTasksByStatus("queued")),
		"processing": len(tm.GetTasksByStatus("processing")),
		"completed":  len(tm.GetTasksByStatus("completed")),
		"failed":     len(tm.GetTasksByStatus("failed")),
		"cancelled":  len(tm.GetTasksByStatus("cancelled")),
		"workers":    tm.workers,
	}
}

// RegisterRoutes registers all async task routes to a Chi router
func (tm *Manager) RegisterRoutes(r chi.Router) {
	r.Route("/async", func(r chi.Router) {
		// Task creation endpoints
		r.Post("/task", tm.createTaskHandler)
		r.Post("/batch", tm.batchTaskHandler)

		// Query endpoints
		r.Get("/status/{taskID}", tm.getTaskStatusHandler)
		r.Get("/tasks", tm.getAllTasksHandler)
		r.Get("/tasks/status/{status}", tm.getTasksByStatusHandler)
		r.Get("/tasks/recent", tm.getRecentTasksHandler)

		// Management endpoints
		r.Delete("/cancel/{taskID}", tm.cancelTaskHandler)
		r.Get("/stats", tm.getStatsHandler)
	})
}

// HTTP Handlers

func (tm *Manager) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkType string `json:"work_type"` // For identifying work function
		Priority int    `json:"priority"`
		Timeout  int    `json:"timeout"` // seconds
		Data     interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.Priority == 0 {
		req.Priority = 5
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	timeout := time.Duration(req.Timeout) * time.Second

	// Example work function - customize based on your needs
	tm.AddTask(taskID, req.Priority, timeout, func(ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(5 * time.Second):
			return map[string]interface{}{
				"data":       req.Data,
				"processed":  true,
				"timestamp":  time.Now(),
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	respondJSON(w, http.StatusAccepted, AsyncResponse{
		TaskID:  taskID,
		Status:  "queued",
		Message: "Task queued successfully",
	})
}

func (tm *Manager) batchTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items    []interface{} `json:"items"`
		Priority int           `json:"priority"`
		Timeout  int           `json:"timeout"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.Priority == 0 {
		req.Priority = 3
	}
	if req.Timeout == 0 {
		req.Timeout = 60
	}

	taskIDs := make([]string, len(req.Items))
	timeout := time.Duration(req.Timeout) * time.Second

	for i, item := range req.Items {
		taskID := fmt.Sprintf("batch-%d-%d", time.Now().UnixNano(), i)
		taskIDs[i] = taskID

		capturedItem := item
		tm.AddTask(taskID, req.Priority, timeout, func(ctx context.Context) (interface{}, error) {
			time.Sleep(2 * time.Second)
			return map[string]interface{}{
				"item":      capturedItem,
				"processed": true,
			}, nil
		})
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":  fmt.Sprintf("%d tasks queued", len(req.Items)),
		"task_ids": taskIDs,
	})
}

func (tm *Manager) getTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	task, exists := tm.GetTask(taskID)
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Task not found"})
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (tm *Manager) getAllTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := tm.GetAllTasks()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(tasks),
		"tasks": tasks,
	})
}

func (tm *Manager) getTasksByStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	tasks := tm.GetTasksByStatus(status)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": status,
		"count":  len(tasks),
		"tasks":  tasks,
	})
}

func (tm *Manager) getRecentTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := tm.GetRecentTasks(20)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(tasks),
		"tasks": tasks,
	})
}

func (tm *Manager) cancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if tm.CancelTask(taskID) {
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Task cancelled",
		})
	} else {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Unable to cancel task",
		})
	}
}

func (tm *Manager) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, tm.GetStats())
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (tm *Manager) CleanupNow() {
    cutoff := time.Now().Add(-1 * time.Hour)
    var toDelete []string

	taskList := tm.tasks.GetOrderedV2()
	for _, task := range taskList {
		if task.Value.EndedAt != nil && task.Value.EndedAt.Before(cutoff) {
			toDelete = append(toDelete, task.Key)
		}
	}
	for _, key := range toDelete {
		tm.tasks.Delete(key)
	}

	log.Printf("Cleanup: %d tasks removed. Total: %d", len(toDelete), len(taskList))
}