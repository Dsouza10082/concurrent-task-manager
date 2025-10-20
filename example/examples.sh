#!/bin/bash
# ====================
# Practical Examples - Go 1.25 Async Server
# ====================

BASE_URL="http://localhost:8080"

echo "🚀 Go 1.25 Async Server - Practical Examples"
echo "==========================================="
echo ""

# ====================
# 1. BASIC EXAMPLE
# ====================
echo "📝 1. BASIC EXAMPLE - Create and monitor task"
echo "-------------------------------------------"

# Create a task
RESPONSE=$(curl -s -X POST $BASE_URL/async/process)
TASK_ID=$(echo $RESPONSE | jq -r '.task_id')

echo "✓ Task created: $TASK_ID"
echo "  Response: $RESPONSE" | jq

# Wait a bit
sleep 2

# Check status
echo ""
echo "📊 Checking status..."
curl -s $BASE_URL/async/status/$TASK_ID | jq

echo ""
echo "================================================"
echo ""

# ====================
# 2. BATCH PROCESSING
# ====================
echo "📦 2. BATCH PROCESSING"
echo "-------------------------------------------"

BATCH_RESPONSE=$(curl -s -X POST $BASE_URL/async/batch-process \
  -H "Content-Type: application/json" \
  -d '{
    "items": ["document1.pdf", "document2.pdf", "document3.pdf", "document4.pdf", "document5.pdf"],
    "priority": 7
  }')

echo "✓ Batch created:"
echo "$BATCH_RESPONSE" | jq

# Extract task IDs
TASK_IDS=$(echo $BATCH_RESPONSE | jq -r '.task_ids[]')

echo ""
echo "⏳ Waiting for processing (3 seconds)..."
sleep 3

echo ""
echo "📊 Status of batch tasks:"
for TASK_ID in $TASK_IDS; do
  STATUS=$(curl -s $BASE_URL/async/status/$TASK_ID | jq -r '.status')
  echo "  - $TASK_ID: $STATUS"
done

echo ""
echo "================================================"
echo ""

# ====================
# 3. PRIORITIES
# ====================
echo "⚡ 3. PRIORITY SYSTEM"
echo "-------------------------------------------"

echo "Creating tasks with different priorities..."
echo ""

# Low priority
LOW=$(curl -s -X POST $BASE_URL/async/heavy-work \
  -H "Content-Type: application/json" \
  -d '{"priority": 1}' | jq -r '.task_id')
echo "✓ Low priority (1):  $LOW"

# Normal
NORMAL=$(curl -s -X POST $BASE_URL/async/heavy-work \
  -H "Content-Type: application/json" \
  -d '{"priority": 5}' | jq -r '.task_id')
echo "✓ Normal (5):        $NORMAL"

# High
HIGH=$(curl -s -X POST $BASE_URL/async/heavy-work \
  -H "Content-Type: application/json" \
  -d '{"priority": 10}' | jq -r '.task_id')
echo "✓ High priority (10): $HIGH"

echo ""
echo "📊 Tasks in processing:"
curl -s $BASE_URL/async/tasks/status/processing | jq '.tasks[] | {task_id, priority, status}'

echo ""
echo "================================================"
echo ""

# ====================
# 4. EMAIL SENDING
# ====================
echo "📧 4. ASYNCHRONOUS EMAIL SENDING"
echo "-------------------------------------------"

EMAILS=("user1@example.com" "user2@example.com" "user3@example.com")

echo "Sending emails to ${#EMAILS[@]} recipients..."
echo ""

for EMAIL in "${EMAILS[@]}"; do
  EMAIL_TASK=$(curl -s -X POST $BASE_URL/async/send-email \
    -H "Content-Type: application/json" \
    -d "{
      \"to\": \"$EMAIL\",
      \"subject\": \"Weekly Newsletter\",
      \"body\": \"Check out this week's updates!\",
      \"priority\": 6
    }" | jq -r '.task_id')
  
  echo "✓ Email to $EMAIL queued: $EMAIL_TASK"
done

echo ""
echo "⏳ Waiting for sending (4 seconds)..."
sleep 4

echo ""
echo "📊 Status of sent emails:"
curl -s $BASE_URL/async/tasks/status/completed | \
  jq '.tasks[] | select(.task_id | startswith("email")) | {task_id, status, duration}'

echo ""
echo "================================================"
echo ""

# ====================
# 5. MONITORING
# ====================
echo "📊 5. MONITORING AND STATISTICS"
echo "-------------------------------------------"

echo "General statistics:"
curl -s $BASE_URL/async/stats | jq

echo ""
echo "Recent tasks:"
curl -s $BASE_URL/async/tasks/recent | jq '.tasks[] | {task_id, status, priority, duration}'

echo ""
echo "Health check:"
curl -s $BASE_URL/health | jq

echo ""
echo "================================================"
echo ""

# ====================
# 6. TASK CANCELLATION
# ====================
echo "❌ 6. TASK CANCELLATION"
echo "-------------------------------------------"

# Create a long task
CANCEL_TASK=$(curl -s -X POST $BASE_URL/async/heavy-work \
  -H "Content-Type: application/json" \
  -d '{"priority": 5}' | jq -r '.task_id')

