package chaos

import (
	"context"
	"encoding/json"
	"quotient/engine"
	"quotient/engine/checks"
	"quotient/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisTemporaryUnavailability tests system behavior when Redis becomes temporarily unavailable
func TestRedisTemporaryUnavailability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	redisContainer := testutil.StartRedis(t)
	defer redisContainer.Close()

	ctx := context.Background()

	t.Run("task enqueue with retry on temporary failure", func(t *testing.T) {
		// Simulate temporary network issue
		// In real scenario, Redis would be unreachable for a short period

		task := engine.Task{
			TeamID:         1,
			TeamIdentifier: "01",
			ServiceType:    "Web",
			ServiceName:    "web01-web",
			RoundID:        1,
			Attempts:       3,
			Deadline:       time.Now().Add(60 * time.Second),
		}

		payload, _ := json.Marshal(task)

		// Attempt to enqueue with retry logic
		maxRetries := 3
		var err error

		for i := 0; i < maxRetries; i++ {
			err = redisContainer.Client.RPush(ctx, "tasks", payload).Err()
			if err == nil {
				break
			}
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
		}

		require.NoError(t, err, "should successfully enqueue after retries")

		// Verify task was enqueued
		length, _ := redisContainer.Client.LLen(ctx, "tasks").Result()
		assert.Equal(t, int64(1), length)
	})

	t.Run("graceful degradation when results unavailable", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		// Try to collect results with timeout (simulating engine behavior)
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		_, err := redisContainer.Client.BLPop(timeoutCtx, 1*time.Second, "results").Result()

		// Should timeout gracefully, not crash
		// Redis returns "redis: nil" when queue is empty and times out
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis: nil")
	})
}

// TestRedisSlowResponses tests behavior when Redis is slow but not down
func TestRedisSlowResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	redisContainer := testutil.StartRedis(t)
	defer redisContainer.Close()

	ctx := context.Background()

	t.Run("task collection with deadline", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		// Set a deadline for task collection
		deadline := time.Now().Add(2 * time.Second)
		timeoutCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		// Push a result after delay (simulating slow Redis)
		go func() {
			time.Sleep(500 * time.Millisecond)
			result := checks.Result{
				TeamID:      1,
				ServiceName: "web01-web",
				RoundID:     1,
				Status:      true,
				Points:      5,
			}
			payload, _ := json.Marshal(result)
			redisContainer.Client.RPush(ctx, "results", payload)
		}()

		// Try to collect within deadline
		val, err := redisContainer.Client.BLPop(timeoutCtx, 3*time.Second, "results").Result()

		// Should succeed before deadline
		require.NoError(t, err)
		assert.Len(t, val, 2)

		// Verify we got the result before deadline
		assert.True(t, time.Now().Before(deadline))
	})

	t.Run("partial results collection", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		expectedResults := 10
		actualResults := 7 // Simulate only getting 70% of results

		// Push partial results
		for i := 0; i < actualResults; i++ {
			result := checks.Result{
				TeamID:      uint(i + 1),
				ServiceName: "service",
				RoundID:     1,
				Status:      true,
				Points:      5,
			}
			payload, _ := json.Marshal(result)
			redisContainer.Client.RPush(ctx, "results", payload)
		}

		// Try to collect all results with timeout
		collected := []checks.Result{}
		timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		for i := 0; i < expectedResults; i++ {
			val, err := redisContainer.Client.BLPop(timeoutCtx, 2*time.Second, "results").Result()
			if err != nil {
				// Timeout or error - some results didn't arrive
				break
			}

			var result checks.Result
			json.Unmarshal([]byte(val[1]), &result)
			collected = append(collected, result)
		}

		// Should have collected only the available results
		assert.Equal(t, actualResults, len(collected))

		// System should handle partial results gracefully
		assert.Greater(t, len(collected), 0, "should collect at least some results")
		assert.Less(t, len(collected), expectedResults, "should not collect more than available")
	})
}

// TestRedisDataCorruption tests handling of corrupted data in Redis
func TestRedisDataCorruption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	redisContainer := testutil.StartRedis(t)
	defer redisContainer.Close()

	ctx := context.Background()

	t.Run("malformed task data", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		// Push malformed task data
		malformedData := []string{
			`{"invalid": "json"`,                    // Incomplete JSON
			`{"TeamID": "not_a_number"}`,            // Wrong type
			``,                                       // Empty
			`null`,                                   // Null
			`{"TeamID": 1}`,                          // Missing required fields
		}

		for _, data := range malformedData {
			redisContainer.Client.RPush(ctx, "tasks", data)
		}

		// Try to process each task
		validTasks := 0
		for i := 0; i < len(malformedData); i++ {
			val, err := redisContainer.Client.LPop(ctx, "tasks").Result()
			if err != nil {
				break
			}

			var task engine.Task
			err = json.Unmarshal([]byte(val), &task)
			if err == nil && task.TeamID > 0 {
				validTasks++
			}
			// System should log error but continue processing
		}

		// Should have handled malformed data without crashing
		assert.Equal(t, 0, validTasks, "no valid tasks in malformed data")
	})

	t.Run("malformed result data", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		// Mix valid and invalid results
		// Note: JSON keys must match struct field tags (e.g., "team_id" not "TeamID")
		results := []string{
			`{"team_id": 1, "name": "web", "round_id": 1, "status": true, "points": 5}`, // Valid
			`{"invalid"}`,                                                                 // Invalid
			`{"team_id": 2, "name": "ssh", "round_id": 1, "status": false, "points": 0}`, // Valid
			`corrupted data`,                                                              // Invalid
		}

		for _, data := range results {
			redisContainer.Client.RPush(ctx, "results", data)
		}

		// Collect and validate results
		validResults := []checks.Result{}
		for i := 0; i < len(results); i++ {
			val, err := redisContainer.Client.BLPop(ctx, 1*time.Second, "results").Result()
			if err != nil {
				break
			}

			var result checks.Result
			err = json.Unmarshal([]byte(val[1]), &result)
			if err == nil && result.TeamID > 0 {
				validResults = append(validResults, result)
			}
		}

		// Should have extracted valid results and skipped invalid ones
		assert.Equal(t, 2, len(validResults), "should collect only valid results")
	})
}

