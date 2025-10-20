# Concurrent Task Manager Module - Quick Start Guide

## 🎯 What is Concurrent Task Manager?

Concurrent Task Manager is a **Go 1.25+ module** for building high-performance asynchronous task processing systems. It's designed to be **integrated into your existing Go applications** as a reusable component.

## ✨ Key Features

- ✅ **Go 1.25 Optimized** - Leverages latest Go features
- ✅ **Drop-in Integration** - Easy to add to existing projects
- ✅ **High Performance** - 2-10x faster with JSON v2
- ✅ **Thread-Safe** - ConcurrentOrderedMap for safe concurrent access
- ✅ **Production Ready** - Proper error handling and graceful shutdown
- ✅ **Container Aware** - Automatic CPU detection in Docker/Kubernetes

## 📦 Installation

```bash
# Install the module
go get github.com/Dsouza10082/concurrent-task-manager@v0.0.1

# Install dependencies
go get github.com/go-chi/chi/v5
go get github.com/Dsouza10082/ConcurrentOrderedMap
```

## 🚀 Quick Start (30 seconds)

### Step 1: Import and Initialize

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    concurrent_task_manager "github.com/Dsouza10082/concurrent-task-manager"
)

func main() {
    // Create manager with 5 workers
    tm := concurrent_task_manager.NewManager(5)
    defer tm.Shutdown()

    // Setup router
    r := chi.NewRouter()
    
    // Register routes (adds /async/* endpoints)
    tm.RegisterRoutes(r)
    
    // Start server
    http.ListenAndServe(":8080", r)
}
```

### Step 2: Test It

```bash
# Create a task
curl -X POST http://localhost:8080/async/task \
  -H "Content-Type: application/json" \
  -d '{"priority": 5, "timeout": 30}'

# Response: {"task_id":"task-123","status":"queued"}

# Check status
curl http://localhost:8080/async/status/task-123

# View statistics
curl http://localhost:8080/async/stats
```

## 🎨 Usage Patterns

### Pattern 1: Simple Integration

```go
// Just add routes to existing Chi router
func setupRouter() chi.Router {
    r := chi.NewRouter()
    
    tm := concurrent_task_manager.NewManager(5)
    tm.RegisterRoutes(r)
    
    // Your existing routes
    r.Get("/api/users", getUsers)
    r.Post("/api/users", createUser)
    
    return r
}
```

### Pattern 2: Custom Service

```go
type EmailService struct {
    tm *concurrent_task_manager.Manager
}

func NewEmailService() *EmailService {
    return &EmailService{
        tm: concurrent_task_manager.NewManager(10),
    }
}

func (s *EmailService) SendAsync(to, subject, body string) string {
    taskID := fmt.Sprintf("email-%d", time.Now().UnixNano())
    
    s.tm.AddTask(taskID, 7, 2*time.Minute, func(ctx context.Context) (interface{}, error) {
        // Your email sending logic
        return sendEmail(to, subject, body), nil
    })
    
    return taskID
}
```

### Pattern 3: Microservice

```go
func main() {
    // Image processing service
    imageProcessor := concurrent_task_manager.NewManager(4)
    defer imageProcessor.Shutdown()
    
    r := chi.NewRouter()
    imageProcessor.RegisterRoutes(r)
    
    r.Post("/process", func(w http.ResponseWriter, r *http.Request) {
        taskID := "img-" + uuid.New().String()
        
        imageProcessor.AddTask(taskID, 8, 10*time.Minute, func(ctx context.Context) (interface{}, error) {
            return processImage(ctx, imageData)
        })
        
        json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
    })
    
    http.ListenAndServe(":8080", r)
}
```

## 🔧 Configuration Options

### Worker Count

```go
// CPU-bound (video encoding, compression)
workers := runtime.GOMAXPROCS(0)

// I/O-bound (API calls, database queries)
workers := runtime.GOMAXPROCS(0) * 2

// Mixed workload
workers := 5  // or tune based on your needs
```

### Priority Levels

```go
const (
    PriorityLow      = 1   // Background tasks
    PriorityNormal   = 5   // Standard tasks
    PriorityHigh     = 8   // Important tasks
    PriorityCritical = 10  // Urgent tasks
)
```

### Timeout Strategy

```go
// Quick operations
timeout := 30 * time.Second

// Standard operations
timeout := 5 * time.Minute

// Long-running operations
timeout := 30 * time.Minute
```

## 📊 API Endpoints (Auto-registered)

When you call `tm.RegisterRoutes(r)`, these endpoints are added:

```
POST   /async/task                 - Create new task
POST   /async/batch                - Batch processing
GET    /async/status/{taskID}      - Get task status
GET    /async/tasks                - List all tasks
GET    /async/tasks/status/{status}- Filter by status
GET    /async/tasks/recent         - Recent tasks
DELETE /async/cancel/{taskID}      - Cancel task
GET    /async/stats                - Statistics
```

## 🏗️ Real-World Examples

### Email Queue

```go
type EmailQueue struct {
    tm *concurrent_task_manager.Manager
}

func (q *EmailQueue) QueueNewsletter(subscribers []string) []string {
    taskIDs := make([]string, len(subscribers))
    
    for i, email := range subscribers {
        taskID := fmt.Sprintf("newsletter-%d", i)
        taskIDs[i] = taskID
        
        q.tm.AddTask(taskID, 5, 5*time.Minute, func(ctx context.Context) (interface{}, error) {
            return sendNewsletter(ctx, email)
        })
    }
    
    return taskIDs
}
```

### Video Transcoding

```go
type VideoTranscoder struct {
    tm *concurrent_task_manager.Manager
}

