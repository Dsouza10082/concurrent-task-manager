package test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"testing/synctest"

	concurrent_task_manager "github.com/Dsouza10082/concurrent-task-manager"
)

func TestAsyncTask(t *testing.T) {

    tm := concurrent_task_manager.NewManager(2)
    defer tm.Shutdown()
    
    taskID := "test-1"
    tm.AddTask(taskID, 5, 10*time.Second, func(ctx context.Context) (interface{}, error) {
        return "success", nil
    })
    
    time.Sleep(1000 * time.Millisecond)
    
    task, exists := tm.GetTask(taskID)
    if !exists {
        t.Fatal("Task not found")
    }
    
    if task.Status != "queued" && task.Status != "processing" {
        t.Errorf("Unexpected status: %s", task.Status)
    }
}


func TestConcurrentTaskManager(t *testing.T) {
	tm := concurrent_task_manager.NewManager(2)
	defer tm.Shutdown()

	taskID := "test-task-1"
	
	tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
		time.Sleep(3 * time.Second)
		return "completed", nil
	})

	time.Sleep(50 * time.Millisecond)

	task, exists := tm.GetTask(taskID)
	if !exists {
		t.Fatal("Task not found")
	}

	if task.Status != "queued" && task.Status != "processing" {
		t.Errorf("Expected queued or processing, got '%s'", task.Status)
	}

	time.Sleep(4 * time.Second)

	task, exists = tm.GetTask(taskID)
	if !exists {
		t.Fatal("Task not found")
	}

	if task.Status != "completed" {
		t.Errorf("Expected completed, got '%s'", task.Status)
	}
}

func TestTaskManagerWithSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(2)
		defer tm.Shutdown()

		start := time.Now()

		taskID := "test-task-1"
		tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
			time.Sleep(3 * time.Second)
			return "completed", nil
		})

		synctest.Wait()

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Test took too long: %v (should be instant)", elapsed)
		}

		task, exists := tm.GetTask(taskID)
		if !exists {
			t.Fatal("Task not found")
		}

		if task.Status != "completed" {
			t.Errorf("Expected 'completed', got '%s'", task.Status)
		}
	})
}


func TestTimeoutWithSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(1)
		defer tm.Shutdown()

		taskID := "timeout-task"
		tm.AddTask(taskID, 5, 2*time.Second, func(ctx context.Context) (interface{}, error) {
			select {
			case <-time.After(10 * time.Second):
				return "never reaches here", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})

		synctest.Wait()

		task, exists := tm.GetTask(taskID)
		if !exists {
			t.Fatal("Tarefa não encontrada")
		}

		if task.Status != "failed" {
			t.Errorf("Esperado status 'failed', obtido '%s'", task.Status)
		}

		if task.Error == "" {
			t.Error("Deveria ter mensagem de erro")
		}
	})
}

func TestConcurrentTasks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(3)
		defer tm.Shutdown()

		var wg sync.WaitGroup
		taskIDs := make([]string, 10)

		for i := 0; i < 10; i++ {
			taskIDs[i] = fmt.Sprintf("concurrent-task-%d", i)
			taskID := taskIDs[i]
			
			wg.Go(func() {
				tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
					time.Sleep(1 * time.Second)
					return i, nil
				})
			})
		}

		wg.Wait()
		synctest.Wait()

		completed := 0
		for _, taskID := range taskIDs {
			task, exists := tm.GetTask(taskID)
			if exists && task.Status == "completed" {
				completed++
			}
		}

		if completed != 10 {
			t.Errorf("Esperado 10 tarefas completas, obtido %d", completed)
		}
	})
}

func TestWaitGroupGo(t *testing.T) {
	var wg sync.WaitGroup
	counter := 0
	mu := sync.Mutex{}

	for i := 0; i < 5; i++ {
		wg.Go(func() {
			mu.Lock()
			counter++
			mu.Unlock()
		})
	}

	wg.Wait()

	if counter != 5 {
		t.Errorf("Esperado counter=5, obtido %d", counter)
	}
}

func TestTaskCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(1)
		defer tm.Shutdown()

		taskID := "cancel-task"
		tm.AddTask(taskID, 5, 10*time.Second, func(ctx context.Context) (interface{}, error) {
			time.Sleep(5 * time.Second)
			return "completed", nil
		})

		time.Sleep(1 * time.Second)
		if !tm.CancelTask(taskID) {
			t.Error("Falha ao cancelar tarefa")
		}

		synctest.Wait()

		task, exists := tm.GetTask(taskID)
		if !exists {
			t.Fatal("Tarefa não encontrada")
		}

		if task.Status != "cancelled" {
			t.Errorf("Esperado status 'cancelled', obtido '%s'", task.Status)
		}
	})
}

func BenchmarkTaskProcessing(b *testing.B) {
	tm := concurrent_task_manager.NewManager(10)
	defer tm.Shutdown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		taskID := fmt.Sprintf("bench-task-%d", i)
		tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
			return "done", nil
		})
	}

	time.Sleep(1 * time.Second)
}

func BenchmarkJSONSerialization(b *testing.B) {
	task := &concurrent_task_manager.TaskResult{
		TaskID:    "bench-123",
		Status:    "completed",
		StartedAt: time.Now(),
		Priority:  5,
		Result: map[string]interface{}{
			"data":  "test data",
			"count": 100,
			"nested": map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			json.Marshal(task)
		}
	})

	b.Run("Unmarshal", func(b *testing.B) {
		data, _ := json.Marshal(task)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var result concurrent_task_manager.TaskResult
			json.Unmarshal(data, &result)
		}
	})
}

func TestContextTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan bool)

		go func() {
			time.Sleep(5 * time.Second)
			done <- true
		}()

		select {
		case <-done:
			t.Error("Não deveria completar antes do timeout")
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("Erro esperado DeadlineExceeded, obtido %v", ctx.Err())
			}
		}
	})
}

func TestIntegrationWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(3)
		defer tm.Shutdown()

		var wg sync.WaitGroup
		taskIDs := []string{"task-1", "task-2", "task-3"}

		for _, id := range taskIDs {
			taskID := id
			wg.Go(func() {
				tm.AddTask(taskID, 5, 10*time.Second, func(ctx context.Context) (interface{}, error) {
					time.Sleep(2 * time.Second)
					return fmt.Sprintf("result-%s", taskID), nil
				})
			})
		}
		wg.Wait()

		synctest.Wait()

		stats := map[string]int{
			"completed": len(tm.GetTasksByStatus("completed")),
			"failed":    len(tm.GetTasksByStatus("failed")),
			"total":     len(tm.GetAllTasks()),
		}

		if stats["total"] != 3 {
			t.Errorf("Esperado 3 tarefas totais, obtido %d", stats["total"])
		}

		if stats["completed"] != 3 {
			t.Errorf("Esperado 3 tarefas completas, obtido %d", stats["completed"])
		}
	})
}

func TestGracefulShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(2)

		for i := 0; i < 5; i++ {
			taskID := fmt.Sprintf("shutdown-task-%d", i)
			tm.AddTask(taskID, 5, 10*time.Second, func(ctx context.Context) (interface{}, error) {
				time.Sleep(1 * time.Second)
				return "done", nil
			})
		}

		time.Sleep(500 * time.Millisecond)

		tm.Shutdown()

		newTaskID := "after-shutdown"
		tm.AddTask(newTaskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
			return "should not process", nil
		})

		synctest.Wait()

		task, exists := tm.GetTask(newTaskID)
		if exists && task.Status == "completed" {
			t.Error("Não deveria processar tarefas após shutdown")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tm := concurrent_task_manager.NewManager(1)
		defer tm.Shutdown()

		testCases := []struct {
			name          string
			workFunc      func(context.Context) (interface{}, error)
			expectedError bool
		}{
			{
				name: "success",
				workFunc: func(ctx context.Context) (interface{}, error) {
					return "success", nil
				},
				expectedError: false,
			},
			{
				name: "custom_error",
				workFunc: func(ctx context.Context) (interface{}, error) {
					return nil, fmt.Errorf("erro customizado")
				},
				expectedError: true,
			},
			{
				name: "timeout_error",
				workFunc: func(ctx context.Context) (interface{}, error) {
					time.Sleep(10 * time.Second)
					return "never", nil
				},
				expectedError: true,
			},
		}

		for i, tc := range testCases {
			taskID := fmt.Sprintf("error-test-%d", i)
			tm.AddTask(taskID, 5, 2*time.Second, tc.workFunc)
		}

		synctest.Wait()

		successCount := 0
		errorCount := 0

		for i := range testCases {
			taskID := fmt.Sprintf("error-test-%d", i)
			task, exists := tm.GetTask(taskID)
			if !exists {
				t.Errorf("Tarefa %s não encontrada", taskID)
				continue
			}

			if task.Status == "completed" {
				successCount++
			} else if task.Status == "failed" {
				errorCount++
			}
		}

		if successCount != 1 {
			t.Errorf("Esperado 1 sucesso, obtido %d", successCount)
		}

		if errorCount != 2 {
			t.Errorf("Esperado 2 erros, obtido %d", errorCount)
		}
	})
}

func TestPriorityOrdering(t *testing.T) {
	tm := concurrent_task_manager.NewManager(1)
	defer tm.Shutdown()

	processed := make([]int, 0)
	mu := sync.Mutex{}

	priorities := []int{1, 10, 5, 3, 8}
	for i, priority := range priorities {
		taskID := fmt.Sprintf("priority-task-%d", i)
		capturedPriority := priority
		
		tm.AddTask(taskID, priority, 5*time.Second, func(ctx context.Context) (interface{}, error) {
			mu.Lock()
			processed = append(processed, capturedPriority)
			mu.Unlock()
			return "done", nil
		})
	}

	time.Sleep(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(processed) < 3 {
		t.Errorf("Esperado pelo menos 3 tarefas processadas, obtido %d", len(processed))
	}

	t.Logf("Ordem de processamento: %v", processed)
}

func TestConcurrentOrderedMap(t *testing.T) {
	tm := concurrent_task_manager.NewManager(10)
	defer tm.Shutdown()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		idx := i
		wg.Go(func() {
			taskID := fmt.Sprintf("concurrent-%d", idx)
			tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return idx, nil
			})
		})
	}

	wg.Wait()
	time.Sleep(3 * time.Second)

	allTasks := tm.GetAllTasks()
	if len(allTasks) < 100 {
		t.Errorf("Esperado pelo menos 100 tarefas, obtido %d", len(allTasks))
	}

	seen := make(map[string]bool)
	for _, task := range allTasks {
		if seen[task.TaskID] {
			t.Errorf("TaskID duplicado encontrado: %s", task.TaskID)
		}
		seen[task.TaskID] = true
	}
}

func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de memória em modo short")
	}

	tm := concurrent_task_manager.NewManager(5)
	defer tm.Shutdown()

	for i := 0; i < 1000; i++ {
		taskID := fmt.Sprintf("leak-test-%d", i)
		tm.AddTask(taskID, 5, 1*time.Second, func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "done", nil
		})

		if i%100 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	time.Sleep(2 * time.Second)

	totalTasks := len(tm.GetAllTasks())
	t.Logf("Total de tarefas após processamento: %d", totalTasks)

	if totalTasks > 1100 {
		t.Errorf("Possível vazamento de memória: %d tarefas em memória", totalTasks)
	}
}