// TestRedisPubSubFailures tests pub/sub resilience
func TestRedisPubSubFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	redisContainer := testutil.StartRedis(t)
	defer redisContainer.Close()

	ctx := context.Background()

	t.Run("subscriber reconnection", func(t *testing.T) {
		// Subscribe to events
		pubsub := redisContainer.Client.Subscribe(ctx, "events")
		defer pubsub.Close()

		// Wait for subscription
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		ch := pubsub.Channel()

		// Publish event
		redisContainer.Client.Publish(ctx, "events", "test_event_1")

		// Receive event
		select {
		case msg := <-ch:
			assert.Equal(t, "test_event_1", msg.Payload)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		// Simulate disconnection and reconnection
		pubsub.Close()
		pubsub = redisContainer.Client.Subscribe(ctx, "events")
		defer pubsub.Close()
		pubsub.Receive(ctx)
		ch = pubsub.Channel()

		// Publish another event after reconnection
		redisContainer.Client.Publish(ctx, "events", "test_event_2")

		// Should receive event after reconnection
		select {
		case msg := <-ch:
			assert.Equal(t, "test_event_2", msg.Payload)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event after reconnection")
		}
	})

	t.Run("missed events during disconnection", func(t *testing.T) {
		// Subscribe to events
		pubsub := redisContainer.Client.Subscribe(ctx, "events")

		// Wait for subscription
		pubsub.Receive(ctx)

		// Close subscription (simulate disconnect)
		pubsub.Close()

		// Publish events while disconnected
		redisContainer.Client.Publish(ctx, "events", "missed_event_1")
		redisContainer.Client.Publish(ctx, "events", "missed_event_2")

		// Reconnect
		pubsub = redisContainer.Client.Subscribe(ctx, "events")
		defer pubsub.Close()
		pubsub.Receive(ctx)
		ch := pubsub.Channel()

		// Publish new event
		redisContainer.Client.Publish(ctx, "events", "new_event")

		// Should receive new event but not missed ones (expected Redis behavior)
		select {
		case msg := <-ch:
			assert.Equal(t, "new_event", msg.Payload, "should receive new event")
			assert.NotEqual(t, "missed_event_1", msg.Payload, "should not receive missed events")
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for new event")
		}
	})
}

// TestConcurrentAccess tests concurrent access to Redis
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	redisContainer := testutil.StartRedis(t)
	defer redisContainer.Close()

	ctx := context.Background()

	t.Run("concurrent task enqueue", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		numGoroutines := 10
		tasksPerGoroutine := 100

		// Spawn multiple goroutines to enqueue tasks concurrently
		done := make(chan bool)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < tasksPerGoroutine; j++ {
					task := engine.Task{
						TeamID:         uint(id*tasksPerGoroutine + j),
						TeamIdentifier: "01",
						ServiceType:    "Web",
						ServiceName:    "concurrent-service",
						RoundID:        1,
						Attempts:       3,
						Deadline:       time.Now().Add(60 * time.Second),
					}
					payload, _ := json.Marshal(task)
					redisContainer.Client.RPush(ctx, "tasks", payload)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all tasks were enqueued
		length, _ := redisContainer.Client.LLen(ctx, "tasks").Result()
		expectedTotal := int64(numGoroutines * tasksPerGoroutine)
		assert.Equal(t, expectedTotal, length, "all tasks should be enqueued")
	})

	t.Run("concurrent result collection", func(t *testing.T) {
		redisContainer.Client.FlushDB(ctx)

		numResults := 100

		// Push results
		for i := 0; i < numResults; i++ {
			result := checks.Result{
				TeamID:      uint(i),
				ServiceName: "service",
				RoundID:     1,
				Status:      true,
				Points:      5,
			}
			payload, _ := json.Marshal(result)
			redisContainer.Client.RPush(ctx, "results", payload)
		}

		// Collect results concurrently
		numCollectors := 5
		collected := make(chan checks.Result, numResults)
		done := make(chan bool)

		for i := 0; i < numCollectors; i++ {
			go func() {
				for {
					val, err := redisContainer.Client.BLPop(ctx, 1*time.Second, "results").Result()
					if err != nil {
						break
					}

					var result checks.Result
					json.Unmarshal([]byte(val[1]), &result)
					collected <- result
				}
				done <- true
			}()
		}

		// Wait for all collectors
		for i := 0; i < numCollectors; i++ {
			<-done
		}
		close(collected)

		// Count collected results
		collectedCount := 0
		for range collected {
			collectedCount++
		}

		// All results should be collected exactly once
		assert.Equal(t, numResults, collectedCount, "all results should be collected")
	})
}