echo "✓ Task created: $CANCEL_TASK"
echo ""

# Wait a bit
sleep 1

# Cancel
echo "❌ Cancelling task..."
CANCEL_RESULT=$(curl -s -X DELETE $BASE_URL/async/cancel/$CANCEL_TASK)
echo "$CANCEL_RESULT" | jq

echo ""
echo "📊 Status after cancellation:"
curl -s $BASE_URL/async/status/$CANCEL_TASK | jq '{task_id, status, error}'

echo ""
echo "================================================"
echo ""

# ====================
# 7. LOAD TEST
# ====================
echo "🔥 7. LOAD TEST (100 requests)"
echo "-------------------------------------------"

echo "Sending 100 simultaneous requests..."
START_TIME=$(date +%s)

for i in {1..100}; do
  curl -s -X POST $BASE_URL/async/process > /dev/null &
done

# Wait for all to finish
wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "✓ 100 requests sent in ${DURATION}s"
echo ""

# Wait for processing
echo "⏳ Waiting for processing (8 seconds)..."
sleep 8

echo ""
echo "📊 Statistics after load test:"
curl -s $BASE_URL/async/stats | jq

echo ""
echo "📈 Processing rate:"
STATS=$(curl -s $BASE_URL/async/stats)
COMPLETED=$(echo $STATS | jq -r '.completed')
echo "  Completed: $COMPLETED/100"
echo "  Rate: $(($COMPLETED * 100 / 100))%"

echo ""
echo "================================================"
echo ""

# ====================
# 8. JSON v1 vs v2 COMPARISON
# ====================
echo "⚡ 8. JSON v1 vs v2 PERFORMANCE"
echo "-------------------------------------------"

echo "JSON v2 (Go 1.25) offers:"
echo "  • 2-10x faster"
echo "  • Zero heap allocations"
echo "  • Better error messages"
echo ""

echo "Example of optimized response:"
curl -s $BASE_URL/async/status/$TASK_ID | jq

echo ""
echo "💡 Compile with GOEXPERIMENT=jsonv2 for best performance!"

echo ""
echo "================================================"
echo ""

# ====================
# 9. ADVANCED FILTERS
# ====================
echo "🔍 9. FILTERS AND QUERIES"
echo "-------------------------------------------"

echo "Completed tasks:"
COMPLETED_COUNT=$(curl -s $BASE_URL/async/tasks/status/completed | jq '.count')
echo "  Total: $COMPLETED_COUNT"

echo ""
echo "Failed tasks:"
FAILED_COUNT=$(curl -s $BASE_URL/async/tasks/status/failed | jq '.count')
echo "  Total: $FAILED_COUNT"

echo ""
echo "Queued tasks:"
QUEUED_COUNT=$(curl -s $BASE_URL/async/tasks/status/queued | jq '.count')
echo "  Total: $QUEUED_COUNT"

echo ""
echo "Last 5 tasks:"
curl -s $BASE_URL/async/tasks/recent | jq '.tasks[:5] | .[] | {task_id, status, duration}'

echo ""
echo "================================================"
echo ""

# ====================
# 10. AUTOMATION SCRIPTS
# ====================
echo "🤖 10. AUTOMATION SCRIPTS"
echo "-------------------------------------------"

cat << 'EOF'
# Continuous monitor (run in another terminal)
watch -n 5 'curl -s http://localhost:8080/async/stats | jq'

# Create tasks every second
while true; do
  curl -s -X POST http://localhost:8080/async/process
  sleep 1
done

# Monitor Docker logs
docker logs -f async-server-go125

# Simple dashboard
watch -n 2 '
echo "=== DASHBOARD ==="
curl -s http://localhost:8080/async/stats | jq
echo ""
echo "Last tasks:"
curl -s http://localhost:8080/async/tasks/recent | jq ".tasks[:5]"
'

# Stress test
for i in {1..1000}; do
  curl -s -X POST http://localhost:8080/async/process > /dev/null &
  if [ $((i % 100)) -eq 0 ]; then
    wait
    echo "$i requests sent..."
  fi
done
wait
EOF

echo ""
echo "================================================"
echo ""

# ====================
# FINAL SUMMARY
# ====================
echo "✅ SUMMARY - Go 1.25 Features"
echo "-------------------------------------------"
echo ""
echo "✓ Container-aware GOMAXPROCS"
echo "  └─ Automatically detects container CPUs"
echo ""
echo "✓ sync.WaitGroup.Go()"
echo "  └─ Simplifies goroutines management"
echo ""
echo "✓ encoding/json/v2"
echo "  └─ 2-10x faster, zero allocations"
echo ""
echo "✓ testing/synctest"
echo "  └─ Deterministic testing with virtual time"
echo ""
echo "✓ ConcurrentOrderedMap"
echo "  └─ Thread-safe map with order preservation"
echo ""
echo "================================================"
echo ""
echo "📚 Next steps:"
echo "  1. Explore the API: curl http://localhost:8080/health"
echo "  2. Check the tests: make test"
echo "  3. Run benchmarks: make bench-json"
echo "  4. Monitor: make stats"
echo ""
echo "🎉 Ready for production with Go 1.25!"
echo ""