func (v *VideoTranscoder) TranscodeVideo(videoURL string, formats []string) string {
    taskID := fmt.Sprintf("video-%d", time.Now().UnixNano())
    
    v.tm.AddTask(taskID, 8, 30*time.Minute, func(ctx context.Context) (interface{}, error) {
        results := make([]string, len(formats))
        
        for i, format := range formats {
            results[i] = transcode(ctx, videoURL, format)
        }
        
        return results, nil
    })
    
    return taskID
}
```

### Report Generation

```go
type ReportService struct {
    tm *concurrent_task_manager.Manager
}

func (r *ReportService) GenerateSalesReport(startDate, endDate time.Time) string {
    taskID := fmt.Sprintf("report-%d", time.Now().UnixNano())
    
    r.tm.AddTask(taskID, 6, 15*time.Minute, func(ctx context.Context) (interface{}, error) {
        data := fetchSalesData(ctx, startDate, endDate)
        report := generatePDF(ctx, data)
        url := uploadToS3(ctx, report)
        
        return map[string]string{"report_url": url}, nil
    })
    
    return taskID
}
```

## 🐳 Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN GOEXPERIMENT=jsonv2 go build -o server .

FROM alpine:latest
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### Build and Run

```bash
# Build
docker build -t myapp .

# Run with CPU limit (Go 1.25 auto-detects!)
docker run --cpus="4" -p 8080:8080 myapp
```

## ⚡ Performance Tips

### 1. Use JSON v2

```bash
# Compile with JSON v2 for 2-10x performance boost
GOEXPERIMENT=jsonv2 go build
```

### 2. Tune Workers

```go
// Monitor and adjust based on load
stats := tm.GetStats()
if stats["queued"].(int) > 100 {
    // Consider scaling up workers
}
```

### 3. Set Appropriate Timeouts

```go
// Don't use one-size-fits-all
quickTask := 30 * time.Second
normalTask := 5 * time.Minute
longTask := 30 * time.Minute
```

### 4. Use Priorities

```go
// Prioritize critical operations
tm.AddTask(id, 10, timeout, criticalWork)  // Processed first
tm.AddTask(id, 1, timeout, backgroundWork)  // Processed last
```

## 🧪 Testing

```go
func TestAsyncTask(t *testing.T) {
    tm := concurrent_task_manager.NewManager(2)
    defer tm.Shutdown()
    
    taskID := "test-1"
    tm.AddTask(taskID, 5, 5*time.Second, func(ctx context.Context) (interface{}, error) {
        return "success", nil
    })
    
    time.Sleep(100 * time.Millisecond)
    
    task, exists := tm.GetTask(taskID)
    if !exists {
        t.Fatal("Task not found")
    }
    
    if task.Status != "queued" && task.Status != "processing" {
        t.Errorf("Unexpected status: %s", task.Status)
    }
}
```

## 📚 Complete Examples

Check the `examples/` directory for full working examples:

- `examples/basic/` - Simple integration
- `examples/email-service/` - Email queue service
- `examples/image-processor/` - Image processing pipeline
- `examples/report-generator/` - Report generation service

## 🔍 Monitoring

```go
// Get current statistics
stats := tm.GetStats()
// Returns: {total, queued, processing, completed, failed, cancelled, workers}

// Get specific task
task, exists := tm.GetTask(taskID)

// Get recent tasks
recentTasks := tm.GetRecentTasks(10)

// Get by status
completedTasks := tm.GetTasksByStatus("completed")
```

## 🤝 Integration Checklist

- [ ] Install module: `go get github.com/Dsouza10082/concurrent-task-manager@v0.0.1`
- [ ] Create manager: `tm := concurrent_task_manager.NewManager(workers)`
- [ ] Register routes: `tm.RegisterRoutes(router)`
- [ ] Add custom task logic with `tm.AddTask()`
- [ ] Handle cleanup: `defer tm.Shutdown()`
- [ ] Compile with JSON v2: `GOEXPERIMENT=jsonv2 go build`
- [ ] Test in container: `docker run --cpus="4" myapp`

## 📖 Further Reading

- [Full README](README.md) - Complete documentation
- [API Documentation](docs/API.md) - Detailed API reference
- [Integration Guide](docs/INTEGRATION.md) - Integration patterns
- [Performance Guide](docs/PERFORMANCE.md) - Optimization tips

## 💡 Pro Tips

1. **Start with default settings** (5 workers) and tune based on metrics
2. **Use priorities** to ensure critical tasks are processed first
3. **Set appropriate timeouts** to prevent hung tasks
4. **Monitor stats regularly** to optimize worker count
5. **Test in containers** to verify Go 1.25 container-aware GOMAXPROCS
6. **Use JSON v2** in production for best performance

## ❓ Common Questions

**Q: How many workers should I use?**
A: Start with `runtime.GOMAXPROCS(0)` and tune based on whether your tasks are CPU or I/O bound.

**Q: Can I use this without Chi router?**
A: Yes! Just use `tm := concurrent_task_manager.NewManager(5)` and call `tm.AddTask()` directly.

**Q: How do I persist tasks?**
A: The module is stateless. For persistence, extend it with Redis or a database.

**Q: What happens on shutdown?**
A: `tm.Shutdown()` waits for running tasks to complete gracefully.

**Q: Can I use custom task IDs?**
A: Yes! Just pass your custom ID to `tm.AddTask(customID, ...)`

---

**Ready to integrate? Start with the basic example and customize for your needs!**

```bash
# Get started now
go get github.com/Dsouza10082/concurrent-task-manager@v0.0.1
```